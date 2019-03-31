build: build-linux build-darwin build-windows

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o do_torrent_downloader_linux_amd64 main.go

build-darwin:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o do_torrent_downloader_darwin_amd64 main.go

build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o do_torrent_downloader_windows_amd64.exe main.go
