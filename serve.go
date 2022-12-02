package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/davecheney/m/m"
	"gorm.io/gorm"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type ServeCmd struct {
	Addr   string `help:"address to listen"`
	Domain string `required:"" help:"domain name of the instance"`
}

func (s *ServeCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	if err := configureDB(db); err != nil {
		return err
	}

	svc, err := m.NewService(db, s.Domain)
	if err != nil {
		return err
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	r.Route("/api", func(r chi.Router) {
		api := svc.API()
		instance := api.Instances()
		r.Route("/v1", func(r chi.Router) {
			r.Post("/apps", api.Applications().Create)
			r.Route("/accounts", func(r chi.Router) {
				r.Get("/verify_credentials", api.Accounts().VerifyCredentials)
				r.Get("/relationships", api.Relationships().Show)
				r.Get("/filters", api.Filters().Index)
				r.Get("/lists", api.Lists().Index)
				r.Get("/instance", instance.IndexV1)
				r.Get("/instance/peers", instance.PeersShow)
				r.Get("/{id}", api.Accounts().Show)
				r.Get("/{id}/statuses", api.Accounts().StatusesShow)
			})
			r.Get("/conversations", api.Conversations().Index)
			r.Get("/custom_emojis", api.Emojis().Index)
			r.Get("/instance", instance.IndexV1)
			r.Get("/markers", api.Markers().Index)
			r.Post("/markers", api.Markers().Create)
			r.Get("/notifications", api.Notifications().Index)

			r.Get("/statuses/{id}/context", api.Contexts().Show)
			r.Post("/statuses/{id}/favourite", api.Favourites().Create)
			r.Post("/statuses/{id}/unfavourite", api.Favourites().Destroy)
			r.Get("/statuses/{id}/favourited_by", api.Favourites().Show)
			r.Get("/statuses/{id}", api.Statuses().Show)
			r.Route("/timelines", func(r chi.Router) {
				timelines := api.Timelines()
				r.Get("/home", timelines.Index)
				r.Get("/public", timelines.Public)
			})

		})
		r.Route("/v2", func(r chi.Router) {
			r.Get("/instance", instance.IndexV2)
		})
	})

	r.Post("/inbox", svc.Inboxes().Create)

	r.Route("/oauth", func(r chi.Router) {
		oauth := svc.OAuth()
		r.Get("/authorize", oauth.Authorize)
		r.Post("/authorize", oauth.Authorize)
		r.Post("/token", oauth.Token)
		r.Post("/revoke", oauth.Revoke)
	})

	r.Route("/nodeinfo", func(r chi.Router) {
		r.Get("/2.0", svc.NodeInfo().Show)
	})

	r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		// no robots, especially not you Bingbot!
		io.WriteString(w, "User-agent: *\nDisallow: /")
	})

	r.Route("/users", func(r chi.Router) {
		r.Get("/{username}", svc.Users().Show)
		r.Post("/{username}/inbox", svc.Inboxes().Create)
	})

	r.Route("/.well-known", func(r chi.Router) {
		wellknown := svc.WellKnown()
		r.Get("/webfinger", wellknown.Webfinger)
		r.Get("/host-meta", wellknown.HostMeta)
		r.Get("/nodeinfo", svc.NodeInfo().Index)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://dave.cheney.net/", http.StatusFound)
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
