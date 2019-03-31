package doTorrentDownloader

import (
	"context"
	"flag"
	"fmt"
	"github.com/digitalocean/godo"
	"os"
	"os/exec"
	"strings"
	"time"
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
var showVersion bool
var droplet *godo.Droplet

func setAndParseFlags() {
	flag.Var(&magnetLinks, "m", "Torrent magnet link.")
	flag.StringVar(&dropletIp, "ip", "", "Public IP of an already running droplet.")
	flag.BoolVar(&showVersion, "v", false, "prints current version")
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

	fmt.Println("\nRunning with the following config:")
	fmt.Println(config)
	fmt.Println("")

	InitDoClient(config.DigitalOceanPat)

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
	if len(magnetLinks) > 0 {
		resp := sshClient.executeCmd(fmt.Sprintf("qbittorrent-nox -d %v", optionsForQbit()))
		fmt.Println(resp)
		fmt.Printf("Torrents will be downloaded. Follow at: http://admin:adminadmin@%v:8080\n", ip)
	} else {
		fmt.Println("No magnet links provided. Skip starting the torrent client.")
	}

	downloadsInProgress := true
	for downloadsInProgress == true {
		incoming := sshClient.executeCmd(fmt.Sprintf("ls %s", config.Qbit.IncomingDir))
		downloaded := sshClient.executeCmd(fmt.Sprintf("ls %s", config.Qbit.CompletedDir))

		if incoming == "" && downloaded != "" {
			fmt.Println("Downloads completed")
			// kill qbit-torrent
			sshClient.executeCmd("pkill qbit")
			downloadsInProgress = false
		} else {
			fmt.Println("Following downloads are in progress...")
			fmt.Println(incoming)
		}
		time.Sleep(5 * time.Second)
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
		fmt.Sprintf("%v@%v:%v", "root", ip, config.Qbit.CompletedDir),
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
