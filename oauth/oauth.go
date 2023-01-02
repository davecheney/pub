package oauth

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/davecheney/pub/internal/models"
	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func AuthorizeNew(w http.ResponseWriter, r *http.Request) {
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
		<p><label>Username</label><input type="text" name="username"></p>
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

func AuthorizeCreate(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.PostFormValue("password")
	redirectURI := r.PostFormValue("redirect_uri")
	clientID := r.PostFormValue("client_id")
	db, _ := r.Context().Value("DB").(*gorm.DB)

	var app models.Application
	if err := db.Where("client_id = ?", clientID).First(&app).Error; err != nil {
		fmt.Println("failed to find application", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var account models.Account
	if err := db.Joins("Actor").First(&account, "name = ? and domain = ?", username, r.Host).Error; err != nil {
		fmt.Println("failed to find account", err)
		http.Error(w, "invalid username", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword(account.EncryptedPassword, []byte(password)); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	token := &models.Token{
		AccessToken:       uuid.New().String(),
		AccountID:         account.ID,
		ApplicationID:     app.ID,
		TokenType:         "Bearer",
		Scope:             "read write follow push",
		AuthorizationCode: uuid.New().String(),
	}
	if err := db.Create(token).Error; err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if redirectURI == "" {
		redirectURI = app.RedirectURI
	}

	w.Header().Set("Location", redirectURI+"?code="+token.AuthorizationCode)
	w.WriteHeader(302)
}

func TokenCreate(w http.ResponseWriter, r *http.Request) {
	var params struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		GrantType    string `json:"grant_type"`
		Code         string `json:"code"`
		RedirectURI  string `json:"redirect_uri"`
	}
	switch strings.Split(r.Header.Get("Content-Type"), ";")[0] {
	case "application/x-www-form-urlencoded":
		params.ClientID = r.FormValue("client_id")
		params.ClientSecret = r.FormValue("client_secret")
		params.GrantType = r.FormValue("grant_type")
		params.Code = r.FormValue("code")
		params.RedirectURI = r.FormValue("redirect_uri")
	case "application/json":
		switch r.ContentLength {
		case 0:
			// god damnint Mammoth, why do you send empty body?
			params.ClientID = r.FormValue("client_id")
			params.ClientSecret = r.FormValue("client_secret")
			params.GrantType = r.FormValue("grant_type")
			params.Code = r.FormValue("code")
			params.RedirectURI = r.FormValue("redirect_uri")
		default:
			if err := json.UnmarshalFull(r.Body, &params); err != nil {
				buf, _ := httputil.DumpRequest(r, false)
				fmt.Println(string(buf))
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
	default:
		buf, _ := httputil.DumpRequest(r, true)
		fmt.Println(string(buf))
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}
	db, _ := r.Context().Value("DB").(*gorm.DB)
	var token models.Token
	if err := db.Where("authorization_code = ?", params.Code).First(&token).Error; err != nil {
		http.Error(w, fmt.Sprintf("token with code %s not found", params.Code), http.StatusUnauthorized)
		return
	}
	var app models.Application
	if err := db.Where("client_id = ?", params.ClientID).First(&app).Error; err != nil {
		fmt.Println("failed to find application", err)
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

func TokenDestroy(w http.ResponseWriter, r *http.Request) {
	var params struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Token        string `json:"token"`
	}
	switch strings.Split(r.Header.Get("Content-Type"), ";")[0] {
	case "application/x-www-form-urlencoded":
		params.ClientID = r.FormValue("client_id")
		params.ClientSecret = r.FormValue("client_secret")
		params.Token = r.FormValue("token")
	case "application/json":
		if err := json.UnmarshalFull(r.Body, &params); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}
	fmt.Println("params", params)
	db, _ := r.Context().Value("DB").(*gorm.DB)
	var token models.Token
	if err := db.Where("access_token = ?", params.Token).First(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := db.Delete(&token).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(200)
}
