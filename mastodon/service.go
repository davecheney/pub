package mastodon

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

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

func (svc *Service) users() *users {
	return &users{db: svc.db}
}

func (svc *Service) tokens() *tokens {
	return &tokens{db: svc.db}
}

func (svc *Service) applications() *applications {
	return &applications{db: svc.db}
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
	if err := svc.applications().create(app); err != nil {
		log.Println("saveApplication:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

func (svc *Service) InstanceFetch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&Instance{
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
	json.NewEncoder(w).Encode([]string{})
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

	app, err := svc.applications().findByClientID(clientID)
	if err != nil {
		log.Println("findApplicationByClientID:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := svc.users().findByEmail(email)
	if err != nil {
		log.Println("findUserByEmail:", err)
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
	code := r.FormValue("code")
	clientId := r.FormValue("client_id")
	fmt.Println(r.URL.Query())
	fmt.Println("OAuthToken", code, clientId)
	token, err := svc.tokens().findByAuthorizationCode(code)
	if err != nil {
		log.Println("findTokenByAuthorizationCode:", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	app, err := svc.applications().findByClientID(clientId)
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

func (svc *Service) WellknownWebfinger(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	if resource != "acct:dave@cheney.net" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/jrd+json")
	json.NewEncoder(w).Encode(map[string]any{
		"subject": resource,
		"links": []map[string]any{
			{
				"rel":  "http://webfinger.net/rel/profile-page",
				"type": "text/html",
				"href": "https://cheney.net/dave",
			},
			{
				"rel":  "self",
				"type": "application/activity+json",
				"href": "https://cheney.net/users/dave",
			},
			{
				"rel":      "http://ostatus.org/schema/1.0/subscribe",
				"template": "https://cheney.net/authorize_interaction?uri={uri}",
			},
		},
	})
}

func (svc *Service) TimelinesHome(w http.ResponseWriter, r *http.Request) {
	since, _ := strconv.ParseInt(r.FormValue("since_id"), 10, 64)
	limit, _ := strconv.ParseInt(r.FormValue("limit"), 10, 64)
	rows, err := svc.db.Queryx("SELECT id, activity FROM activitypub_inbox WHERE activity_type=? AND object_type=? AND id > ? ORDER BY created_at DESC LIMIT ?", "Create", "Note", since, limit)

	var statuses []Status
	for rows.Next() {
		var entry string
		var id int
		if err = rows.Scan(&id, &entry); err != nil {
			break
		}
		var activity map[string]any
		json.NewDecoder(strings.NewReader(entry)).Decode(&activity)
		object, _ := activity["object"].(map[string]interface{})
		statuses = append(statuses, Status{
			Id:         strconv.Itoa(id),
			Uri:        object["atomUri"].(string),
			CreatedAt:  object["published"].(string),
			Content:    object["content"].(string),
			Visibility: "public",
		})
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(statuses) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}
