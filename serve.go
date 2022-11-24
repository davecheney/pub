package main

import (
	"net/http"
	"os"
	"time"

	"github.com/davecheney/m/activitypub"
	"github.com/davecheney/m/mastodon"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

type ServeCmd struct {
	Addr string `help:"address to listen"`
	DSN  string `help:"data source name"`
}

func (s *ServeCmd) Run(ctx *Context) error {
	db, err := sqlx.Connect("mysql", s.DSN+"?parseTime=true")
	if err != nil {
		return err
	}

	mastodon := mastodon.NewService(db)

	r := mux.NewRouter()

	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/apps", mastodon.AppsCreate).Methods("POST")
	v1.HandleFunc("/accounts/verify_credentials", mastodon.AccountsVerify).Methods("GET")
	v1.HandleFunc("/statuses", mastodon.StatusesCreate).Methods("POST")

	v1.HandleFunc("/instance", mastodon.InstanceFetch).Methods("GET")
	v1.HandleFunc("/instance/peers", mastodon.InstancePeers).Methods("GET")

	v1.HandleFunc("/timelines/home", mastodon.TimelinesHome).Methods("GET")

	oauth := r.PathPrefix("/oauth").Subrouter()
	oauth.HandleFunc("/authorize", mastodon.OAuthAuthorize).Methods("GET", "POST")
	oauth.HandleFunc("/token", mastodon.OAuthToken).Methods("POST")

	wellknown := r.PathPrefix("/.well-known").Subrouter()
	wellknown.HandleFunc("/webfinger", mastodon.WellknownWebfinger).Methods("GET")

	activitypub := activitypub.NewService(db)

	inbox := r.Path("/inbox").Subrouter()
	inbox.Use(activitypub.ValidateSignature())
	inbox.HandleFunc("", activitypub.InboxCreate).Methods("POST")

	users := r.PathPrefix("/users").Subrouter()
	users.HandleFunc("/{username}", activitypub.UsersShow).Methods("GET")
	users.HandleFunc("/{username}/inbox", activitypub.InboxCreate).Methods("POST")

	r.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://dave.cheney.net/", http.StatusFound)
	})

	svr := &http.Server{
		Addr:         s.Addr,
		Handler:      handlers.ProxyHeaders(handlers.CombinedLoggingHandler(os.Stdout, r)),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return svr.ListenAndServe()
}
