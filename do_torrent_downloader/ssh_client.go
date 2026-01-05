package doTorrentDownloader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type sshClient struct {
	hostname      string
	port          string
	config        *ssh.ClientConfig
	isDebugModeOn bool
}

type Torrent struct {
	Name       string  `json:"name"`
	Progress   float64 `json:"progress"`
	Dlspeed    int     `json:"dlspeed"`
	Eta        int     `json:"eta"`
	State      string  `json:"state"`
	Size       int64   `json:"size"`
	Downloaded int64   `json:"downloaded"`
}

type SshClientOp interface {
	executeCmd(string) string
	SetupQbittorrent(*config)
	StopQbittorrent()
	GetAuthSidForQbitAPI(password string) (string, error)
	GetTorrents(sid string) ([]Torrent, error)
	AddTorrents([]string, string)
}

func NewSshClient(hostname string, port string, username string, privateKeyPath string, isDebugModeOn bool) SshClientOp {
	client := &sshClient{
		hostname:      hostname,
		port:          port,
		isDebugModeOn: isDebugModeOn,
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
	if sshClient.isDebugModeOn {
		fmt.Printf("Will execute command: %s\n", command)
	}

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

func (sshClient sshClient) StopQbittorrent() {
	fmt.Println("Stopping and removing qbittorrent container to stop seeding...")
	sshClient.executeCmd("docker stop qbittorrent || true && docker rm qbittorrent || true")
	fmt.Println("qBittorrent container removed.")
}

func (sshClient sshClient) SetupQbittorrent(conf *config) {

	fmt.Printf("Creating directories: %s, %s\n", conf.Qbit.IncomingDir, conf.Qbit.CompletedDir)
	sshClient.executeCmd(fmt.Sprintf("mkdir -p %s %s /root/config/qBittorrent", conf.Qbit.IncomingDir, conf.Qbit.CompletedDir))

	fmt.Println("Configuring qBittorrent...")
	pwdHash, err := generateQbittorrentHash(conf.QbittorrentPassword)
	if err != nil {
		fmt.Printf("Error generating password hash: %v. Using default/hardcoded hash might fail login if password changed.\n", err)
		// Fallback or panic? For now let's just panic or warn.
		// If we can't generate hash, we can't set the correct password.
		panic(err)
	}

	configContent := fmt.Sprintf(`[LegalNotice]
Accepted=true

[BitTorrent]
Session\TempPath=/downloads/incoming
Session\DefaultSavePath=/downloads/completed
Session\TempPathEnabled=true
[Preferences]
# Disable OS Cache to prevent Linux write-buffer corruption (Libtorrent 2.0 bug)
Advanced\DiskIOReadMode=DisableOSCache
Advanced\DiskIOWriteMode=DisableOSCache
# This ensures that EVERY finished file is verified against the hash before finishing
Advanced\RecheckOnCompletion=true
WebUI\Username=admin
WebUI\Password_PBKDF2="%s"`, pwdHash)

	sshClient.executeCmd(fmt.Sprintf("cat <<'EOF' > /root/config/qBittorrent/qBittorrent.conf\n%s\nEOF", configContent))

	fmt.Printf("Pulling image: linuxserver/qbittorrent:%s\n", conf.QbittorrentVersion)
	sshClient.executeCmd(fmt.Sprintf("docker pull linuxserver/qbittorrent:%s", conf.QbittorrentVersion))

	fmt.Println("Stopping and removing existing qbittorrent container...")
	sshClient.executeCmd("docker stop qbittorrent || true && docker rm qbittorrent || true")

	fmt.Println("Starting qbittorrent container...")
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

func (sshClient sshClient) GetAuthSidForQbitAPI(password string) (string, error) {
	fmt.Println("Waiting for qBittorrent to initialize...")
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
		return "", fmt.Errorf("could not extract SID from login response")
	}

	fmt.Printf("Authenticated. Session ID obtained.\n")
	return sid, nil
}

func (sshClient sshClient) GetTorrents(sid string) ([]Torrent, error) {
	cmd := fmt.Sprintf("curl -s --cookie 'SID=%s' http://localhost:8080/api/v2/torrents/info", sid)
	output := sshClient.executeCmd(cmd)

	var torrents []Torrent
	err := json.Unmarshal([]byte(output), &torrents)
	if err != nil {
		return nil, err
	}
	return torrents, nil
}

func (sshClient sshClient) AddTorrents(magnetLinks []string, sid string) {
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
