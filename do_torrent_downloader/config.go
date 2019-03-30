package doTorrentDownloader

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
)

type config struct {
	Size              string `yaml:"size"`
	ImageId           int    `yaml:"image_id"`
	DropletName       string `yaml:"droplet_name"`
	Region            string `yaml:"region"`
	SshKey            string `yaml:"ssh_key"`
	SshPrivateKeyPath string `yaml:"ssh_private_key_path"`
	DownloadDir       string `yaml:"download_dir"`
	DigitalOceanPat   string `yaml:"digital_ocean_pat"`
	Qbit              struct {
		IncomingDir  string `yaml:"incoming_dir"`
		CompletedDir string `yaml:"completed_dir"`
	}
}

func LoadConfiguration(file string) *config {
	return readFile(file)
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
