package doTorrentDownloader

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type sshClient struct {
	hostname string
	port     string
	config   *ssh.ClientConfig
}

type SshClientOp interface {
	executeCmd(string) string
	SetupQbittorrent(*config)
	AddTorrents([]string, string)
}

func NewSshClient(hostname string, port string, username string, privateKeyPath string) SshClientOp {
	client := &sshClient{
		hostname: hostname,
		port:     port,
	}
	client.config = &ssh.ClientConfig{
		User:            username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			publicKeyFile(privateKeyPath),
		},
	}
	return client
}

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	// TODO: Support Passphrase protected keyfiles.
	// key, err := ssh.ParsePrivateKeyWithPassphrase(buffer, []byte(""))
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return ssh.PublicKeys(key)
}

func (sshClient sshClient) executeCmd(command string) string {
	fmt.Println("Will execute command: %s", command)
	time.Sleep(5 * time.Second)

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", sshClient.hostname, sshClient.port), sshClient.config)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"error opening connection to host %s port %s: %s", sshClient.hostname, sshClient.port, err,
		)
		return ""
	}

	session, err := conn.NewSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running cmd on host %s port %s: %s", sshClient.hostname, sshClient.port, err)
		return ""
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Run(command)

	return stdoutBuf.String()
}

func (sshClient sshClient) SetupQbittorrent(conf *config) {

	// 1. Create the directories on host
	fmt.Printf("Creating directories: %s, %s\n", conf.Qbit.IncomingDir, conf.Qbit.CompletedDir)
	sshClient.executeCmd(fmt.Sprintf("mkdir -p %s", conf.Qbit.IncomingDir))
	sshClient.executeCmd(fmt.Sprintf("mkdir -p %s", conf.Qbit.CompletedDir))
	sshClient.executeCmd("mkdir -p /root/config/qBittorrent")

	// 2. Write qBittorrent configuration
	// We configure it to accept local connections without auth for easy curl access,
	// and set the Temp/Save paths to map to the host directories we just created.
	fmt.Println("Configuring qBittorrent...")

	sshClient.executeCmd("echo '[LegalNotice]' > /root/config/qBittorrent/qBittorrent.conf")
	sshClient.executeCmd("echo 'Accepted=true' >> /root/config/qBittorrent/qBittorrent.conf")
	sshClient.executeCmd("echo '' >> /root/config/qBittorrent/qBittorrent.conf")
	sshClient.executeCmd("echo '[BitTorrent]' >> /root/config/qBittorrent/qBittorrent.conf")
	sshClient.executeCmd(fmt.Sprintf("echo 'Session\\TempPath=/downloads/incoming' >> /root/config/qBittorrent/qBittorrent.conf", conf.Qbit.IncomingDir))
	sshClient.executeCmd(fmt.Sprintf("echo 'Session\\DefaultSavePath=/downloads/completed' >> /root/config/qBittorrent/qBittorrent.conf", conf.Qbit.CompletedDir))
	sshClient.executeCmd("echo 'Session\\TempPathEnabled=true' >> /root/config/qBittorrent/qBittorrent.conf")
	sshClient.executeCmd("echo '[Preferences]' >> /root/config/qBittorrent/qBittorrent.conf")
	sshClient.executeCmd("echo 'WebUI\\Username=admin' >> /root/config/qBittorrent/qBittorrent.conf")

	// Generate password hash from config
	pwdHash, err := generateQbittorrentHash(conf.QbittorrentPassword)
	if err != nil {
		fmt.Printf("Error generating password hash: %v. Using default/hardcoded hash might fail login if password changed.\n", err)
		// Fallback or panic? For now let's just panic or warn.
		// If we can't generate hash, we can't set the correct password.
		panic(err)
	}
	// Escape quotes for echo
	// The hash format is @ByteArray(...) which doesn't contain double quotes, but we wrap it in double quotes for the config line.
	sshClient.executeCmd(fmt.Sprintf("echo 'WebUI\\Password_PBKDF2=\"%s\"' >> /root/config/qBittorrent/qBittorrent.conf", pwdHash))

	// 3. Pull the image
	fmt.Printf("Pulling image: linuxserver/qbittorrent:%s\n", conf.QbittorrentVersion)
	sshClient.executeCmd(fmt.Sprintf("docker pull linuxserver/qbittorrent:%s", conf.QbittorrentVersion))

	// 4. Stop and remove existing container
	fmt.Println("Stopping and removing existing qbittorrent container...")
	sshClient.executeCmd("docker stop qbittorrent || true")
	sshClient.executeCmd("docker rm qbittorrent || true")

	// 5. Run the container
	fmt.Println("Starting qbittorrent container...")
	// Map host incoming/completed to container /downloads/incoming|completed
	cmd := fmt.Sprintf(`docker run -d \
		--name=qbittorrent \
		-e PUID=0 \
		-e PGID=0 \
		-e TZ=Etc/UTC \
		-e WEBUI_PORT=8080 \
		-p 8080:8080 \
		-p 6881:6881 \
		-p 6881:6881/udp \
		-v %s:/downloads/incoming \
		-v %s:/downloads/completed \
		-v /root/config:/config \
		--restart unless-stopped \
		linuxserver/qbittorrent:%s`,
		conf.Qbit.IncomingDir,
		conf.Qbit.CompletedDir,
		conf.QbittorrentVersion)
	out := sshClient.executeCmd(cmd)
	fmt.Println("Container start output:", out)
}

func (sshClient sshClient) AddTorrents(magnetLinks []string, password string) {
	fmt.Println("Waiting for qBittorrent to initialize...")
	// Simple wait loop to ensure Web UI is up
	for i := 0; i < 12; i++ {
		// Try to fetch the login page (or just check port)
		check := sshClient.executeCmd("curl -s -I http://localhost:8080")
		if check != "" {
			break
		}
		time.Sleep(5 * time.Second)
	}

	fmt.Println("Authenticating...")
	// Authenticate and capture cookies
	// qBittorrent v4.x login: POST /api/v2/auth/login with username/password
	// Default username is admin
	loginCmd := fmt.Sprintf("curl -i -X POST -d 'username=admin&password=%s' http://localhost:8080/api/v2/auth/login", password)
	loginOut := sshClient.executeCmd(loginCmd)

	fmt.Println("Response: %s", loginOut)

	// Parse SID from headers
	var sid string
	// Example Header: Set-Cookie: SID=e6c4...; HttpOnly; path=/
	lines := strings.Split(loginOut, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "set-cookie:") && strings.Contains(line, "SID=") {
			parts := strings.Split(line, "SID=")
			if len(parts) > 1 {
				// The value might be terminated by ;
				val := strings.Split(parts[1], ";")[0]
				sid = val
				break
			}
		}
	}

	if sid == "" {
		fmt.Println("Warning: Could not extract SID from login response. Adding torrents might fail if auth is enforced.")
		// We proceed anyway, but likely to fail if auth is required
	} else {
		fmt.Printf("Authenticated. Session ID obtained.\n")
	}

	fmt.Println("Adding torrents...")
	for _, link := range magnetLinks {
		// Use curl to add torrent
		// Endpoint: /api/v2/torrents/add
		// Form data: urls=...
		// Need to pass cookie
		safeLink := fmt.Sprintf("urls=%s", link)
		cmd := fmt.Sprintf("curl --cookie 'SID=%s' -X POST -F '%s' http://localhost:8080/api/v2/torrents/add", sid, safeLink)
		sshClient.executeCmd(cmd)
	}
	fmt.Println("Torrents added.")
}
