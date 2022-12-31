package main

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/davecheney/m/activitypub"
	"github.com/davecheney/m/internal/group"
	"github.com/davecheney/m/internal/models"
	"github.com/davecheney/m/mastodon"
	"github.com/davecheney/m/oauth"
	"github.com/davecheney/m/wellknown"
	"gorm.io/gorm"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type ServeCmd struct {
	Addr             string `help:"address to listen" default:"127.0.0.1:9999"`
	DebugPrintRoutes bool   `help:"print routes to stdout on startup"`
	LogHTTP          bool   `help:"log HTTP requests"`
}

func (s *ServeCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	if err := configureDB(db); err != nil {
		return err
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	if s.LogHTTP {
		r.Use(middleware.Logger)
	}

	r.Route("/api", func(r chi.Router) {
		mastodon := mastodon.NewService(db)
		instance := mastodon.Instances()
		r.Route("/v1", func(r chi.Router) {
			r.Post("/apps", mastodon.Applications().Create)
			r.Route("/accounts", func(r chi.Router) {
				accounts := mastodon.Accounts()
				r.Get("/verify_credentials", accounts.VerifyCredentials)
				r.Patch("/update_credentials", accounts.Update)
				r.Get("/relationships", mastodon.Relationships().Show)
				r.Get("/filters", mastodon.Filters().Index)
				r.Get("/instance", instance.IndexV1)
				r.Get("/instance/peers", instance.PeersShow)
				r.Get("/{id}", accounts.Show)
				r.Get("/{id}/lists", mastodon.Lists().Show)
				r.Get("/{id}/statuses", accounts.StatusesShow)
				r.Post("/{id}/follow", mastodon.Relationships().Create)
				r.Get("/{id}/followers", accounts.FollowersShow)
				r.Get("/{id}/following", accounts.FollowingShow)
				r.Post("/{id}/unfollow", mastodon.Relationships().Destroy)
				r.Post("/{id}/mute", mastodon.Mutes().Create)
				r.Post("/{id}/unmute", mastodon.Mutes().Destroy)
				r.Post("/{id}/block", mastodon.Blocks().Create)
				r.Post("/{id}/unblock", mastodon.Blocks().Destroy)
			})
			r.Get("/blocks", mastodon.Blocks().Index)
			r.Get("/conversations", mastodon.Conversations().Index)
			r.Get("/custom_emojis", mastodon.Emojis().Index)
			r.Get("/directory", mastodon.Directory().Index)
			r.Get("/filters", mastodon.Filters().Index)
			r.Get("/lists", mastodon.Lists().Index)
			r.Get("/instance", instance.IndexV1)
			r.Get("/instance/", instance.IndexV1) // sigh
			r.Get("/instance/peers", instance.PeersShow)
			r.Get("/instance/activity", instance.ActivityShow)
			r.Get("/instance/domain_blocks", instance.DomainBlocksShow)
			r.Get("/markers", mastodon.Markers().Index)
			r.Post("/markers", mastodon.Markers().Create)
			r.Get("/mutes", mastodon.Mutes().Index)
			r.Get("/notifications", mastodon.Notifications().Index)

			r.Post("/statuses", mastodon.Statuses().Create)
			r.Get("/statuses/{id}/context", mastodon.Contexts().Show)
			r.Post("/statuses/{id}/favourite", mastodon.Favourites().Create)
			r.Post("/statuses/{id}/unfavourite", mastodon.Favourites().Destroy)
			r.Get("/statuses/{id}/favourited_by", mastodon.Favourites().Show)
			r.Get("/statuses/{id}", mastodon.Statuses().Show)
			r.Delete("/statuses/{id}", mastodon.Statuses().Destroy)
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
			r.Get("/2.0", wellknown.NewService(db).NodeInfo().Show)
		})
	})

	ap := activitypub.NewService(db)
	getKey := func(keyID string) (crypto.PublicKey, error) {
		actorId := trimKeyId(keyID)
		var instance models.Instance
		if err := db.Joins("Admin").Preload("Admin.Actor").First(&instance, "admin_id is not null").Error; err != nil {
			return nil, err
		}
		fetcher := activitypub.NewRemoteActorFetcher(instance.Admin, db)
		actor, err := models.NewActors(db).FindOrCreate(actorId, fetcher.Fetch)
		if err != nil {
			return nil, err
		}
		return pemToPublicKey(actor.PublicKey)
	}
	r.Post("/inbox", ap.Inboxes(getKey).Create)

	r.Route("/oauth", func(r chi.Router) {
		oauth := oauth.New(db)
		r.Get("/authorize", oauth.Authorize)
		r.Post("/authorize", oauth.Authorize)
		r.Post("/token", oauth.Token)
		r.Post("/revoke", oauth.Revoke)
	})

	r.Route("/u/{username}", func(r chi.Router) {
		r.Get("/", ap.Users().Show)
		r.Post("/inbox", ap.Inboxes(getKey).Create)
		r.Get("/outbox", ap.Outboxes().Index)
		r.Get("/followers", ap.Followers().Index)
		r.Get("/following", ap.Following().Index)
		r.Get("/collections/{collection}", ap.Collections().Show)
	})

	r.Route("/.well-known", func(r chi.Router) {
		wellknown := wellknown.NewService(db)
		r.Get("/webfinger", wellknown.Webfinger().Show)
		r.Get("/host-meta", wellknown.HostMeta)
		r.Get("/nodeinfo", wellknown.NodeInfo().Index)
	})

	if s.DebugPrintRoutes {
		walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			route = strings.Replace(route, "/*/", "/", -1)
			fmt.Printf("%s %s\n", method, route)
			return nil
		}

		if err := chi.Walk(r, walkFunc); err != nil {
			fmt.Printf("Logging err: %s\n", err.Error())
		}
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	g := group.New(signalCtx)
	g.AddContext(func(ctx context.Context) error {
		fmt.Println("http.ListenAndServe", s.Addr, "started")
		defer fmt.Println("http.ListenAndServe", s.Addr, "stopped")
		svr := &http.Server{
			Addr:         s.Addr,
			Handler:      r,
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
		}
		go func() {
			<-ctx.Done()
			svr.Shutdown(ctx)
		}()
		return svr.ListenAndServe()
	})
	relrp := activitypub.NewRelationshipRequestProcessor(db)
	g.Add(relrp.Run)
	reacrp := activitypub.NewReactionRequestProcessor(db)
	g.Add(reacrp.Run)

	return g.Wait()
}

func pemToPublicKey(key []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(key)
	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("pemToPublicKey: invalid pem type: %s", block.Type)
	}
	var publicKey interface{}
	var err error
	if publicKey, err = x509.ParsePKIXPublicKey(block.Bytes); err != nil {
		return nil, fmt.Errorf("pemToPublicKey: parsepkixpublickey: %w", err)
	}
	return publicKey, nil
}

// trimKeyId removes the #main-key suffix from the key id.
func trimKeyId(id string) string {
	if i := strings.Index(id, "#"); i != -1 {
		return id[:i]
	}
	return id
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
