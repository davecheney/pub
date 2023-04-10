package oauth

import (
	"errors"
	"fmt"
	"io"
	"log"

	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/davecheney/pub/activitypub"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func AuthorizeNew(env *activitypub.Env, w http.ResponseWriter, r *http.Request) error {
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	if clientID == "" {
		return httpx.Error(http.StatusBadRequest, fmt.Errorf("client_id is required"))
	}
	if redirectURI == "" {
		return httpx.Error(http.StatusBadRequest, fmt.Errorf("redirect_uri is required"))
	}

	var app models.Application
	if err := env.DB.Where("client_id = ?", clientID).First(&app).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.Error(http.StatusUnauthorized, fmt.Errorf("client_id not found"))
		}
		return err
	}

	w.Header().Set("Content-Type", "text/html")
	_, err := io.WriteString(w, `
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
	return err
}

func AuthorizeCreate(env *activitypub.Env, w http.ResponseWriter, r *http.Request) error {
	username := r.FormValue("username")
	password := r.PostFormValue("password")
	redirectURI := r.PostFormValue("redirect_uri")
	clientID := r.PostFormValue("client_id")

	var app models.Application
	if err := env.DB.Where("client_id = ?", clientID).First(&app).Error; err != nil {
		return httpx.Error(http.StatusBadRequest, fmt.Errorf("failed to find application: %v", err))
	}

	var account models.Account
	if err := env.DB.Joins("Actor").First(&account, "name = ? and domain = ?", username, r.Host).Error; err != nil {
		return httpx.Error(http.StatusUnauthorized, fmt.Errorf("invalid username"))
	}

	if err := bcrypt.CompareHashAndPassword(account.EncryptedPassword, []byte(password)); err != nil {
		return httpx.Error(http.StatusUnauthorized, fmt.Errorf("invalid password"))
	}

	token := &models.Token{
		AccessToken:       uuid.New().String(),
		AccountID:         account.ID,
		ApplicationID:     app.ID,
		TokenType:         models.TokenType("Bearer"),
		Scope:             "read write follow push",
		AuthorizationCode: uuid.New().String(),
	}
	if err := env.DB.Create(token).Error; err != nil {
		return err
	}

	if redirectURI == "" {
		redirectURI = app.RedirectURI
	}

	return httpx.Redirect(w, redirectURI+"?code="+token.AuthorizationCode)
}

func TokenCreate(env *activitypub.Env, w http.ResponseWriter, r *http.Request) error {
	var params struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		GrantType    string `json:"grant_type"`
		Code         string `json:"code"`
		RedirectURI  string `json:"redirect_uri"`
	}
	switch httpx.MediaType(r) {
	case "":
		// ice cubes, why you gotta do me like this?
		fallthrough
	case "multipart/form-data", "application/x-www-form-urlencoded":
		params.ClientID = r.FormValue("client_id")
		params.ClientSecret = r.FormValue("client_secret")
		params.GrantType = r.FormValue("grant_type")
		params.Code = r.FormValue("code")
		params.RedirectURI = r.FormValue("redirect_uri")
	case "application/json":
		switch r.ContentLength {
		case 0:
			// god damnit Mammoth, why do you send empty body?
			params.ClientID = r.FormValue("client_id")
			params.ClientSecret = r.FormValue("client_secret")
			params.GrantType = r.FormValue("grant_type")
			params.Code = r.FormValue("code")
			params.RedirectURI = r.FormValue("redirect_uri")
		default:
			if err := json.UnmarshalFull(r.Body, &params); err != nil {
				buf, _ := httputil.DumpRequest(r, false)
				fmt.Println(string(buf))
				return httpx.Error(http.StatusBadRequest, fmt.Errorf("failed to parse request body: %w", err))
			}
		}
	default:
		buf, _ := httputil.DumpRequest(r, true)
		fmt.Println(string(buf))
		return httpx.Error(http.StatusUnsupportedMediaType, fmt.Errorf("unsupported media type: %s", r.Header.Get("Content-Type")))
	}
	var token models.Token
	if err := env.DB.Where("authorization_code = ?", params.Code).First(&token).Error; err != nil {
		return httpx.Error(http.StatusUnauthorized, fmt.Errorf("token with code %s not found", params.Code))
	}
	var app models.Application
	if err := env.DB.Where("client_id = ?", params.ClientID).First(&app).Error; err != nil {
		return httpx.Error(http.StatusBadRequest, fmt.Errorf("failed to find application: %w", err))
	}

	if token.ApplicationID != app.ID {
		log.Println("client_id mismatch", token.ApplicationID, app.ID)
		return httpx.Error(http.StatusUnauthorized, fmt.Errorf("client_id mismatch"))
	}
	return to.JSON(w, map[string]any{
		"access_token": token.AccessToken,
		"token_type":   token.TokenType,
		"scope":        token.Scope,
		"created_at":   token.CreatedAt.Unix(),
	})
}

func TokenDestroy(env *activitypub.Env, w http.ResponseWriter, r *http.Request) error {
	var params struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Token        string `json:"token"`
	}
	switch strings.Split(r.Header.Get("Content-Type"), ";")[0] {
	case "application/x-www-form-urlencoded", "multipart/form-data":
		params.ClientID = r.FormValue("client_id")
		params.ClientSecret = r.FormValue("client_secret")
		params.Token = r.FormValue("token")
	case "application/json":
		if err := json.UnmarshalFull(r.Body, &params); err != nil {
			return httpx.Error(http.StatusBadRequest, fmt.Errorf("failed to parse request body: %w", err))
		}
	default:
		return httpx.Error(http.StatusUnsupportedMediaType, fmt.Errorf("unsupported media type"))
	}
	fmt.Println("params", params)
	var token models.Token
	if err := env.DB.Where("access_token = ?", params.Token).First(&token).Error; err != nil {
		return httpx.Error(http.StatusUnauthorized, fmt.Errorf("token not found"))
	}
	if err := env.DB.Delete(&token).Error; err != nil {
		return httpx.Error(http.StatusInternalServerError, fmt.Errorf("failed to delete token: %w", err))
	}
	w.WriteHeader(200)
	return nil
}
