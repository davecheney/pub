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
	Addr string `help:"address to listen"`
}

func (s *ServeCmd) Run(ctx *Context) error {
	db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	if err != nil {
		return err
	}

	// user := &mastodon.User{
	// 	Email:             "dave@cheney.net",
	// 	EncryptedPassword: []byte("$2a$04$0k4j6NbaaPSrGwDb0ufOK.KKYBCigiXk95YNUAQXk74CQVg4FUrre"),
	// }
	// if err := db.Create(user).Error; err != nil {
	// 	return err
	// }
	// var user mastodon.User
	// db.First(&user)
	// user.Account = mastodon.Account{
	// 	Username:    "dave",
	// 	Domain:      "cheney.net",
	// 	Acct:        "dave@cheney.net",
	// 	DisplayName: "Dave Cheney",
	// 	Locked:      false,
	// 	Bot:         false,
	// 	Note:        "I like cheese!",
	// 	URL:         "https://cheney.net/@dave",
	// 	Avatar:      "https://cheney.net/avatar.png",
	// 	Header:      "https://cheney.net/header.png",
	// }
	// if err := db.Save(&user).Error; err != nil {
	// 	return err
	// }

	m := mastodon.NewService(db)
	emojis := mastodon.NewEmojis(db)
	statuses := mastodon.NewStatuses(db)
	oauth := mastodon.NewOAuth(db)
	accounts := mastodon.NewAccounts(db)
	instance := mastodon.NewInstance(db)
	apps := mastodon.NewApplications(db)
	timeline := mastodon.NewTimeline(db)

	r := mux.NewRouter()

	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/apps", apps.New).Methods(http.MethodPost)
	v1.HandleFunc("/accounts/verify_credentials", accounts.VerifyCredentials).Methods("GET")
	v1.HandleFunc("/accounts/{id}", m.AccountsFetch).Methods("GET")
	v1.HandleFunc("/accounts/{id}/statuses", m.AccountsStatusesFetch).Methods("GET")
	v1.HandleFunc("/statuses", statuses.Create).Methods("POST")
	v1.HandleFunc("/custom_emojis", emojis.Index).Methods("GET")

	v1.HandleFunc("/instance", instance.Index).Methods("GET")
	v1.HandleFunc("/instance/peers", instance.Peers).Methods("GET")

	v1.HandleFunc("/timelines/home", timeline.Index).Methods("GET")

	r.HandleFunc("/oauth/authorize", oauth.Authorize).Methods("GET", "POST")
	r.HandleFunc("/oauth/token", oauth.Token).Methods("POST")
	r.HandleFunc("/oauth/revoke", oauth.Revoke).Methods("POST")

	wellknown := r.PathPrefix("/.well-known").Subrouter()
	wellknown.HandleFunc("/webfinger", m.WellknownWebfinger).Methods("GET")

	users := activitypub.NewUsers(db)
	r.HandleFunc("/users/{username}", users.Show).Methods("GET")
	r.HandleFunc("/users/{username}/inbox", users.InboxCreate).Methods("POST")
	activitypub := activitypub.NewService(db)

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
