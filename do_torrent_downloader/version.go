package doTorrentDownloader

import (
	"fmt"
)

const Version = "0.0.0"

func PrintVersion() {
	fmt.Printf("v%s\n", Version)
}
