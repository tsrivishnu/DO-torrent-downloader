#!/usr/bin/env bash

# Set version to latest unless set by user
if [ -z "$VERSION" ]; then
  VERSION="1.0.0"
fi

echo "Downloading version ${VERSION}..."

# OS information (contains e.g. darwin x86_64)
UNAME=`uname -a | awk '{print tolower($0)}'`
if [[ ($UNAME == *"mac os x"*) || ($UNAME == *darwin*) ]]
then
  PLATFORM="darwin"
else
  PLATFORM="linux"
fi
if [[ ($UNAME == *x86_64*) || ($UNAME == *amd64*) ]]
then
  ARCH="amd64"
elif [[ ($UNAME == *armv7*)  ]]
then
  ARCH="armv7"
else
  echo "Currently, there are no 32bit binaries provided."
  echo "You will need to build binaries yourself."
  exit 1
fi

# Download binary
curl -L -o do_torrent_downloader "https://github.com/tsrivishnu/DO-torrent-downloader/releases/download/v${VERSION}/do_torrent_downloader_${PLATFORM}_${ARCH}"

# Make binary executable
chmod +x do_torrent_downloader

echo "Done."
