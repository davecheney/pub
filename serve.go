package main

import (
	"net/http"
	"os"
	"time"

	"github.com/davecheney/m/activitypub"
	"github.com/davecheney/m/m"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
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

	r := mux.NewRouter()
	r = r.Host(s.Domain).Subrouter()

	v1 := r.PathPrefix("/api/v1").Subrouter()
	api := svc.API()
	apps := api.Applications()
	v1.HandleFunc("/apps", apps.Create).Methods(http.MethodPost)
	v1.HandleFunc("/markers", api.Markers().Index).Methods("GET")
	v1.HandleFunc("/markers", api.Markers().Create).Methods("POST")

	accounts := v1.PathPrefix("/accounts").Subrouter()
	accounts.HandleFunc("/verify_credentials", api.Accounts().VerifyCredentials).Methods("GET")
	accounts.HandleFunc("/relationships", api.Relationships().Show).Methods("GET")

	account := accounts.PathPrefix("/{id:[0-9]+}").Subrouter()
	account.HandleFunc("", api.Accounts().Show).Methods("GET")
	account.HandleFunc("/statuses", api.Accounts().StatusesShow).Methods("GET")

	conversations := api.Conversations()
	v1.HandleFunc("/conversations", conversations.Index).Methods("GET")

	statuses := v1.PathPrefix("/statuses").Subrouter()
	statuses.HandleFunc("", api.Statuses().Create).Methods("POST")

	status := statuses.PathPrefix("/{id:[0-9]+}").Subrouter()
	status.HandleFunc("", api.Statuses().Show).Methods("GET")
	status.HandleFunc("/context", api.Contexts().Show).Methods("GET")
	status.HandleFunc("/favourite", api.Favourites().Create).Methods("POST")
	status.HandleFunc("/unfavourite", api.Favourites().Destroy).Methods("POST")
	status.HandleFunc("/favourited_by", api.Favourites().Show).Methods("GET")

	emojis := api.Emojis()
	v1.HandleFunc("/custom_emojis", emojis.Index).Methods("GET")

	notifications := api.Notifications()
	v1.HandleFunc("/notifications", notifications.Index).Methods("GET")

	instance := api.Instances()
	v1.HandleFunc("/instance", instance.IndexV1).Methods("GET")
	v1.HandleFunc("/instance/peers", instance.PeersShow).Methods("GET")

	filters := api.Filters()
	v1.HandleFunc("/filters", filters.Index).Methods("GET")

	timelines := api.Timelines()
	v1.HandleFunc("/timelines/home", timelines.Index).Methods("GET")
	v1.HandleFunc("/timelines/public", timelines.Public).Methods("GET")

	v1.HandleFunc("/lists", api.Lists().Index).Methods("GET")

	v2 := r.PathPrefix("/api/v2").Subrouter()
	v2.HandleFunc("/instance", instance.IndexV2).Methods("GET")

	oauth := svc.OAuth()
	r.HandleFunc("/oauth/authorize", oauth.Authorize).Methods("GET", "POST")
	r.HandleFunc("/oauth/token", oauth.Token).Methods("POST")
	r.HandleFunc("/oauth/revoke", oauth.Revoke).Methods("POST")

	wk := r.PathPrefix("/.well-known").Subrouter()
	wellknown := svc.WellKnown()
	wk.HandleFunc("/webfinger", wellknown.Webfinger).Methods("GET")
	wk.HandleFunc("/host-meta", wellknown.HostMeta).Methods("GET")
	wk.HandleFunc("/nodeinfo", svc.NodeInfo().Index).Methods("GET")

	ni := r.PathPrefix("/nodeinfo").Subrouter()
	ni.HandleFunc("/2.0", svc.NodeInfo().Show).Methods("GET")

	users := r.PathPrefix("/users").Subrouter()
	users.HandleFunc("/{username}", svc.Users().Show).Methods("GET")

	inbox := activitypub.NewInbox(db, svc)
	users.HandleFunc("/{username}/inbox", inbox.Create).Methods("POST")
	r.Path("/inbox").HandlerFunc(inbox.Create).Methods("POST")

	r.Path("/robots.txt").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		// no robots, especially not you Bingbot!
		w.Write([]byte("User-agent: *\nDisallow: /"))
	})

	r.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://dave.cheney.net/", http.StatusFound)
	})

	svr := &http.Server{
		Addr:         s.Addr,
		Handler:      handlers.ProxyHeaders(handlers.LoggingHandler(os.Stdout, r)),
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
