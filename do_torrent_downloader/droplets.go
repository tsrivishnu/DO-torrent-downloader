package doTorrentDownloader

import (
	"context"
	"fmt"
	"github.com/digitalocean/godo"
)

func ListDomains() []godo.Domain {
	domains, _, _ := DoClient.Domains.List(context.TODO(), nil)
	return domains
}

func ListImages() []godo.Image {
	images, _, _ := DoClient.Images.List(context.TODO(), &godo.ListOptions{PerPage: 200})
	return images
}

func ListRegions() []godo.Region {
	regions, _, _ := DoClient.Regions.List(context.TODO(), &godo.ListOptions{PerPage: 100})
	return regions
}

func ListKeys() []godo.Key {
	keys, _, _ := DoClient.Keys.List(context.TODO(), &godo.ListOptions{PerPage: 100})
	return keys
}

func FindKey(name string) godo.Key {
	var ssh_key godo.Key
	for _, key := range ListKeys() {
		if key.Name == "bpi@casa.v.totakura.in" {
			ssh_key = key
			break
		}
	}
	return ssh_key
}

func CreateDroplet(config *config) (*godo.Droplet, error) {
	createRequest := &godo.DropletCreateRequest{
		Name:   config.DropletName,
		Region: config.Region,
		Size:   config.Size,
		Image: godo.DropletCreateImage{
			Slug: config.ImageSlug,
		},
		SSHKeys: []godo.DropletCreateSSHKey{
			{Fingerprint: FindKey(config.SshKey).Fingerprint},
		},
	}

	newDroplet, _, err := DoClient.Droplets.Create(context.TODO(), createRequest)

	if err != nil {
		fmt.Printf("Something bad happened: %s\n\n", err)
		return nil, err
	}
	return newDroplet, nil
}

func GetByIp(ip string) *godo.Droplet {

	droplets, _, err := DoClient.Droplets.List(context.TODO(), &godo.ListOptions{PerPage: 200})
	// TODO: Support searching with pagination.
	if err != nil {
		panic(fmt.Sprintf("Error looking for Droplet with IP: %v:%v", ip, err))
	}

	for _, droplet := range droplets {
		dropletIp, _ := droplet.PublicIPv4()
		if dropletIp == ip {
			return &droplet
		}
	}
	return nil
}
