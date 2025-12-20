# Changelog

## Unreleased

* [Feature] Pass droplet size as an argument to override the value set in config file.

## 2.0.0 (2025-12-18)

* [Feature] **Progress Tracking**: Switched to using qBittorrent API for real-time progress monitoring (speed in MB/s, ETA, and percentage).
* [Feature] **Debug Mode**: Added `-debug` flag to enable detailed logging of SSH commands.
* [Feature] **Docker Support**: Switched to using `docker-20-04` base image and running qBittorrent in a Docker container.
* [Feature] **Cleanup Mode**: Added `-cleanRemote` flag to delete all droplets tagged with the configured `droplet_tag`.
* [Feature] **Dynamic Configuration**: Added `qbittorrent_version` and `qbittorrent_password` to `config.yml`.
* [Feature] **Tagging**: Droplets are now created with a configurable tag (`droplet_tag`) for easier identification.
* [Enhancement] **Timeout Logic**: Added a 1-minute timeout when waiting for torrents to appear or during API connection issues.
* [Enhancement] **Security**: Implemented PBKDF2 hash generation for qBittorrent password to avoid hardcoded credentials.
* [Enhancement] **SSH Optimization**: Combined multiple SSH commands during setup to improve performance.
* [Refactor] Updated logic to authenticate with qBittorrent WebUI before adding torrents and improved authentication flow.

## 1.1.0 (2019-04-07)

* [Enhancement] Pass download dir as an argument to override the value set in config file.

## 1.0.0 (2019-03-31)

* [Feature] Read configuration from a file.
* [Feature] Support multiple magnet links.
* [Feature] Accept Droplet IP to resume rsync.
