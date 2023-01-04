package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/davecheney/pub/activitypub"
	"github.com/davecheney/pub/internal/group"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/mastodon"
	"github.com/davecheney/pub/media"
	"github.com/davecheney/pub/oauth"
	"github.com/davecheney/pub/wellknown"
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
		envFn := func(r *http.Request) *mastodon.Env {
			return &mastodon.Env{
				Env: &models.Env{
					DB: db.WithContext(r.Context()),
				},
			}
		}
		r.Route("/v1", func(r chi.Router) {
			r.Post("/apps", httpx.HandlerFunc(envFn, mastodon.AppsCreate))
			r.Route("/accounts", func(r chi.Router) {
				r.Get("/verify_credentials", httpx.HandlerFunc(envFn, mastodon.AccountsVerifyCredentials))
				r.Patch("/update_credentials", httpx.HandlerFunc(envFn, mastodon.AccountsUpdateCredentials))
				r.Get("/relationships", httpx.HandlerFunc(envFn, mastodon.RelationshipsShow))
				r.Get("/{id}", httpx.HandlerFunc(envFn, mastodon.AccountsShow))
				r.Get("/{id}/lists", httpx.HandlerFunc(envFn, mastodon.AccountsShowListMembership)) // todo
				r.Get("/{id}/statuses", httpx.HandlerFunc(envFn, mastodon.AccountsStatusesShow))
				r.Post("/{id}/follow", httpx.HandlerFunc(envFn, mastodon.RelationshipsCreate))
				r.Get("/{id}/followers", httpx.HandlerFunc(envFn, mastodon.AccountsFollowersShow))
				r.Get("/{id}/following", httpx.HandlerFunc(envFn, mastodon.AccountsFollowingShow))
				r.Post("/{id}/unfollow", httpx.HandlerFunc(envFn, mastodon.RelationshipsDestroy))
				r.Post("/{id}/mute", httpx.HandlerFunc(envFn, mastodon.MutesCreate))
				r.Post("/{id}/unmute", httpx.HandlerFunc(envFn, mastodon.MutesDestroy))
				r.Post("/{id}/block", httpx.HandlerFunc(envFn, mastodon.BlocksCreate))
				r.Post("/{id}/unblock", httpx.HandlerFunc(envFn, mastodon.BlocksDestroy))
			})
			r.Get("/blocks", httpx.HandlerFunc(envFn, mastodon.BlocksIndex))
			r.Get("/conversations", httpx.HandlerFunc(envFn, mastodon.ConversationsIndex))
			r.Get("/custom_emojis", httpx.HandlerFunc(envFn, mastodon.EmojisIndex))
			r.Get("/directory", httpx.HandlerFunc(envFn, mastodon.DirectoryIndex))
			r.Get("/filters", httpx.HandlerFunc(envFn, mastodon.FiltersIndex))
			r.Get("/lists", httpx.HandlerFunc(envFn, mastodon.ListsIndex))
			r.Post("/lists", httpx.HandlerFunc(envFn, mastodon.ListsCreate))
			r.Get("/lists/{id}", httpx.HandlerFunc(envFn, mastodon.ListsShow))
			r.Get("/lists/{id}/accounts", httpx.HandlerFunc(envFn, mastodon.ListsViewMembers)) // todo
			r.Post("/lists/{id}/accounts", httpx.HandlerFunc(envFn, mastodon.ListsAddMembers))
			r.Delete("/lists/{id}/accounts", httpx.HandlerFunc(envFn, mastodon.ListsRemoveMembers))
			r.Get("/instance", httpx.HandlerFunc(envFn, mastodon.InstancesIndexV1))
			r.Options("/instance", func(w http.ResponseWriter, r *http.Request) {
				// wtf elk, why do you send an OPTIONS request to /instance?
				x, _ := httputil.DumpRequest(r, true)
				fmt.Println(string(x))
				w.WriteHeader(http.StatusOK)
			})
			r.Get("/instance/", httpx.HandlerFunc(envFn, mastodon.InstancesIndexV1)) // sigh
			r.Get("/instance/peers", httpx.HandlerFunc(envFn, mastodon.InstancesPeersShow))
			r.Get("/instance/activity", httpx.HandlerFunc(envFn, mastodon.InstancesActivityShow))
			r.Get("/instance/domain_blocks", httpx.HandlerFunc(envFn, mastodon.InstancesDomainBlocksShow))
			r.Get("/markers", httpx.HandlerFunc(envFn, mastodon.MarkersIndex))
			r.Post("/markers", httpx.HandlerFunc(envFn, mastodon.MarkersCreate))
			r.Get("/mutes", httpx.HandlerFunc(envFn, mastodon.MutesIndex))
			r.Get("/notifications", httpx.HandlerFunc(envFn, mastodon.NotificationsIndex))

			r.Post("/statuses", httpx.HandlerFunc(envFn, mastodon.StatusesCreate))
			r.Get("/statuses/{id}/context", httpx.HandlerFunc(envFn, mastodon.StatusesContextsShow))
			r.Post("/statuses/{id}/favourite", httpx.HandlerFunc(envFn, mastodon.FavouritesCreate))
			r.Post("/statuses/{id}/unfavourite", httpx.HandlerFunc(envFn, mastodon.FavouritesDestroy))
			r.Get("/statuses/{id}/favourited_by", httpx.HandlerFunc(envFn, mastodon.FavouritesShow))
			r.Get("/statuses/{id}", httpx.HandlerFunc(envFn, mastodon.StatusesShow))
			r.Delete("/statuses/{id}", httpx.HandlerFunc(envFn, mastodon.StatusesDestroy))
			r.Route("/timelines", func(r chi.Router) {
				r.Get("/home", httpx.HandlerFunc(envFn, mastodon.TimelinesHome))
				r.Get("/public", httpx.HandlerFunc(envFn, mastodon.TimelinesPublic))
				r.Get("/list/{id}", httpx.HandlerFunc(envFn, mastodon.TimelinesListShow))
				r.Get("/tag/{tag}", httpx.HandlerFunc(envFn, mastodon.TimelinesTagShow))
			})

		})
		r.Route("/v2", func(r chi.Router) {
			r.Get("/instance", httpx.HandlerFunc(envFn, mastodon.InstancesIndexV2))
			r.Get("/search", httpx.HandlerFunc(envFn, mastodon.SearchIndex))
		})
	})

	envFn := func(r *http.Request) *activitypub.Env {
		return &activitypub.Env{
			Env: &models.Env{
				DB: db.WithContext(r.Context()),
			},
		}
	}
	r.Post("/inbox", httpx.HandlerFunc(envFn, activitypub.InboxCreate))

	r.Route("/oauth", func(r chi.Router) {
		r.Get("/authorize", httpx.HandlerFunc(envFn, oauth.AuthorizeNew))
		r.Post("/authorize", httpx.HandlerFunc(envFn, oauth.AuthorizeCreate))
		r.Post("/token", httpx.HandlerFunc(envFn, oauth.TokenCreate))
		r.Post("/revoke", httpx.HandlerFunc(envFn, oauth.TokenDestroy))
	})

	r.Route("/u/{username}", func(r chi.Router) {
		r.Get("/", httpx.HandlerFunc(envFn, activitypub.UsersShow))
		r.Post("/inbox", httpx.HandlerFunc(envFn, activitypub.InboxCreate))
		r.Get("/outbox", activitypub.OutboxIndex)
		r.Get("/followers", activitypub.FollowersIndex)
		r.Get("/following", activitypub.FollowingIndex)
		r.Get("/collections/{collection}", activitypub.CollectionsShow)
	})

	r.Route("/.well-known", func(r chi.Router) {
		r.Get("/webfinger", httpx.HandlerFunc(envFn, wellknown.WebfingerShow))
		r.Get("/host-meta", httpx.HandlerFunc(envFn, wellknown.HostMetaIndex))
		r.Get("/nodeinfo", httpx.HandlerFunc(envFn, wellknown.NodeInfoShow))
	})
	r.Get("/nodeinfo/2.0", httpx.HandlerFunc(envFn, wellknown.NodeInfoShow))

	modelEnvFn := func(r *http.Request) *models.Env {
		return &models.Env{
			DB: db.WithContext(r.Context()),
		}
	}

	r.Get("/media/{kind}/{hash}/{id}", httpx.HandlerFunc(modelEnvFn, media.Show))

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
	g.Add(activitypub.NewRelationshipRequestProcessor(db).Run)
	g.Add(activitypub.NewReactionRequestProcessor(db).Run)

	return g.Wait()
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
