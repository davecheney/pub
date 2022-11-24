package mastodon

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"

	"github.com/go-json-experiment/json"
)

// Service implements a Mastodon service.
type Service struct {
	db *gorm.DB
}

// NewService returns a new instance of Service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

func (svc *Service) accounts() *accounts {
	return &accounts{db: svc.db}
}

func (svc *Service) applications() *applications {
	return &applications{db: svc.db}
}
func (svc *Service) tokens() *tokens {
	return &tokens{db: svc.db}
}

func (svc *Service) statuses() *statuses {
	db, _ := svc.db.DB()
	return &statuses{db: sqlx.NewDb(db, "mysql")}
}

func (svc *Service) users() *users {
	return &users{db: svc.db}
}

func (svc *Service) AppsCreate(w http.ResponseWriter, r *http.Request) {
	var params struct {
		ClientName   string  `json:"client_name"`
		Website      *string `json:"website"`
		RedirectURIs string  `json:"redirect_uris"`
		Scopes       string  `json:"scopes"`
	}
	if err := json.UnmarshalFull(r.Body, &params); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Println("/api/v1/apps: params:", params)

	app := &Application{
		Name:         params.ClientName,
		Website:      params.Website,
		ClientID:     uuid.New().String(),
		ClientSecret: uuid.New().String(),
		RedirectURI:  params.RedirectURIs,
		VapidKey:     "BCk-QqERU0q-CfYZjcuB6lnyyOYfJ2AifKqfeGIm7Z-HiTU5T9eTG5GxVA0_OH5mMlI4UkkDTpaZwozy0TzdZ2M=",
	}
	if err := svc.db.Create(app).Error; err != nil {
		http.Error(w, "failed to create application", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, app)
}

func (svc *Service) InstanceFetch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, &Instance{
		URI:              "https://cheney.net/",
		Title:            "Casa del Cheese",
		ShortDescription: "🧀",
		Email:            "dave@cheney.net",
		Version:          "0.1.2",
		Languages:        []string{"en"},
	})
}

func (svc *Service) InstancePeers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, []string{})
}

