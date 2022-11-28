package main

import (
	"net/http"
	"os"
	"time"

	"github.com/davecheney/m/activitypub"
	"github.com/davecheney/m/mastodon"
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

	// the instance this service represents
	var theInstance mastodon.Instance
	if err := db.Where("domain = ?", s.Domain).First(&theInstance).Error; err != nil {
		return err
	}

	r := mux.NewRouter()

	v1 := r.PathPrefix("/api/v1").Subrouter()
	apps := mastodon.NewApplications(db, &theInstance)
	v1.HandleFunc("/apps", apps.New).Methods(http.MethodPost)

	accounts := mastodon.NewAccounts(db, &theInstance)
	v1.HandleFunc("/accounts/verify_credentials", accounts.VerifyCredentials).Methods("GET")
	v1.HandleFunc("/accounts/relationships", accounts.Relationships).Methods("GET")
	v1.HandleFunc("/accounts/{id}", accounts.Show).Methods("GET")
	v1.HandleFunc("/accounts/{id}/statuses", accounts.StatusesShow).Methods("GET")

	statuses := mastodon.NewStatuses(db, &theInstance)
	v1.HandleFunc("/statuses", statuses.Create).Methods("POST")

	emojis := mastodon.NewEmojis(db)
	v1.HandleFunc("/custom_emojis", emojis.Index).Methods("GET")

	notifications := mastodon.NewNotifications(db)
	v1.HandleFunc("/notifications", notifications.Index).Methods("GET")

	instance := mastodon.NewInstances(db, &theInstance)
	v1.HandleFunc("/instance", instance.IndexV1).Methods("GET")
	v1.HandleFunc("/instance/peers", instance.PeersShow).Methods("GET")

	filters := mastodon.NewFilters(db)
	v1.HandleFunc("/filters", filters.Index).Methods("GET")

	timelines := mastodon.NewTimeslines(db, &theInstance)
	v1.HandleFunc("/timelines/home", timelines.Index).Methods("GET")
	v1.HandleFunc("/timelines/public", timelines.Index).Methods("GET")

	lists := mastodon.NewLists(db, &theInstance)
	v1.HandleFunc("/lists", lists.Index).Methods("GET")

	v2 := r.PathPrefix("/api/v2").Subrouter()
	v2.HandleFunc("/instance", instance.IndexV2).Methods("GET")

	oauth := mastodon.NewOAuth(db, &theInstance)
	r.HandleFunc("/oauth/authorize", oauth.Authorize).Methods("GET", "POST")
	r.HandleFunc("/oauth/token", oauth.Token).Methods("POST")
	r.HandleFunc("/oauth/revoke", oauth.Revoke).Methods("POST")

	wk := r.PathPrefix("/.well-known").Subrouter()
	wellknown := mastodon.NewWellKnown(db, &theInstance)
	wk.HandleFunc("/webfinger", wellknown.Webfinger).Methods("GET")
	wk.HandleFunc("/host-meta", wellknown.HostMeta).Methods("GET")
	wk.HandleFunc("/nodeinfo", wellknown.NodeInfo).Methods("GET")

	ni := r.PathPrefix("/nodeinfo").Subrouter()
	nodeinfo := mastodon.NewNodeInfo(db, theInstance.Domain)
	ni.HandleFunc("/2.0", nodeinfo.Get).Methods("GET")

	users := activitypub.NewUsers(db, &theInstance)
	r.HandleFunc("/users/{username}", users.Show).Methods("GET")
	r.HandleFunc("/users/{username}/inbox", users.InboxCreate).Methods("POST")
	activitypub := activitypub.NewService(db, &theInstance)

	inbox := r.Path("/inbox").Subrouter()
	inbox.Use(activitypub.ValidateSignature())
	inbox.HandleFunc("", users.InboxCreate).Methods("POST")

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
