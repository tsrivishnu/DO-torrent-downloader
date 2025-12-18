package doTorrentDownloader

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
)

type config struct {
	Size                string `yaml:"size"`
	ImageSlug           string `yaml:"image_slug"`
	DropletName         string `yaml:"droplet_name"`
	Region              string `yaml:"region"`
	SshKey              string `yaml:"ssh_key"`
	SshPrivateKeyPath   string `yaml:"ssh_private_key_path"`
	DownloadDir         string `yaml:"download_dir"`
	DigitalOceanPat     string `yaml:"digital_ocean_pat"`
	QbittorrentVersion  string `yaml:"qbittorrent_version"`
	QbittorrentPassword string `yaml:"qbittorrent_password"`
	DropletDownloadDir  string `yaml:"droplet_download_dir"`
	DropletTag          string `yaml:"droplet_tag"`
	Qbit                struct {
		IncomingDir  string `yaml:"incoming_dir"`
		CompletedDir string `yaml:"completed_dir"`
	}
}

func LoadConfiguration(filename string) *config {
	file := findConfigFile(filename)
	return readFile(file)
}

func findConfigFile(filename string) string {
	var dirs []string

	// Look in current working directory for the file.
	wDir, _ := os.Getwd()
	// Check user's home directory
	usr, _ := user.Current()

	dirs = append(dirs, wDir, usr.HomeDir)

	for _, dir := range dirs {
		filePath := filepath.Join(dir, filename)
		if _, err := os.Stat(filePath); err == nil {
			fmt.Printf("Using the configuration: %v\n", filePath)
			return filePath
		}
	}

	panic("Didn't find the config file.")
}

// readFile will read the config file
// and return the created config.
func readFile(filename string) *config {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(fmt.Sprintf("Error reading file %v", err))
	}

	ext := filepath.Ext(filename)
	return unmarshal(data, ext)
}

// unmarshal converts YAML into a config object.
func unmarshal(data []byte, ext string) *config {
	var config *config
	var err error
	if ext == ".json" {
		fmt.Println("Not supported yet!")
		// err = json.Unmarshal(data, &config)
	} else if ext == ".yml" || ext == ".yaml" {
		err = yaml.Unmarshal(data, &config)
	} else {
		panic("Unrecognized file extension")
	}
	if err != nil {
		panic(fmt.Sprintf("Error during Unmarshal: %v", err))
	}
	return config
}
