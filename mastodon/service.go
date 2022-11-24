package mastodon

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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

func (svc *Service) tokens() *tokens {
	return &tokens{db: svc.db}
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
	json.MarshalFull(w, map[string]any{
		"id":            strconv.Itoa(int(app.ID)),
		"name":          app.Name,
		"website":       app.Website,
		"redirect_uri":  app.RedirectURI,
		"client_id":     app.ClientID,
		"client_secret": app.ClientSecret,
		"vapid_key":     app.VapidKey,
	})
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

	var app Application
	if err := svc.db.Where("client_id = ?", clientID).First(&app).Error; err != nil {
		http.Error(w, "invalid client_id", http.StatusBadRequest)
		return
	}

	var user User
	if err := svc.db.Preload("Account").Where("email = ?", email).First(&user).Error; err != nil {
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
		AccountID:         user.Account.ID,
		AccessToken:       uuid.New().String(),
		TokenType:         "bearer",
		Scope:             "read write follow push",
		AuthorizationCode: uuid.New().String(),
	}
	if err := svc.db.Create(token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", redirectURI+"?code="+token.AuthorizationCode)
	w.WriteHeader(302)
}

func (svc *Service) OAuthToken(w http.ResponseWriter, r *http.Request) {
	var params struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		GrantType    string `json:"grant_type"`
		Code         string `json:"code"`
		RedirectURI  string `json:"redirect_uri"`
	}

	if err := json.UnmarshalFull(r.Body, &params); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	token, err := svc.tokens().findByAuthorizationCode(params.Code)
	if err != nil {
		log.Println("findTokenByAuthorizationCode:", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var app Application
	if err := svc.db.Where("client_id = ?", params.ClientID).First(&app).Error; err != nil {
		http.Error(w, "invalid client_id", http.StatusBadRequest)
		return
	}

	if token.ApplicationID != app.ID {
		log.Println("client_id mismatch", token.ApplicationID, app.ID)
		http.Error(w, "invalid client_id", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, map[string]any{
		"access_token": token.AccessToken,
		"token_type":   token.TokenType,
		"scope":        token.Scope,
		"created_at":   token.CreatedAt.Unix(),
	})
}

func (svc *Service) OAuthRevoke(w http.ResponseWriter, r *http.Request) {
	bearer := r.Header.Get("Authorization")
	accessToken := strings.TrimPrefix(bearer, "Bearer ")
	var token Token
	if err := svc.db.Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := svc.db.Delete(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(204)
}

func (svc *Service) AccountsVerify(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	var token Token
	if err := svc.db.Preload("Account").Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, serialiseAccount(&token.Account))
}

func (svc *Service) AccountsFetch(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token Token
	if err := svc.db.Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var account Account
	if err := svc.db.Where("id = ?", id).First(&account).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, serialiseAccount(&account))
}

func serialiseAccount(a *Account) map[string]any {
	fmt.Printf("%+v\n", a)
	return map[string]any{
		"id":              strconv.Itoa(int(a.ID)),
		"username":        a.Username,
		"acct":            a.Acct,
		"display_name":    a.DisplayName,
		"locked":          a.Locked,
		"bot":             a.Bot,
		"created_at":      a.CreatedAt.Format("2006-01-02T15:04:05.006Z"),
		"note":            a.Note,
		"url":             a.URL,
		"avatar":          a.Avatar,
		"avatar_static":   a.Avatar,
		"header":          a.Header,
		"header_static":   a.Header,
		"followers_count": a.FollowersCount,
		"following_count": a.FollowingCount,
		"statuses_count":  a.StatusesCount,
		"last_status_at":  a.LastStatusAt.Format("2006-01-02T15:04:05.006Z"),
		"emojis":          []map[string]any{},
		"fields":          []map[string]any{},
	}
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
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token Token
	if err := svc.db.Preload("Account").Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, []map[string]any{{
		"id":                     "3",
		"created_at":             time.Now().Format("2006-01-02T15:04:05.006Z"),
		"in_reply_to_id":         nil,
		"in_reply_to_account_id": nil,
		"sensitive":              false,
		"spoiler_text":           "",
		"visibility":             "public",
		"language":               "en",
		"uri":                    "https://cheney.net/users/dave/statuses/3",
		"url":                    "https://cheney.net/@dave/3",
		"replies_count":          7,
		"reblogs_count":          98,
		"favourites_count":       112,
		"favourited":             false,
		"reblogged":              false,
		"muted":                  false,
		"bookmarked":             false,
		"content":                "<p>Hello world</p>",
		"reblog":                 nil,
		"application":            nil,
		"account":                serialiseAccount(&token.Account),
		"media_attachments":      []map[string]any{},
		"mentions":               []map[string]any{},
		"tags":                   []map[string]any{},
		"emojis":                 []map[string]any{},
		"card":                   nil,
		"poll":                   nil,
	}})
}

func (svc *Service) StatusesCreate(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var token Token
	if err := svc.db.Preload("Account").Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	req, err := httputil.DumpRequest(r, true)
	fmt.Println(string(req), err)
}
