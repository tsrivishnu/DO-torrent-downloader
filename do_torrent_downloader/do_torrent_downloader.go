package doTorrentDownloader

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/digitalocean/godo"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var magnetLinks arrayFlags
var dropletIp string
var downloadDir string
var showVersion bool
var cleanRemote bool
var droplet *godo.Droplet

func setAndParseFlags() {
	flag.Var(&magnetLinks, "m", "Torrent magnet link.")
	flag.StringVar(&dropletIp, "ip", "", "Public IP of an already running droplet.")
	flag.StringVar(&downloadDir, "dir", "", "Download to directory (overrides what is set in the config file)")
	flag.BoolVar(&showVersion, "v", false, "prints current version")
	flag.BoolVar(&cleanRemote, "cleanRemote", false, "Delete all droplets with the configured tag")
	flag.Parse()
}

func optionsForQbit() string {
	var optionString strings.Builder
	for _, link := range magnetLinks {
		optionString.WriteString(fmt.Sprintf("\"%v\" ", link))
	}
	return optionString.String()
}

func RealMain() {
	setAndParseFlags()
	if showVersion {
		PrintVersion()
		return
	}

	config := LoadConfiguration("do-torrent-downloader.yml")
	if downloadDir != "" {
		// Override with argument
		config.DownloadDir = downloadDir
	}

	fmt.Println("\nRunning with the following config:")
	fmt.Println(config)
	fmt.Println("")

	InitDoClient(config.DigitalOceanPat)

	if cleanRemote {
		fmt.Printf("Cleaning up droplets with tag: %s\n", config.DropletTag)
		DeleteDropletsByTag(config.DropletTag)
		return
	}

	if dropletIp == "" { // No droplet ID passed.
		fmt.Println("Create a new droplet")
		droplet, _ = CreateDroplet(config)
		// Wait until the droplet is active
		for i := 0; i < 30; i++ {
			// refresh the droplet status
			droplet, _, _ = DoClient.Droplets.Get(context.TODO(), droplet.ID)

			fmt.Printf("Droplet status: %v \r\n", droplet.Status)
			if droplet.Status == "active" {
				fmt.Println("Droplet's now active")
				break
			} else {
				time.Sleep(5 * time.Second)
			}
		}
		time.Sleep(30 * time.Second)
	} else {
		droplet = GetByIp(dropletIp)
	}

	ip, _ := droplet.PublicIPv4()
	fmt.Printf("Droplet IPv4 %v \n", ip)

	sshClient := NewSshClient(ip, "22", "root", config.SshPrivateKeyPath)
	// delete firewall rules preventing SSH access
	sshClient.executeCmd("sudo ufw delete limit 22/tcp || true")
	sshClient.executeCmd("sudo ufw allow ssh || true && sudo ufw reload")

	sshClient.SetupQbittorrent(config)
	if len(magnetLinks) > 0 {
		sshClient.AddTorrents(magnetLinks, config.QbittorrentPassword)
		fmt.Printf("Torrents added. Monitor at: http://%v:8080\n", ip)
	} else {
		fmt.Println("No magnet links provided. Only starting the torrent client.")
	}

	downloadsInProgress := true
	for downloadsInProgress == true {
		// qBittorrent configured to move files from incoming to completed.
		// We check if incoming is empty (all moved) and completed has files.
		// Note: This logic assumes we started with empty dirs and added torrents.
		// If torrents are large, they stay in incoming.

		// Check incoming directory content
		incoming := sshClient.executeCmd(fmt.Sprintf("ls -A %s", config.Qbit.IncomingDir))
		// Check completed directory content
		downloaded := sshClient.executeCmd(fmt.Sprintf("ls -A %s", config.Qbit.CompletedDir))

		// Trimming whitespace is important as ls might output newlines
		incoming = strings.TrimSpace(incoming)
		downloaded = strings.TrimSpace(downloaded)

		if incoming == "" && downloaded != "" {
			fmt.Println("Downloads completed (Incoming empty, Completed has files)")
			downloadsInProgress = false
		} else {
			fmt.Println("Downloads in progress...")
			if incoming != "" {
				fmt.Printf("Incoming: %s\n", incoming)
			}
			time.Sleep(5 * time.Second)
		}
	}

	// Rsync the files: https://github.com/refola/golang/blob/master/backup/rsync.go
	fmt.Printf("Rsync files down to %v\n", config.DownloadDir)
	cmd := exec.Command(
		"rsync",
		"-e",
		fmt.Sprintf("ssh -o StrictHostKeyChecking=no -i %v", config.SshPrivateKeyPath),
		"-a",
		"--partial",
		"--progress",
		fmt.Sprintf("%v@%v:%v/", "root", ip, config.Qbit.CompletedDir),
		config.DownloadDir)
	// show rsync's output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}

	fmt.Println("Deleting the droplet...")
	DoClient.Droplets.Delete(context.TODO(), droplet.ID)
}
