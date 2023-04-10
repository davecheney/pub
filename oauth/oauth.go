package oauth

import (
	"errors"
	"fmt"
	"io"
	"log"

	"net/http"

	"github.com/davecheney/pub/activitypub"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func AuthorizeNew(env *activitypub.Env, w http.ResponseWriter, r *http.Request) error {
	var params struct {
		ResponseType string `json:"-" schema:"response_type"`
		ClientID     string `json:"-" schema:"client_id,required"`
		RedirectURI  string `json:"-" schema:"redirect_uri,required"`
		Scope        string `json:"-" schema:"scope"`
		ForceLogin   bool   `json:"-" schema:"force_login"` // elk.zone, ignored
		Lang         string `json:"-" schema:"lang"`        // elk.zone, ignored
	}
	if err := httpx.Params(r, &params); err != nil {
		return err
	}
	var app models.Application
	if err := env.DB.Where("client_id = ?", params.ClientID).First(&app).Error; err != nil {
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
		<input type="hidden" name="client_id" value="`+params.ClientID+`">
		<input type="hidden" name="redirect_uri" value="`+params.RedirectURI+`">
		<input type="hidden" name="response_type" value="code"> 
		<p><input type="submit" value="I solemnly swear that I am up to no good"></p>
		</form>
		</body>
		</html>
	`)
	return err
}

func AuthorizeCreate(env *activitypub.Env, w http.ResponseWriter, r *http.Request) error {
	var params struct {
		Username     string `json:"-" schema:"username,required"`
		Password     string `json:"-" schema:"password,required"`
		RedirectURI  string `json:"-" schema:"redirect_uri,required"`
		ClientID     string `json:"-" schema:"client_id,required"`
		ResponseType string `json:"-" schema:"response_type"` // ignored
	}
	if err := httpx.Params(r, &params); err != nil {
		return err
	}

	var app models.Application
	if err := env.DB.Where("client_id = ?", params.ClientID).First(&app).Error; err != nil {
		return httpx.Error(http.StatusBadRequest, fmt.Errorf("failed to find application: %v", err))
	}

	var account models.Account
	if err := env.DB.Joins("Actor").First(&account, "name = ? and domain = ?", params.Username, r.Host).Error; err != nil {
		return httpx.Error(http.StatusUnauthorized, fmt.Errorf("invalid username"))
	}

	if err := bcrypt.CompareHashAndPassword(account.EncryptedPassword, []byte(params.Password)); err != nil {
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

	if params.RedirectURI == "" {
		params.RedirectURI = app.RedirectURI
	}

	return httpx.Redirect(w, params.RedirectURI+"?code="+token.AuthorizationCode)
}

func TokenCreate(env *activitypub.Env, w http.ResponseWriter, r *http.Request) error {
	var params struct {
		ClientID     string `json:"client_id" schema:"client_id,required"`
		ClientSecret string `json:"client_secret" schema:"client_secret,required"`
		GrantType    string `json:"grant_type" schema:"grant_type,required"`
		Code         string `json:"code" schema:"code,required"`
		RedirectURI  string `json:"redirect_uri" schema:"redirect_uri,required"`
		Scope        string `json:"-" schema:"scope"` // ignored
	}
	if err := httpx.Params(r, &params); err != nil {
		return err
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
		ClientID     string `json:"client_id" schema:"client_id"`
		ClientSecret string `json:"client_secret" schema:"client_secret"`
		Token        string `json:"token" schema:"token"`
	}
	if err := httpx.Params(r, &params); err != nil {
		return err
	}
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
