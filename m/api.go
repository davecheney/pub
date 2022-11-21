package m

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/davecheney/m/activitypub"
	"github.com/davecheney/m/mastodon"
	"github.com/go-fed/httpsig"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"

	_ "embed"
)

//go:embed public.pem
var publicKey string

func New(db *sqlx.DB) http.Handler {
	r := mux.NewRouter()

	v1 := r.PathPrefix("/api/v1").Subrouter()
	api := &api{
		db: db,
	}

	v1.HandleFunc("/apps", api.appsCreate).Methods("POST")
	v1.HandleFunc("/instance", api.instanceFetch).Methods("GET")
	v1.HandleFunc("/instance/peers", api.instancePeers).Methods("GET")
	v1.HandleFunc("/accounts/verify_credentials", api.accountsVerify).Methods("GET")
	v1.HandleFunc("/timelines/home", api.timelinesHome).Methods("GET")

	oauth := r.PathPrefix("/oauth").Subrouter()
	handler := &OAuth{}
	oauth.HandleFunc("/authorize/", handler.Authorize).Methods("GET")
	oauth.HandleFunc("/authorize", handler.Authorize).Methods("GET")
	oauth.HandleFunc("/token", handler.Token).Methods("POST")

	wellknown := r.PathPrefix("/.well-known").Subrouter()
	wellknown.HandleFunc("/webfinger", api.wellknownWebfinger).Methods("GET")

	svc := &activitypub.Service{
		StoreActivity: api.storeActivity,
	}

	inbox := r.Path("/inbox").Subrouter()
	inbox.Use(api.validateSignature())
	inbox.HandleFunc("", svc.Inbox).Methods("POST")

	users := r.PathPrefix("/users").Subrouter()
	users.HandleFunc("/{username}", api.usersShow).Methods("GET")
	users.HandleFunc("/{username}/inbox", svc.Inbox).Methods("POST")

	r.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://dave.cheney.net/", http.StatusFound)
	})

	return r
}

type api struct {
	db *sqlx.DB
}

func (a *api) storeActivity(activity map[string]interface{}) error {
	b, err := json.Marshal(activity)
	if err != nil {
		return err
	}
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	object, _ := activity["object"].(map[string]interface{})
	objectType, _ := object["type"].(string)
	if _, err := tx.Exec("INSERT INTO activitypub_inbox (activity_type, object_type, activity) VALUES (?,?,?)", activity["type"], objectType, b); err != nil {
		log.Println("storeActivity:", err)
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func query[K comparable, V any](m map[K]V, k K) V {
	v, _ := m[k]
	return v
}

func (a *api) validateSignature() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			verifier, err := httpsig.NewVerifier(r)
			if err != nil {
				log.Println("validateSignature:", err)
			}
			log.Println("keyId:", verifier.KeyId())
			pubKey, err := a.GetKey(verifier.KeyId())
			if err != nil {
				log.Println("getkey:", err)
			}
			if err := verifier.Verify(pubKey, httpsig.RSA_SHA256); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (a *api) GetKey(id string) (crypto.PublicKey, error) {
	actor_id := trimKeyId(id)
	if actor, err := a.findActorById(actor_id); err == nil {
		return pemToPublicKey(actor["publicKey"].(map[string]interface{})["publicKeyPem"].(string))
	} else {
		log.Println("findActorById:", err)
	}

	actor, err := fetchActor(actor_id)
	if err != nil {
		return nil, err
	}
	if err := a.saveActor(actor); err != nil {
		return nil, err
	}
	return pemToPublicKey(actor["publicKey"].(map[string]interface{})["publicKeyPem"].(string))
}

// fetchActor fetches an actor from the remote server.
func fetchActor(id string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", id, nil)
	if err != nil {
		return nil, fmt.Errorf("fetchActor: newrequest: %w", err)
	}
	req.Header.Set("Accept", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetchActor: do: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetchActor: status: %d", resp.StatusCode)
	}
	var v map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("fetchActor: jsondecode: %w", err)
	}
	return v, nil
}

// pemToPublicKey converts a PEM encoded public key to a crypto.PublicKey.
func pemToPublicKey(pemEncoded string) (crypto.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemEncoded))
	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("pemToPublicKey: invalid pem type: %s", block.Type)
	}
	var publicKey interface{}
	var err error
	if publicKey, err = x509.ParsePKIXPublicKey(block.Bytes); err != nil {
		return nil, fmt.Errorf("pemToPublicKey: parsepkixpublickey: %w", err)
	}
	return publicKey, nil
}

// trimKeyId removes the #main-key suffix from the key id.
func trimKeyId(id string) string {
	if i := strings.Index(id, "#"); i != -1 {
		return id[:i]
	}
	return id
}

