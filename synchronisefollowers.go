package main

import (
	"context"
	"fmt"

	"github.com/carlmjohnson/requests"
)

type SynchroniseFollowersCmd struct {
	Source string `required:"" help:"source actor"`
	Dest   string `required:"" help:"destination actor"`
}

type OrderedCollection struct {
	Type       string `json:"type"`
	TotalItems int    `json:"totalItems"`
	First      string `json:"first"`
}

type OrderedCollectionPage struct {
	Type         string   `json:"type"`
	TotalItems   int      `json:"totalItems"`
	Next         string   `json:"next"`
	Prev         string   `json:"prev"`
	OrderedItems []string `json:"orderedItems"`
}

func (s *SynchroniseFollowersCmd) Run(_ *Context) error {
	var col OrderedCollection
	err := requests.URL(s.Source+"/following").
		Header("Accept", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).
		ToJSON(&col).
		Fetch(context.Background())
	if err != nil {
		return err
	}

	url := col.First

	for {
		var page OrderedCollectionPage
		err := requests.URL(url).
			Header("Accept", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`).
			ToJSON(&page).
			Fetch(context.Background())
		if err != nil {
			return err
		}
		for _, item := range page.OrderedItems {
			fmt.Println(item)
		}
		if page.Next == "" {
			break
		}
		url = page.Next
	}
	return nil
}
