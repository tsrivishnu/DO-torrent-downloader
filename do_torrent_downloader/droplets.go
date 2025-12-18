package doTorrentDownloader

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

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
		Tags: []string{config.DropletTag},
	}

	newDroplet, _, err := DoClient.Droplets.Create(context.TODO(), createRequest)

	if err != nil {
		fmt.Printf("Something bad happened: %s\n\n", err)
		return nil, err
	}
	return newDroplet, nil
}

func DeleteDropletsByTag(tag string) {
	if tag == "" {
		fmt.Println("No tag specified, skipping cleanup.")
		return
	}

	opt := &godo.ListOptions{PerPage: 200}
	var allDroplets []godo.Droplet

	// 1. Collect all droplets
	for {
		droplets, resp, err := DoClient.Droplets.ListByTag(context.TODO(), tag, opt)
		if err != nil {
			fmt.Printf("Error listing droplets by tag: %v\n", err)
			return
		}
		allDroplets = append(allDroplets, droplets...)

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			break
		}
		opt.Page = page + 1
	}

	if len(allDroplets) == 0 {
		fmt.Printf("No droplets found with tag: %s\n", tag)
		return
	}

	// 2. Prompt for confirmation
	fmt.Printf("Found %d droplets with tag '%s'.\n", len(allDroplets), tag)
	for _, d := range allDroplets {
		fmt.Printf("- %s (ID: %d, IP: %s)\n", d.Name, d.ID, func() string {
			ip, _ := d.PublicIPv4()
			return ip
		}())
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Are you sure you want to delete them? [y/n]: ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "n" || response == "no" {
			fmt.Println("Aborted.")
			return
		} else if response == "y" || response == "yes" {
			break
		}
	}

	// 3. Delete droplets
	for _, d := range allDroplets {
		fmt.Printf("Deleting droplet: %s (ID: %d)\n", d.Name, d.ID)
		_, err := DoClient.Droplets.Delete(context.TODO(), d.ID)
		if err != nil {
			fmt.Printf("Error deleting droplet %d: %v\n", d.ID, err)
		} else {
			fmt.Println("Deleted.")
		}
	}
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
