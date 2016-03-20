#!/usr/bin/env ruby
# Script that would fireup a digital ocean droplet with an image that is specified
# and runs the qbitclient on the droplet to download the magnet url provided in the
# arguments and rsync the downloaded files to local filesystem.

# This programs needs you to have an image in your digitalocean that has
# `qbittorrent` installed on it. Also, needs you to have the ssh key of the script's
# host machine to be added in the digitalocean account.
#
# Author: Sri Vishnu Totakura
# Date: 23 January, 2016



require 'droplet_kit'
require 'optparse'
require 'net/ssh'
require 'pry'
require 'open3'

# TODO: Get these values from a env
access_token    = "" # TODO: SET THIS VALUE
keys = [''] # TODO: SET THIS VALUE
USER = 'root'

options = { :size => "512MB" }

usage_opts = nil

ARGV.options do |opts|

  opts.on('-r', '--region region_name', 'Select Regions name') do |region|
    options[:region] = region
  end

  opts.on('-i', '--image image_name', 'Select image name') do |image|
    options[:image] = image
  end

  opts.on('-s', '--size [size]', 'Select size of the droplet') do |size|
    options[:size] = size
  end

  opts.on('-k', '--key ssh_key', 'SSH key name') do |ssh_key|
    options[:ssh_key] = ssh_key
  end

  opts.on('-l', '--list-opts', 'List possible values for above options') do

    client = DropletKit::Client.new(access_token: access_token)

    puts "\nAvailable Regions:(provide the shothand name as argument)"
    client.regions.all().each{ |r| puts "\t#{r.slug} - #{r.name}" }

    puts "\nAvailable Images:"
    client.images.all().each{ |r| puts "\t#{r.name}" }

    puts "\nAvailable Sizes:"
    client.sizes.all().each{ |r| puts "\t#{r.slug}" }

    puts "\nAvailable SSH keys:"
    client.ssh_keys.all().each{ |r| puts "\t#{r.name}" }

    exit
  end

  opts.on('-m', "--magnet-url magnet_url", "Magnet URL to download") do |magnet_url|
    options[:magnet_url] = magnet_url
  end

  opts.on('-h', '--help',
          'Show this message.') do
    puts opts
    exit
  end

  begin
    opts.parse!
  rescue Exception => e
    puts "Some error"
    puts opts
    exit
  end

  usage_opts = opts
end

client = DropletKit::Client.new(access_token: access_token)

all_regions = client.regions.all()

#TODO: Add more checks here
if options[:magnet_url].nil?
  puts "Invalid options"
  puts usage_opts
  exit
end

magnet_link = options[:magnet_url]

# Use private images won't have a slug. We should find the id and use that instead.
image_id = client.images.all.select{|x| x.name == options[:image]}.first.id
ssh_key_id = client.ssh_keys.all.select{|x| x.name == options[:ssh_key]}.first.id

# TODO: Probably take the name from options
droplet = DropletKit::Droplet.new(
  :name => "#{options[:image]}-jarvis",
  :size => options[:size],
  :region => options[:region],
  :image => image_id,
  :ssh_keys => [ssh_key_id]
)
puts options

begin
  new_droplet = client.droplets.create(droplet)

  #now keep checking for its status
  100.times do
    new_droplet = client.droplets.find(:id => new_droplet.id)
    if new_droplet.status == 'active'
      break;
    end
    puts "Sleeping 30 seconds"
    sleep 30
  end

  raise "The droplet has not started as it should" if new_droplet.status != 'active'

  puts "Droplet must have started!"

  host = new_droplet.networks.v4.first.ip_address

  sleep 15

  Net::SSH.start(host, USER, :keys => keys) do |session|
    puts "starting qbittorrent"
    session.exec!("qbittorrent-nox -d #{magnet_link}")
  end

  download_in_progress = true
  sleep 10

  while download_in_progress do
    Net::SSH.start(host, USER, :keys => keys) do |session|
      puts "Checking if downloads are done."
      # TODO: Your directories might be different. Check and update accordingly.
      ls_incoming_folder = session.exec!("ls ~/incomplete-torrents")
      ls_downlaod_folder = session.exec!("ls ~/Downloads")

      if ls_incoming_folder.empty? and !ls_downlaod_folder.empty?
        # Downloads completed

        # kill qbit-torrent
        session.exec!("pkill qbit")
        download_in_progress = false
        puts ls_downlaod_folder
      else
        puts "Still downloading.."
        puts ls_incoming_folder
      end
    end

    sleep 2
  end

  puts "Download Completed\r\n"

  puts 'Gonna Rsync em down to local filesystem'

  #TODO: Handle any exceptions that could be raised by Rsync block.
  #TODO: This block doesn't check if the rsync was successful. Implement
  # to check and restart rsync if it crashes.
  Open3.popen3("rsync -a --partial --progress --rsh=ssh #{USER}@#{host}:/root/Downloads ~/Desktop/") do |stdout, stderr, status, thread|
    while line=stderr.gets do
      puts(line)
    end
  end
  puts "the copy is done"

  # Delete the droplet.
  puts "Deleting the droplet."

  client.droplets.delete(id: new_droplet.id)

  puts "Droplet must have been deleted"
  puts "All Done!"

rescue Exception => e
  puts e.message
  puts e.backtrace
  exit
end
