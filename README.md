# DigitalOcean torrent downloader

Program to download torrents using `qbittorrent` on a droplet on Digitalocean and Rsync the files via SSH to the local disk.

> Note: I built this program to download legal torrents to a machine that is behind a firewall preventing torrent traffic. This program can howwever download any torrent but the user needs to be careful and know whether they are legally allowed to download the torrents they are downloading.

### Prerequisites

* DigitalOcean's Personal Access Token for API access. You can manage them [here](https://cloud.digitalocean.com/settings/applications)
* Custom image on DigitalOcean with [`qbittorrent-nox`](https://github.com/qbittorrent/qBittorrent) installed.
* SSH access to the droplet usign a key file that is not password protected. (Will support password protected files soon in later versions.)
* `rsync` program installed on the host machine.

### Installation

* Download the latest release (1.1.0)
  ```
  bash -c "`curl -sL https://raw.githubusercontent.com/tsrivishnu/DO-torrent-downloader/v1.1.0/download.sh`"
  ```
* Make the `do-torrent-downloader.yml` from the example file and update the configuration to match yours.
  ```bash
  curl -L -o do-torrent-downloader.yml "https://raw.githubusercontent.com/tsrivishnu/DO-torrent-downloader/v1.1.0/do-torrent-downloader.example.yml" && \
  cp do-torrent-downloader.yml $HOME/do-torrent-downloader.yml
  ```
  > The script looks for the configuration file in the current working directory or the user's home directory. You can place the file in any of the locations.

### Build from source code.

* Clone this repository.
* Get the dependencies
  ```console
  $ cd do_torrent_downloader
  $ go get ./...
  ```
* Build
  ```console
  # From within the root of the project.
  $ go build main.go do_torrent_downloader
  ```

### Usage

#### To download using magnet links
```bash
$ ./do-torrent-downloader -m "<your-torrent-1-magnet-link" -m "<your-torrent-2-magnet-link"
```
 This above will start a new droplet from the image that is specified in the configuration file, starts the torrent client, waits till the downloads are completed, stops the torrent client and rsyncs the files to the local machine.

#### Resume a failed copy to local

If in case the program failed or the copy didn't finish. If your droplet is still running, you can resume the whole process by passing the droplet's public IP to the script.

```bash
$ ./do-torrent-downloader -ip xxx.xxx.xxx.xxx
```

#### Add a torrent to already running instance.

Not supported yet. Soon will add support for this as well.

# Ruby version (discontinued)

Looking for the discontinued ruby version of this project?
Its moved to the [ruby branch](https://github.com/tsrivishnu/DO-torrent-downloader/tree/ruby) in the git repository.
