package mastodon

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Service implements a Mastodon service.
type Service struct {
	db *sqlx.DB
}

// NewService returns a new instance of Service.
func NewService(db *sqlx.DB) *Service {
	return &Service{
		db: db,
	}
}

func (svc *Service) AppsCreate(w http.ResponseWriter, r *http.Request) {
	clientName := r.FormValue("client_name")
	redirectURIs := r.FormValue("redirect_uris")
	redirectURI := r.FormValue("redirect_uri")
	fmt.Println("AppsCreate", clientName, redirectURIs, redirectURI)
	app := &Application{
		Name:         clientName,
		ClientID:     uuid.New().String(),
		ClientSecret: uuid.New().String(),
		RedirectURI:  redirectURIs,
		VapidKey:     "BCk-QqERU0q-CfYZjcuB6lnyyOYfJ2AifKqfeGIm7Z-HiTU5T9eTG5GxVA0_OH5mMlI4UkkDTpaZwozy0TzdZ2M=",
	}
	if err := svc.saveApplication(app); err != nil {
		log.Println("saveApplication:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

func (svc *Service) saveApplication(app *Application) error {
	result, err := svc.db.NamedExec(`INSERT INTO applications (name, website, redirect_uri, client_id, client_secret, vapid_key) VALUES (:name, :website, :redirect_uri, :client_id, :client_secret, :vapid_key)`, app)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	app.ID = int(id)
	return nil
}

func (svc *Service) Authorize(w http.ResponseWriter, r *http.Request) {
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
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")
	fmt.Println("authorizeGet", redirectURI, clientID)
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
		<p><input type="submit" value="I solemnly swear that I am up to no good"></p>
		</form>
		</body>
		</html>
	`)
}

func (svc *Service) authorizePost(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")

	app, err := svc.findApplicationByClientID(clientID)
	if err != nil {
		log.Println("findApplicationByClientID:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := svc.findUserByEmail(email)
	if err != nil {
		log.Println("findUserByEmail:", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !user.comparePassword(password) {
		http.Error(w, "invalid password", http.StatusUnauthorized)
		return
	}

	token, err := svc.createToken(user, app)
	if err != nil {
		log.Println("createAccessToken:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", redirectURI+"?code="+token.AuthorizationCode)
	w.WriteHeader(302)
}

type User struct {
	ID                int       `db:"id"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
	Email             string    `db:"email"`
	EncryptedPassword []byte    `db:"encrypted_password"`
}

func (u *User) comparePassword(password string) bool {
	if err := bcrypt.CompareHashAndPassword(u.EncryptedPassword, []byte(password)); err != nil {
		return false
	}
	return true
}

func (svc *Service) findUserByEmail(email string) (*User, error) {
	user := &User{}
	err := svc.db.QueryRowx(`SELECT * FROM users WHERE email = ?`, email).StructScan(user)
	return user, err
}

func (svc *Service) findApplicationByClientID(clientID string) (*Application, error) {
	app := &Application{}
	err := svc.db.QueryRowx(`SELECT * FROM applications WHERE client_id = ?`, clientID).StructScan(app)
	return app, err
}

func (svc *Service) createToken(user *User, app *Application) (*Token, error) {
	token := &Token{
		UserID:            user.ID,
		ApplicationID:     app.ID,
		AccessToken:       uuid.New().String(),
		TokenType:         "bearer",
		Scope:             "read write follow push",
		AuthorizationCode: uuid.New().String(),
	}
	result, err := svc.db.NamedExec(`INSERT INTO tokens (access_token, token_type, scope, user_id, application_id, authorization_code) VALUES (:access_token, :token_type, :scope, :user_id, :application_id, :authorization_code)`, token)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	token.ID = int(id)
	return token, nil
}

func (svc *Service) OAuthToken(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	clientId := r.FormValue("client_id")
	fmt.Println("OAuthToken", code, clientId)
	token, err := svc.findTokenByAuthorizationCode(code)
	if err != nil {
		log.Println("findTokenByAuthorizationCode:", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	app, err := svc.findApplicationByClientID(clientId)
	if err != nil {
		log.Println("findApplicationByClientID:", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if token.ApplicationID != app.ID {
		http.Error(w, "invalid client_id", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(token)
}

func (svc *Service) findTokenByAuthorizationCode(code string) (*Token, error) {
	token := &Token{}
	err := svc.db.QueryRowx(`SELECT * FROM tokens WHERE authorization_code = ?`, code).StructScan(token)
	return token, err
}

func (svc *Service) AccountsVerify(w http.ResponseWriter, r *http.Request) {
	token, err := svc.findTokenByAccessToken(r.Header.Get("Authorization"))
	if err != nil {
		log.Println("findTokenByAccessToken:", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	user, err := svc.findUserByID(token.UserID)
	if err != nil {
		log.Println("findUserByID:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	account, err := svc.findAccountByUserID(user.ID)
	if err != nil {
		log.Println("findAccountByUserID:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)

}

func (svc *Service) findUserByID(id int) (*User, error) {
	user := &User{}
	err := svc.db.QueryRowx(`SELECT * FROM users WHERE id = ?`, id).StructScan(user)
	return user, err
}

func (svc *Service) findAccountByUserID(id int) (*Account, error) {
	account := &Account{}
	err := svc.db.QueryRowx(`SELECT * FROM accounts WHERE user_id = ?`, id).StructScan(account)
	if err != nil {
		return nil, err
	}
	account.URI = fmt.Sprintf("https://%s/users/%s", account.Domain, account.Username)
	return account, nil
}

func (svc *Service) findTokenByAccessToken(accessToken string) (*Token, error) {
	accessToken = strings.TrimPrefix(accessToken, "Bearer ")
	token := &Token{}
	err := svc.db.QueryRowx(`SELECT * FROM tokens WHERE access_token = ?`, accessToken).StructScan(token)
	return token, err
}
