package doTorrentDownloader

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
var dropletSize string
var showVersion bool
var cleanRemote bool
var isDebugModeOn bool
var droplet *godo.Droplet

func setAndParseFlags() {
	flag.Var(&magnetLinks, "m", "Torrent magnet link.")
	flag.StringVar(&dropletIp, "ip", "", "Public IP of an already running droplet.")
	flag.StringVar(&downloadDir, "dir", "", "Download to directory (overrides what is set in the config file)")
	flag.StringVar(&dropletSize, "size", "", "Size slug of the droplet (overrides what is set in the config file)")
	flag.BoolVar(&showVersion, "v", false, "prints current version")
	flag.BoolVar(&cleanRemote, "cleanRemote", false, "Delete all droplets with the configured tag")
	flag.BoolVar(&isDebugModeOn, "debug", false, "enable debug mode")
	flag.Parse()
}

func getTerminalWidth() int {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 80 // default
	}
	parts := strings.Fields(string(out))
	if len(parts) >= 2 {
		width, err := strconv.Atoi(parts[1])
		if err == nil {
			return width
		}
	}
	return 80
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
	if dropletSize != "" {
		// Override with argument
		config.Size = dropletSize
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

	sshClient := NewSshClient(ip, "22", "root", config.SshPrivateKeyPath, isDebugModeOn)
	// delete firewall rules preventing SSH access
	sshClient.executeCmd("sudo ufw allow ssh || true && sudo ufw reload")
	sshClient.executeCmd("sudo ufw delete limit 22/tcp || true")

	sshClient.SetupQbittorrent(config)

	var sid string
	var err error

	sid, err = sshClient.GetAuthSidForQbitAPI(config.QbittorrentPassword)
	if err != nil {
		fmt.Printf("Error authenticating: %v\n", err)
		return
	}

	if len(magnetLinks) > 0 {
		sshClient.AddTorrents(magnetLinks, sid)
		fmt.Printf("Torrents added. Monitor at: http://%v:8080\n", ip)
	} else {
		fmt.Println("No magnet links provided. Only starting the torrent client.")
	}

	downloadsInProgress := true
	waitForTorrentsCounter := 0
	const maxWaitAttempts = 12 // 1 minute (12 * 5 seconds)
	lastLinesPrinted := 0

	for downloadsInProgress == true {
		torrents, err := sshClient.GetTorrents(sid)
		if err != nil {
			lastLinesPrinted = 0
			fmt.Printf("Error getting torrents: %v\n", err)
			time.Sleep(5 * time.Second)
			waitForTorrentsCounter++
			if waitForTorrentsCounter >= maxWaitAttempts {
				fmt.Println("Timeout waiting for torrents/connection. Exiting loop.")
				break
			}
			continue
		}

		if len(torrents) == 0 {
			lastLinesPrinted = 0
			if len(magnetLinks) > 0 {
				fmt.Println("No torrents found yet...")
			} else {
				fmt.Println("No torrents in list. Waiting...")
			}
			time.Sleep(5 * time.Second)
			waitForTorrentsCounter++
			if waitForTorrentsCounter >= maxWaitAttempts {
				fmt.Println("Timeout waiting for torrents to appear. Exiting loop.")
				break
			}
			continue
		}

		// Reset counter if we found torrents
		waitForTorrentsCounter = 0

		allCompleted := true

		if lastLinesPrinted > 0 {
			fmt.Printf("\033[%dA", lastLinesPrinted)
		}
		fmt.Print("\033[2K\r--- Torrent Status ---\n")
		termWidth := getTerminalWidth()
		for _, t := range torrents {
			speedMB := float64(t.Dlspeed) / 1024 / 1024
			etaDuration := time.Duration(t.Eta) * time.Second
			etaString := fmt.Sprintf("%dm:%ds", int(etaDuration.Minutes()), int(etaDuration.Seconds())%60)
			if t.Eta == 8640000 { // qBittorrent returns 8640000 for infinity/unknown
				etaString = "∞"
			}

			// fmt.Printf("\033[2K\r[%s] %s - %.2f%% - Speed: %.2f MB/s - ETA: %s\n", t.State, t.Name, t.Progress*100, speedMB, etaString)
			// Construct parts to calculate length
			prefix := fmt.Sprintf("[%s] ", t.State)
			suffix := fmt.Sprintf(" - %.2f%% - Speed: %.2f MB/s - ETA: %s", t.Progress*100, speedMB, etaString)
			
			availableSpace := termWidth - len(prefix) - len(suffix)
			name := t.Name
			if availableSpace < 5 { // Minimal space fallback
				// Just print as is or very short, but let's assume at least some space. 
				// If strictly enforcing no wrap, we might hide name.
				if len(name) > 10 {
					name = name[:7] + "..."
				}
			} else if len(name) > availableSpace {
				name = name[:availableSpace-3] + "..."
			}

			fmt.Printf("\033[2K\r%s%s%s\n", prefix, name, suffix)


			// Check completion
			// States: downloading, stalledDL, metaDL, pausedDL, queuedDL, allocating, uploading, stalledUP, pausedUP, queuedUP, moving, missingFiles, error
			// We consider it done if it's seeding, uploading, pausedUP, or progress is 1.0
			isComplete := false
			if t.Progress >= 1.0 {
				isComplete = true
			}
			// qBittorrent states for completion usually involve "UP" (uploading) or "pausedUP" (completed and paused)
			if strings.Contains(t.State, "UP") || t.State == "uploading" || t.State == "stalledUP" {
				isComplete = true
			}

			if !isComplete {
				allCompleted = false
			}
		}
		fmt.Print("\033[2K\r----------------------\n")
		lastLinesPrinted = len(torrents) + 2

		if allCompleted && len(torrents) > 0 {
			fmt.Println("All downloads completed.")
			downloadsInProgress = false
		} else {
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
	err = cmd.Run()
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}

	fmt.Println("Deleting the droplet...")
	DoClient.Droplets.Delete(context.TODO(), droplet.ID)
}