func (svc *Service) OAuthAuthorize(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		svc.authorizeGet(w, r)
	case "POST":
		svc.authorizePost(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (svc *Service) authorizeGet(w http.ResponseWriter, r *http.Request) {
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	fmt.Println("/oauth/authorize(get): query:", r.URL.Query(), "form:", r.Form)
	if clientID == "" {
		http.Error(w, "client_id is required", http.StatusBadRequest)
		return
	}
	if redirectURI == "" {
		http.Error(w, "redirect_uri is required", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	io.WriteString(w, `
		<!DOCTYPE html>
		<html>
		<head>
		<meta charset="utf-8">
		<title>Authorize</title>
		</head>
		<body>
		<form method="POST" action="/oauth/authorize">
		<p><label>Email</label><input type="text" name="email"></p>
		<p><label>Password</label><input type="password" name="password"></p>
		<input type="hidden" name="client_id" value="`+clientID+`">
		<input type="hidden" name="redirect_uri" value="`+redirectURI+`">
		<input type="hidden" name="response_type" value="code"> 
		<p><input type="submit" value="I solemnly swear that I am up to no good"></p>
		</form>
		</body>
		</html>
	`)
}

func (svc *Service) authorizePost(w http.ResponseWriter, r *http.Request) {
	email := r.PostFormValue("email")
	password := r.PostFormValue("password")
	redirectURI := r.PostFormValue("redirect_uri")
	clientID := r.PostFormValue("client_id")
	fmt.Println("/oauth/authorize(post): query:", r.URL.Query(), "form:", r.Form)
	app, err := svc.applications().findByClientID(clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := svc.users().findByEmail(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !user.comparePassword(password) {
		http.Error(w, "invalid password", http.StatusUnauthorized)
		return
	}

	token := &Token{
		UserID:            user.ID,
		ApplicationID:     app.ID,
		AccessToken:       uuid.New().String(),
		TokenType:         "bearer",
		Scope:             "read write follow push",
		AuthorizationCode: uuid.New().String(),
	}
	if err := svc.tokens().create(token); err != nil {
		log.Println("saveToken:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", redirectURI+"?code="+token.AuthorizationCode)
	w.WriteHeader(302)
}

func (svc *Service) OAuthToken(w http.ResponseWriter, r *http.Request) {

	var body = map[string]string{}
	if err := json.UnmarshalFull(r.Body, &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Println("/oauth/token: query:", r.URL.Query(), "form:", r.Form, "headers:", r.Header)
	fmt.Println("body:", body)
	token, err := svc.tokens().findByAuthorizationCode(body["code"])
	if err != nil {
		log.Println("findTokenByAuthorizationCode:", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	app, err := svc.applications().findByClientID(body["client_id"])
	if err != nil {
		log.Println("findApplicationByClientID:", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if token.ApplicationID != app.ID {
		log.Println("client_id mismatch", token.ApplicationID, app.ID)
		http.Error(w, "invalid client_id", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, token)
}

func (svc *Service) AccountsVerify(w http.ResponseWriter, r *http.Request) {
	bearer := r.Header.Get("Authorization")
	accessToken := strings.TrimPrefix(bearer, "Bearer ")
	token, err := svc.tokens().findByAccessToken(accessToken)
	if err != nil {
		log.Println("findTokenByAccessToken:", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	user, err := svc.users().findByID(token.UserID)
	if err != nil {
		log.Println("findUserByID:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	account, err := svc.accounts().findByUserID(user.ID)
	if err != nil {
		log.Println("findAccountByUserID:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, account)
}

func (svc *Service) WellknownWebfinger(w http.ResponseWriter, r *http.Request) {
	account, err := svc.accounts().findByAcct(r.URL.Query().Get("resource"))
	if err != nil {
		log.Println("findAccountByAcct:", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	webfinger := fmt.Sprintf("https://%s/@%s", account.Domain, account.Username)
	self := fmt.Sprintf("https://%s/users/%s", account.Domain, account.Username)
	w.Header().Set("Content-Type", "application/jrd+json")
	json.MarshalFull(w, map[string]any{
		"subject": account.Acct,
		"aliases": []string{webfinger, self},
		"links": []map[string]any{
			{
				"rel":  "http://webfinger.net/rel/profile-page",
				"type": "text/html",
				"href": webfinger,
			},
			{
				"rel":  "self",
				"type": "application/activity+json",
				"href": self,
			},
			{
				"rel":      "http://ostatus.org/schema/1.0/subscribe",
				"template": fmt.Sprintf("https://%s/authorize_interaction?uri={uri}", account.Domain),
			},
		},
	})
}

func (svc *Service) TimelinesHome(w http.ResponseWriter, r *http.Request) {
	bearer := r.Header.Get("Authorization")
	accessToken := strings.TrimPrefix(bearer, "Bearer ")
	token, err := svc.tokens().findByAccessToken(accessToken)
	if err != nil {
		log.Println("findTokenByAccessToken:", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	user, err := svc.users().findByID(token.UserID)
	if err != nil {
		log.Println("findUserByID:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = svc.accounts().findByUserID(user.ID)
	if err != nil {
		log.Println("findAccountByUserID:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type status struct {
		ID                 int       `json:"id,string"`
		CreatedAt          time.Time `json:"created_at,format:'2006-01-02T15:04:05.006Z"`
		InReplyTo          *int      `json:"in_reply_to_id,string"`
		InReplyToAccountID *int      `json:"in_reply_to_account_id,string"`
		Sensitive          bool      `json:"sensitive"`
		SpoilerText        string    `json:"spoiler_text"`
		Visibility         string    `json:"visibility"`
		Language           string    `json:"language"`
		URI                string    `json:"uri"`
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `[{
		"id": "1",
		"created_at": "2016-03-16T14:44:31.580Z",
		"in_reply_to_id": null,
		"in_reply_to_account_id": null,
		"sensitive": false,
		"spoiler_text": "",
		"visibility": "public",
		"language": "en",
		"uri": "https://mastodon.social/users/Gargron/statuses/1",
		"url": "https://mastodon.social/@Gargron/1",
		"replies_count": 7,
		"reblogs_count": 98,
		"favourites_count": 112,
		"favourited": false,
		"reblogged": false,
		"muted": false,
		"bookmarked": false,
		"content": "<p>Hello world</p>",
		"reblog": null,
		"application": null,
		"account": {
		  "id": "1",
		  "username": "Gargron",
		  "acct": "Gargron",
		  "display_name": "Eugen",
		  "locked": false,
		  "bot": false,
		  "created_at": "2016-03-16T14:34:26.392Z",
		  "note": "<p>Developer of Mastodon and administrator of mastodon.social. I post service announcements, development updates, and personal stuff.</p>",
		  "url": "https://mastodon.social/@Gargron",
		  "avatar": "https://files.mastodon.social/accounts/avatars/000/000/001/original/d96d39a0abb45b92.jpg",
		  "avatar_static": "https://files.mastodon.social/accounts/avatars/000/000/001/original/d96d39a0abb45b92.jpg",
		  "header": "https://files.mastodon.social/accounts/headers/000/000/001/original/c91b871f294ea63e.png",
		  "header_static": "https://files.mastodon.social/accounts/headers/000/000/001/original/c91b871f294ea63e.png",
		  "followers_count": 320472,
		  "following_count": 453,
		  "statuses_count": 61163,
		  "last_status_at": "2019-12-05T03:03:02.595Z",
		  "emojis": [],
		  "fields": [
			{
			  "name": "Patreon",
			  "value": "<a href=\"https://www.patreon.com/mastodon\" rel=\"me nofollow noopener noreferrer\" target=\"_blank\"><span class=\"invisible\">https://www.</span><span class=\"\">patreon.com/mastodon</span><span class=\"invisible\"></span></a>",
			  "verified_at": null
			},
			{
			  "name": "Homepage",
			  "value": "<a href=\"https://zeonfederated.com\" rel=\"me nofollow noopener noreferrer\" target=\"_blank\"><span class=\"invisible\">https://</span><span class=\"\">zeonfederated.com</span><span class=\"invisible\"></span></a>",
			  "verified_at": "2019-07-15T18:29:57.191+00:00"
			}
		  ]
		},
		"media_attachments": [],
		"mentions": [],
		"tags": [],
		"emojis": [],
		"card": null,
		"poll": null
	  }]`)
}

func (svc *Service) StatusesCreate(w http.ResponseWriter, r *http.Request) {
	bearer := r.Header.Get("Authorization")
	accessToken := strings.TrimPrefix(bearer, "Bearer ")
	token, err := svc.tokens().findByAccessToken(accessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	user, err := svc.users().findByID(token.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	_, err = svc.accounts().findByUserID(user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	req, err := httputil.DumpRequest(r, true)
	fmt.Println(string(req), err)
}
