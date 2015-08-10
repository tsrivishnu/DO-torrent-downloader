#!/usr/bin/env ruby

require 'Digitalocean'
require 'optparse'

# TODO: Get these values from a env
Digitalocean.client_id  = ""
Digitalocean.api_key    = ""

options = { :size => "512MB" }

usage_opts = nil

ARGV.options do |opts|

  opts.on('-r', '--region region_name', 'Select Regions name') do |region|
    options[:region] = region
  end

  opts.on('-i', 'image [image_name]', 'Select image name') do |image|
    options[:image] = image
  end

  opts.on('-s', '--size [size]', 'Select size of the droplet') do |size|
    option[:size] = size
  end

  opts.in('-k', '--key ssh_key', 'SSH key name') do |ssh_key|
    options[:ssh_key] = ssh_key
  end

  opts.on('-h', '--help',
          'Show this message.') do
    puts opts
    exit
  end

  begin
    opts.parse!
  rescue
    puts opts
    exit
  end

  usage_opts = opts
end

# Pick the region
region = Digitalocean::Region.all.select{
  |region| region[:name] == options[:region]
}.first

if region.nil?
  puts opts
  exit
end
