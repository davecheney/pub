package main

import (
	"context"
	"fmt"
	"net/url"
	"path"

	"github.com/carlmjohnson/requests"
	"github.com/davecheney/m/internal/activitypub"
	"github.com/davecheney/m/m"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FollowCmd struct {
	Object string `help:"object to follow"`
	Actor  string `help:"actor to follow with"`
}

func (f *FollowCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}
	var instance m.Instance
	if err := db.First(&instance).Error; err != nil {
		return err
	}

	username, domain, err := parseActor(f.Actor)
	if err != nil {
		return err
	}

	account, err := findLocalAccount(db, username, domain)
	if err != nil {
		return err
	}

	var actor map[string]interface{}
	if err := requests.URL(f.Actor).Accept(`application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).ToJSON(&actor).Fetch(context.Background()); err != nil {
		return err
	}

	u, err := url.Parse(f.Object)
	if err != nil {
		return err
	}
	c, err := activitypub.NewClient(account.Actor.PublicKeyID(), nil) // account.LocalAccount.PrivateKey)
	if err != nil {
		return err
	}
	return c.Post(fmt.Sprintf("%s://%s/inbox", u.Scheme, u.Host), map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       fmt.Sprintf("https://%s/%s", instance.Domain, uuid.New().String()),
		"type":     "Follow",
		"object":   f.Object,
		"actor":    f.Actor,
	})
}

func findLocalAccount(db *gorm.DB, username, domain string) (*m.Account, error) {
	var account m.Account
	if err := db.Where("username = ? AND domain = ?", username, domain).Joins("LocalAccount").First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func parseActor(acct string) (string, string, error) {
	url, err := url.Parse(acct)
	if err != nil {
		return "", "", fmt.Errorf("splitAcct: %w", err)
	}
	return path.Base(url.Path), url.Host, nil
}