func (a *api) findActorById(id string) (map[string]any, error) {
	var b []byte
	err := a.db.QueryRowx("SELECT object FROM activitypub_actors WHERE actor_id = ? ORDER BY created_at desc LIMIT 1", id).Scan(&b)
	if err != nil {
		return nil, err
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func (a *api) saveActor(actor map[string]interface{}) error {
	b, err := json.Marshal(actor)
	if err != nil {
		return err
	}
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("INSERT INTO activitypub_actors (actor_id, type, object, publickey) VALUES (?,?,?,?)", actor["id"], actor["type"], b, actor["publicKey"].(map[string]interface{})["publicKeyPem"].(string)); err != nil {
		return err
	}
	return tx.Commit()
}

func (a *api) usersShow(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	actor_id := fmt.Sprintf("https://cheney.net/users/%s", username)
	actor, err := a.findActorById(actor_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/activity+json")
	json.NewEncoder(w).Encode(actor)
}

func (a *api) accountsVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{
		"id": "14715",
		"username": "trwnh",
		"acct": "trwnh",
		"display_name": "infinite love ⴳ",
		"locked": false,
		"bot": false,
		"created_at": "2016-11-24T10:02:12.085Z",
		"note": "<p>i have approximate knowledge of many things. perpetual student. (nb/ace/they)</p><p>xmpp/email: a@trwnh.com<br /><a href=\"https://trwnh.com\" rel=\"nofollow noopener noreferrer\" target=\"_blank\"><span class=\"invisible\">https://</span><span class=\"\">trwnh.com</span><span class=\"invisible\"></span></a><br />help me live: <a href=\"https://liberapay.com/at\" rel=\"nofollow noopener noreferrer\" target=\"_blank\"><span class=\"invisible\">https://</span><span class=\"\">liberapay.com/at</span><span class=\"invisible\"></span></a> or <a href=\"https://paypal.me/trwnh\" rel=\"nofollow noopener noreferrer\" target=\"_blank\"><span class=\"invisible\">https://</span><span class=\"\">paypal.me/trwnh</span><span class=\"invisible\"></span></a></p><p>- my triggers are moths and glitter<br />- i have all notifs except mentions turned off, so please interact if you wanna be friends! i literally will not notice otherwise<br />- dm me if i did something wrong, so i can improve<br />- purest person on fedi, do not lewd in my presence<br />- #1 ami cole fan account</p><p>:fatyoshi:</p>",
		"url": "https://mastodon.social/@trwnh",
		"avatar": "https://files.mastodon.social/accounts/avatars/000/014/715/original/34aa222f4ae2e0a9.png",
		"avatar_static": "https://files.mastodon.social/accounts/avatars/000/014/715/original/34aa222f4ae2e0a9.png",
		"header": "https://files.mastodon.social/accounts/headers/000/014/715/original/5c6fc24edb3bb873.jpg",
		"header_static": "https://files.mastodon.social/accounts/headers/000/014/715/original/5c6fc24edb3bb873.jpg",
		"followers_count": 821,
		"following_count": 178,
		"statuses_count": 33120,
		"last_status_at": "2019-11-24T15:49:42.251Z",
		"source": {
		  "privacy": "public",
		  "sensitive": false,
		  "language": "",
		  "note": "i have approximate knowledge of many things. perpetual student. (nb/ace/they)\r\n\r\nxmpp/email: a@trwnh.com\r\nhttps://trwnh.com\r\nhelp me live: https://liberapay.com/at or https://paypal.me/trwnh\r\n\r\n- my triggers are moths and glitter\r\n- i have all notifs except mentions turned off, so please interact if you wanna be friends! i literally will not notice otherwise\r\n- dm me if i did something wrong, so i can improve\r\n- purest person on fedi, do not lewd in my presence\r\n- #1 ami cole fan account\r\n\r\n:fatyoshi:",
		  "fields": [
			{
			  "name": "Website",
			  "value": "https://trwnh.com",
			  "verified_at": "2019-08-29T04:14:55.571+00:00"
			},
			{
			  "name": "Sponsor",
			  "value": "https://liberapay.com/at",
			  "verified_at": "2019-11-15T10:06:15.557+00:00"
			},
			{
			  "name": "Fan of:",
			  "value": "Punk-rock and post-hardcore (Circa Survive, letlive., La Dispute, THE FEVER 333)Manga (Yu-Gi-Oh!, One Piece, JoJo's Bizarre Adventure, Death Note, Shaman King)Platformers and RPGs (Banjo-Kazooie, Boktai, Final Fantasy Crystal Chronicles)",
			  "verified_at": null
			},
			{
			  "name": "Main topics:",
			  "value": "systemic analysis, design patterns, anticapitalism, info/tech freedom, theory and philosophy, and otherwise being a genuine and decent wholesome poster. i'm just here to hang out and talk to cool people!",
			  "verified_at": null
			}
		  ],
		  "follow_requests_count": 0
		},
		"emojis": [
		  {
			"shortcode": "fatyoshi",
			"url": "https://files.mastodon.social/custom_emojis/images/000/023/920/original/e57ecb623faa0dc9.png",
			"static_url": "https://files.mastodon.social/custom_emojis/images/000/023/920/static/e57ecb623faa0dc9.png",
			"visible_in_picker": true
		  }
		],
		"fields": [
		  {
			"name": "Website",
			"value": "<a href=\"https://trwnh.com\" rel=\"me nofollow noopener noreferrer\" target=\"_blank\"><span class=\"invisible\">https://</span><span class=\"\">trwnh.com</span><span class=\"invisible\"></span></a>",
			"verified_at": "2019-08-29T04:14:55.571+00:00"
		  },
		  {
			"name": "Sponsor",
			"value": "<a href=\"https://liberapay.com/at\" rel=\"me nofollow noopener noreferrer\" target=\"_blank\"><span class=\"invisible\">https://</span><span class=\"\">liberapay.com/at</span><span class=\"invisible\"></span></a>",
			"verified_at": "2019-11-15T10:06:15.557+00:00"
		  },
		  {
			"name": "Fan of:",
			"value": "Punk-rock and post-hardcore (Circa Survive, letlive., La Dispute, THE FEVER 333)Manga (Yu-Gi-Oh!, One Piece, JoJo&apos;s Bizarre Adventure, Death Note, Shaman King)Platformers and RPGs (Banjo-Kazooie, Boktai, Final Fantasy Crystal Chronicles)",
			"verified_at": null
		  },
		  {
			"name": "Main topics:",
			"value": "systemic analysis, design patterns, anticapitalism, info/tech freedom, theory and philosophy, and otherwise being a genuine and decent wholesome poster. i&apos;m just here to hang out and talk to cool people!",
			"verified_at": null
		  }
		]
	  }`)
}

func (a *api) appsCreate(w http.ResponseWriter, r *http.Request) {
	redirectURIs := r.FormValue("redirect_uris")
	resp := map[string]any{
		"id":            "563419",
		"name":          "test app",
		"website":       nil,
		"redirect_uri":  redirectURIs,
		"client_id":     uuid.New().String(),
		"client_secret": uuid.New().String(),
		"vapid_key":     "BCk-QqERU0q-CfYZjcuB6lnyyOYfJ2AifKqfeGIm7Z-HiTU5T9eTG5GxVA0_OH5mMlI4UkkDTpaZwozy0TzdZ2M=",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *api) instanceFetch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&mastodon.Instance{
		URI:              "https://cheney.net/",
		Title:            "Casa del Cheese",
		ShortDescription: "🧀",
		Email:            "dave@cheney.net",
		Version:          "0.1.2",
		Languages:        []string{"en"},
	})
}

func (a *api) instancePeers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]string{})
}

type OAuth struct {
}

func (o *OAuth) Authorize(w http.ResponseWriter, r *http.Request) {
	redirectURI := r.FormValue("redirect_uri")
	w.Header().Set("Location", redirectURI+"?code="+uuid.New().String())
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (o *OAuth) Token(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"access_token": uuid.New().String(),
		"token_type":   "Bearer",
		"scope":        "read write follow push",
		"created_at":   time.Now().Unix(),
	})
}

func (a *api) timelinesHome(w http.ResponseWriter, r *http.Request) {
	since, _ := strconv.ParseInt(r.FormValue("since_id"), 10, 64)
	limit, _ := strconv.ParseInt(r.FormValue("limit"), 10, 64)
	rows, err := a.db.Queryx("SELECT id, activity FROM activitypub_inbox WHERE activity_type=? AND object_type=? AND id > ? ORDER BY created_at DESC LIMIT ?", "Create", "Note", since, limit)

	var statuses []mastodon.Status
	for rows.Next() {
		var entry string
		var id int
		if err = rows.Scan(&id, &entry); err != nil {
			break
		}
		var activity map[string]any
		json.NewDecoder(strings.NewReader(entry)).Decode(&activity)
		object, _ := activity["object"].(map[string]interface{})
		statuses = append(statuses, mastodon.Status{
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

func (a *api) wellknownWebfinger(w http.ResponseWriter, r *http.Request) {
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

func dumpRequest(w io.Writer, r *http.Request) error {
	fmt.Fprintf(w, "%s %s %s\n", r.Method, r.URL, r.Proto)
	for k := range r.Header {
		fmt.Fprintf(w, "%s: %s\n", k, r.Header.Get(k))
	}
	fmt.Fprintln(w)
	_, err := io.Copy(w, r.Body)
	fmt.Fprintln(w)
	return err
}
