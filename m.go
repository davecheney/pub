package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/davecheney/m/activitypub"
	"github.com/davecheney/m/mastodon"
	"github.com/go-fed/httpsig"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql"
)

type Context struct {
	Debug bool
}

type ServeCmd struct {
	Addr string `help:"address to listen"`
	DSN  string `help:"data source name"`
}

func (s *ServeCmd) Run(ctx *Context) error {
	db, err := sqlx.Connect("mysql", s.DSN+"?parseTime=true")
	if err != nil {
		return err
	}

	mastodon := mastodon.NewService(db)

	r := mux.NewRouter()

	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/apps", mastodon.AppsCreate).Methods("POST")
	v1.HandleFunc("/accounts/verify_credentials", mastodon.AccountsVerify).Methods("GET")

	v1.HandleFunc("/instance", mastodon.InstanceFetch).Methods("GET")
	v1.HandleFunc("/instance/peers", mastodon.InstancePeers).Methods("GET")

	v1.HandleFunc("/timelines/home", mastodon.TimelinesHome).Methods("GET")

	oauth := r.PathPrefix("/oauth").Subrouter()
	oauth.HandleFunc("/authorize/", mastodon.Authorize).Methods("GET", "POST")
	oauth.HandleFunc("/authorize", mastodon.Authorize).Methods("GET", "POST")
	oauth.HandleFunc("/token", mastodon.OAuthToken).Methods("POST")

	wellknown := r.PathPrefix("/.well-known").Subrouter()
	wellknown.HandleFunc("/webfinger", mastodon.WellknownWebfinger).Methods("GET")

	activitypub := activitypub.NewService(db)

	inbox := r.Path("/inbox").Subrouter()
	inbox.Use(activitypub.ValidateSignature())
	inbox.HandleFunc("", activitypub.Inbox).Methods("POST")

	users := r.PathPrefix("/users").Subrouter()
	users.HandleFunc("/{username}", activitypub.UsersShow).Methods("GET")
	users.HandleFunc("/{username}/inbox", activitypub.Inbox).Methods("POST")

	r.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://dave.cheney.net/", http.StatusFound)
	})

	svr := &http.Server{
		Addr:         s.Addr,
		Handler:      r,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return svr.ListenAndServe()
}

type FollowCmd struct {
	Object string `help:"object to follow"`
	Actor  string `help:"actor to follow with"`
}

func (f *FollowCmd) Run(ctx *Context) error {
	body, err := json.Marshal(map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       fmt.Sprintf("https://cheney.net/%s", uuid.New().String()),
		"type":     "Follow",
		"object":   f.Object,
		"actor":    f.Actor,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", f.Actor, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetchActor: status: %d", resp.StatusCode)
	}
	var actor map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&actor); err != nil {
		return fmt.Errorf("fetchActor: jsondecode: %w", err)
	}

	u, err := url.Parse(f.Object)
	if err != nil {
		return err
	}
	req, err = http.NewRequest("POST", fmt.Sprintf("%s://%s/inbox", u.Scheme, u.Host), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/activity+json")
	sign(req, body, actor["publicKey"].(map[string]interface{})["id"].(string))

	if ctx.Debug {
		fmt.Printf("%s %s %s\n", req.Method, req.URL, req.Proto)
		for k := range req.Header {
			fmt.Printf("%s: %s\n", k, req.Header.Get(k))
		}
		fmt.Println()

	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if ctx.Debug {
		fmt.Printf("%s %s\n", resp.Proto, resp.Status)
		for k := range resp.Header {
			fmt.Printf("%s: %s\n", k, resp.Header.Get(k))
		}
		fmt.Println()
		io.Copy(os.Stdout, resp.Body)
		fmt.Println()
	}
	return nil
}

func sign(r *http.Request, body []byte, pubKeyId string) {
	r.Header.Set("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")) // Date must be in GMT, not UTC 🤯
	priv, err := ioutil.ReadFile("private.pem")
	if err != nil {
		log.Fatal(err)
	}
	privPem, _ := pem.Decode(priv)
	var privPemBytes []byte
	if privPem.Type != "RSA PRIVATE KEY" {
		log.Fatal("expected RSA PRIVATE KEY")
	}

	privPemBytes = privPem.Bytes

	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKCS1PrivateKey(privPemBytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(privPemBytes); err != nil { // note this returns type `interface{}`
			log.Fatal(err)
		}
	}

	var privateKey *rsa.PrivateKey
	var ok bool
	privateKey, ok = parsedKey.(*rsa.PrivateKey)
	if !ok {
		log.Fatal("failed to parse RSA private key")
	}
	// The "Date" and "Digest" headers must already be set on r, as well as r.URL.
	headersToSign := []string{httpsig.RequestTarget, "date", "digest"}
	signer, _, err := httpsig.NewSigner(
		[]httpsig.Algorithm{httpsig.RSA_SHA256},
		httpsig.DigestSha256,
		headersToSign,
		httpsig.Signature,
		60,
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := signer.SignRequest(privateKey, pubKeyId, r, body); err != nil {
		log.Fatal(err)
	}
}

var cli struct {
	Debug bool `help:"Enable debug mode."`

	Serve  ServeCmd  `cmd:"" help:"Serve a local web server."`
	Follow FollowCmd `cmd:"" help:"Follow an object."`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}
