package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/davecheney/m/m"
	"github.com/davecheney/m/mastodon"
	"github.com/davecheney/m/oauth"
	"gorm.io/gorm"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type ServeCmd struct {
	Addr string `help:"address to listen"`
}

func (s *ServeCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	if err := configureDB(db); err != nil {
		return err
	}

	svc := m.NewService(db)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	r.Route("/api", func(r chi.Router) {
		api := svc.API()
		mastodon := mastodon.NewService(svc)
		instance := mastodon.Instances()
		r.Route("/v1", func(r chi.Router) {
			r.Post("/apps", api.Applications().Create)
			r.Route("/accounts", func(r chi.Router) {
				accounts := mastodon.Accounts()
				r.Get("/verify_credentials", accounts.VerifyCredentials)
				r.Patch("/update_credentials", accounts.Update)
				r.Get("/relationships", mastodon.Relationships().Show)
				r.Get("/filters", mastodon.Filters().Index)
				r.Get("/lists", mastodon.Lists().Index)
				r.Get("/instance", instance.IndexV1)
				r.Get("/instance/peers", instance.PeersShow)
				r.Get("/{id}", accounts.Show)
				r.Get("/{id}/statuses", accounts.StatusesShow)
				r.Post("/{id}/follow", mastodon.Relationships().Create)
				r.Post("/{id}/unfollow", mastodon.Relationships().Destroy)
			})
			r.Get("/conversations", mastodon.Conversations().Index)
			r.Get("/custom_emojis", mastodon.Emojis().Index)
			r.Get("/instance", instance.IndexV1)
			r.Get("/markers", mastodon.Markers().Index)
			r.Post("/markers", mastodon.Markers().Create)
			r.Get("/notifications", mastodon.Notifications().Index)

			r.Post("/statuses", api.Statuses().Create)
			r.Get("/statuses/{id}/context", mastodon.Contexts().Show)
			r.Post("/statuses/{id}/favourite", mastodon.Favourites().Create)
			r.Post("/statuses/{id}/unfavourite", mastodon.Favourites().Destroy)
			r.Get("/statuses/{id}/favourited_by", mastodon.Favourites().Show)
			r.Get("/statuses/{id}", api.Statuses().Show)
			r.Delete("/statuses/{id}", api.Statuses().Destroy)
			r.Route("/timelines", func(r chi.Router) {
				timelines := mastodon.Timelines()
				r.Get("/home", timelines.Home)
				r.Get("/public", timelines.Public)
			})

		})
		r.Route("/v2", func(r chi.Router) {
			r.Get("/instance", instance.IndexV2)
			r.Get("/search", mastodon.Search().Index)
		})
		r.Route("/nodeinfo", func(r chi.Router) {
			r.Get("/2.0", svc.NodeInfo().Show)
		})
	})

	activitypub := svc.ActivityPub()
	r.Post("/inbox", activitypub.Inboxes().Create)

	r.Route("/oauth", func(r chi.Router) {
		oauth := oauth.New(db)
		r.Get("/authorize", oauth.Authorize)
		r.Post("/authorize", oauth.Authorize)
		r.Post("/token", oauth.Token)
		r.Post("/revoke", oauth.Revoke)
	})

	r.Route("/users/{username}", func(r chi.Router) {

		r.Get("/", svc.Users().Show)
		r.Post("/inbox", activitypub.Inboxes().Create)
		r.Get("/outbox", activitypub.Outboxes().Index)
		r.Get("/followers", activitypub.Followers().Index)
		r.Get("/following", activitypub.Following().Index)
		r.Get("/collections/{collection}", activitypub.Collections().Show)
	})

	r.Route("/.well-known", func(r chi.Router) {
		wellknown := svc.WellKnown()
		r.Get("/webfinger", wellknown.Webfinger)
		r.Get("/host-meta", wellknown.HostMeta)
		r.Get("/nodeinfo", svc.NodeInfo().Index)
	})

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		route = strings.Replace(route, "/*/", "/", -1)
		fmt.Printf("%s %s\n", method, route)
		return nil
	}

	if err := chi.Walk(r, walkFunc); err != nil {
		fmt.Printf("Logging err: %s\n", err.Error())
	}

	svr := &http.Server{
		Addr:         s.Addr,
		Handler:      r,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return svr.ListenAndServe()
}

func configureDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)

	return nil
}
