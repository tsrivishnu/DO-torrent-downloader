package doTorrentDownloader

import (
	"context"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

var DoClient *godo.Client

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func InitDoClient(pat string) {
	tokenSource := &TokenSource{
		AccessToken: pat,
	}
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	DoClient = godo.NewClient(oauthClient)
}
