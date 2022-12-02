package m

import (
	"io"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type OAuth struct {
	db       *gorm.DB
	instance *Instance
}

func NewOAuth(db *gorm.DB, instance *Instance) *OAuth {
	return &OAuth{
		db:       db,
		instance: instance,
	}
}

func (o *OAuth) Authorize(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		o.authorizeGet(w, r)
	case "POST":
		o.authorizePost(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (o *OAuth) authorizeGet(w http.ResponseWriter, r *http.Request) {
	buf, _ := httputil.DumpRequest(r, false)
	log.Println("authorizeGet", string(buf))
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
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

func (o *OAuth) authorizePost(w http.ResponseWriter, r *http.Request) {
	buf, _ := httputil.DumpRequest(r, false)
	log.Println("authorizePost", string(buf))
	email := r.FormValue("email")
	password := r.PostFormValue("password")
	redirectURI := r.PostFormValue("redirect_uri")
	clientID := r.PostFormValue("client_id")

	var app Application
	if err := o.db.Where("client_id = ?", clientID).First(&app).Error; err != nil {
		http.Error(w, "invalid client_id", http.StatusBadRequest)
		return
	}

	var account Account
	if err := o.db.Where("email = ?", email).First(&account).Error; err != nil {
		http.Error(w, "invalid username", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword(account.EncryptedPassword, []byte(password)); err != nil {
		http.Error(w, "invalid password", http.StatusUnauthorized)
		return
	}

	token := &Token{
		ApplicationID:     app.ID,
		AccountID:         account.ID,
		AccessToken:       uuid.New().String(),
		TokenType:         "bearer",
		Scope:             "read write follow push",
		AuthorizationCode: uuid.New().String(),
	}
	if err := o.db.Create(token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", redirectURI+"?code="+token.AuthorizationCode)
	w.WriteHeader(302)
}

func (o *OAuth) Token(w http.ResponseWriter, r *http.Request) {
	var params struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		GrantType    string `json:"grant_type"`
		Code         string `json:"code"`
		RedirectURI  string `json:"redirect_uri"`
	}
	switch r.Header.Get("Content-Type") {
	case "application/json":
		if err := json.UnmarshalFull(r.Body, &params); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		params.ClientID = r.FormValue("client_id")
		params.ClientSecret = r.FormValue("client_secret")
		params.GrantType = r.FormValue("grant_type")
		params.Code = r.FormValue("code")
		params.RedirectURI = r.FormValue("redirect_uri")
	}
	var token Token
	if err := o.db.Where("authorization_code = ?", params.Code).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var app Application
	if err := o.db.Where("client_id = ?", params.ClientID).First(&app).Error; err != nil {
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

func (o *OAuth) Revoke(w http.ResponseWriter, r *http.Request) {
	var params struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Token        string `json:"token"`
	}
	var token Token
	switch r.Header.Get("Content-Type") {
	case "application/json":
		if err := json.UnmarshalFull(r.Body, &params); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		params.ClientID = r.FormValue("client_id")
		params.ClientSecret = r.FormValue("client_secret")
		params.Token = r.FormValue("token")
	}
	if err := o.db.Where("access_token = ?", params.Token).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := o.db.Delete(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(200)
}
