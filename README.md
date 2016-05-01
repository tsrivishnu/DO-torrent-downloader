# digitalocean-torrent-downloader
Ruby gem(may be not a gem yet!) to fire up digitalocean droplets dynamically and download torrents via them to you local hardrive.

<b>Note: Downloading copyrighted content is illegal! </b>

<b>Note</b>: The script assumes  that you have an image under your digitalocean account that has `qbittorrent` installed on it. Also, it assumes that the ssh key of the machine is added under your digitalocean account(It creates the droplet with it).

## Usage

* Install Ruby if you don't have it installed already
* Install `droplet_kit` gem with `gem install droplet_kit`
* Follow instructions [here](https://www.digitalocean.com/community/tutorials/how-to-use-the-digitalocean-api-v2) and get an access token. Assign it to the variable `access_token` in the script. 
* Add the path of the ssh key pair on your host machine to the array `keys` in the script. Example
   
     ````ruby
      keys = ['/Users/badAssCowBoy/.ssh/id_rsa']
    ````


## Run

```console
./helper.rb -i "your-snapshot-name" -s 512mb -r sgp1 -k "your_ssh_key_name_in_your_digital_ocean_account" -m 'your_torrents_magnetic_link'
```

## Options

-- Will add em soon. However, run `./helper.rb --help` to see the options.

Will add more details soon. Feel free to write to me if you have any questions.
