# digitalocean-torrent-downloader-client
Client ruby gem(may be not a gem yet!) for accessing digitalocean droplets to download torrents on them and rsync them to you local hardrive.

Run as follows

```console
./helper.rb -i "your-snapshot-name" -s 512mb -r sgp1 -k "your_ssh_key_name_in_your_digital_ocean_account" -m 'your_torrents_magnetic_link'
```

The implementation assumes qbit-torrent is installed on the digital ocean image that you build your droplet from.

Will add more details soon. Feel free to write to me if any questions.
