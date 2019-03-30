package doTorrentDownloader

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
)

type sshClient struct {
	hostname string
	port     string
	config   *ssh.ClientConfig
}

type SshClientOp interface {
	executeCmd(string) string
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
