package doTorrentDownloader

import (
	"fmt"
)

const Version = "1.1.0"

func PrintVersion() {
	fmt.Printf("v%s\n", Version)
}
