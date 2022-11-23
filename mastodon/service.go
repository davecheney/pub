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

func (svc *Service) accounts() *accounts {
	return &accounts{db: svc.db}
}

func (svc *Service) applications() *applications {
	return &applications{db: svc.db}
}
func (svc *Service) tokens() *tokens {
	return &tokens{db: svc.db}
}

func (svc *Service) users() *users {
	return &users{db: svc.db}
}

func (svc *Service) AppsCreate(w http.ResponseWriter, r *http.Request) {
	clientName := r.FormValue("client_name")
	redirectURIs := r.FormValue("redirect_uris")
	fmt.Println("AppsCreate", r.Form)
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
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
	json.NewEncoder(w).Encode(token)
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
	json.NewEncoder(w).Encode(account)
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}
