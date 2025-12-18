package doTorrentDownloader

import (
	"fmt"
)

const Version = "2.0.0"

func PrintVersion() {
	fmt.Printf("v%s\n", Version)
}
