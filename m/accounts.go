package m

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-fed/httpsig"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
	InstanceID     uint
	Instance       *Instance
	Domain         string `gorm:"uniqueIndex:idx_domainusername;size:64"`
	Username       string `gorm:"uniqueIndex:idx_domainusername;size:64"`
	DisplayName    string `gorm:"size:64"`
	Local          bool
	LocalAccount   *LocalAccount `gorm:"foreignKey:AccountID"`
	Locked         bool
	Bot            bool
	Note           string
	Avatar         string
	AvatarStatic   string
	Header         string
	HeaderStatic   string
	FollowersCount int `gorm:"default:0;not null"`
	FollowingCount int `gorm:"default:0;not null"`
	StatusesCount  int `gorm:"default:0;not null"`
	LastStatusAt   time.Time
	PublicKey      []byte

	Lists         []AccountList
	Statuses      []Status
	Markers       []Marker
	Favourites    []Favourite
	Notifications []Notification
}

type LocalAccount struct {
	AccountID         uint   `gorm:"primarykey;autoIncrement:false"`
	Email             string `gorm:"size:64"`
	EncryptedPassword []byte // only used for local accounts
	PrivateKey        []byte // only used for local accounts
}

func (a *Account) AfterCreate(tx *gorm.DB) error {
	// update count of accounts on instance
	var instance Instance
	if err := tx.Where("domain = ?", a.Domain).First(&instance).Error; err != nil {
		return err
	}
	return instance.updateAccountsCount(tx)
}

func (a *Account) updateStatusesCount(tx *gorm.DB) error {
	var count int64
	if err := tx.Model(&Status{}).Where("account_id = ?", a.ID).Count(&count).Error; err != nil {
		return err
	}
	return tx.Model(a).Update("statuses_count", count).Error
}

func (a *Account) Acct() string {
	if a.Local {
		return a.Username
	}
	return a.Username + "@" + a.Domain
}

func (a *Account) URL() string {
	return fmt.Sprintf("https://%s/@%s", a.Domain, a.Username)
}

func (a *Account) PublicKeyID() string {
	return fmt.Sprintf("https://%s/users/%s#main-key", a.Domain, a.Username)
}

func (a *Account) serialize() map[string]any {
	return map[string]any{
		"id":              strconv.Itoa(int(a.ID)),
		"username":        a.Username,
		"acct":            a.Acct(),
		"display_name":    a.DisplayName,
		"locked":          a.Locked,
		"bot":             a.Bot,
		"discoverable":    true,
		"group":           false, // todo
		"created_at":      a.CreatedAt.Format("2006-01-02T15:04:05.006Z"),
		"note":            a.Note,
		"url":             a.URL(),
		"avatar":          stringOrDefault(a.Avatar, fmt.Sprintf("https://%s/avatar.png", a.Domain)),
		"avatar_static":   stringOrDefault(a.AvatarStatic, fmt.Sprintf("https://%s/avatar.png", a.Domain)),
		"header":          stringOrDefault(a.Header, fmt.Sprintf("https://%s/header.png", a.Domain)),
		"header_static":   stringOrDefault(a.HeaderStatic, fmt.Sprintf("https://%s/header.png", a.Domain)),
		"followers_count": a.FollowersCount,
		"following_count": a.FollowingCount,
		"statuses_count":  a.StatusesCount,
		"last_status_at":  a.LastStatusAt.Format("2006-01-02"),
		"noindex":         false, // todo
		"emojis":          []map[string]any{},
		"fields":          []map[string]any{},
	}
}

type Accounts struct {
	db      *gorm.DB
	service *Service
}

func (a *Accounts) Show(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := a.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var account Account
	if err := a.db.First(&account, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, account.serialize())
}

func (a *Accounts) VerifyCredentials(w http.ResponseWriter, r *http.Request) {
	user, err := a.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, user.serialize())
}

func (a *Accounts) StatusesShow(w http.ResponseWriter, r *http.Request) {
	_, err := a.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	var statuses []Status
	tx := a.db.Preload("Account").Where("account_id = ?", id)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 40 {
		limit = 20
	}
	tx = tx.Limit(limit)
	sinceID, _ := strconv.Atoi(r.URL.Query().Get("since_id"))
	if sinceID > 0 {
		tx = tx.Where("id > ?", sinceID)
	}
	if err := tx.Order("id desc").Find(&statuses).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []any
	for _, status := range statuses {
		resp = append(resp, status.serialize())
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, resp)
}

type accounts struct {
	db      *gorm.DB
	service *Service
}

// FindOrCreateAccount finds an account by username and domain, or creates a new
// one if it doesn't exist.
func (a *accounts) FindOrCreateAccount(uri string) (*Account, error) {
	username, domain, err := splitAcct(uri)
	if err != nil {
		return nil, err
	}
	instance, err := a.service.instances().FindOrCreateInstance(domain)
	if err != nil {
		return nil, err
	}

	var account Account
	err = a.db.Where("username = ? AND domain = ?", username, instance.Domain).First(&account).Error
	if err == nil {
		// found cached key
		return &account, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// use admin account to sign the request
	signAs, err := a.service.Accounts().FindAdminAccount()
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)
	if err := sign(req, signAs); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch account: %s", resp.Status)
	}

	var obj map[string]interface{}
	if err := json.UnmarshalFull(resp.Body, &obj); err != nil {
		return nil, err
	}

	account = Account{
		Username:       username,
		Domain:         domain,
		InstanceID:     instance.ID,
		Instance:       instance,
		DisplayName:    stringFromAny(obj["name"]),
		Locked:         boolFromAny(obj["manuallyApprovesFollowers"]),
		Bot:            stringFromAny(obj["type"]) == "Service",
		Note:           stringFromAny(obj["summary"]),
		Avatar:         stringFromAny(mapFromAny(obj["icon"])["url"]),
		AvatarStatic:   stringFromAny(mapFromAny(obj["icon"])["url"]),
		Header:         stringFromAny(mapFromAny(obj["image"])["url"]),
		HeaderStatic:   stringFromAny(mapFromAny(obj["image"])["url"]),
		FollowersCount: 0,
		FollowingCount: 0,
		StatusesCount:  0,
		LastStatusAt:   time.Now(),

		PublicKey: []byte(stringFromAny(mapFromAny(obj["publicKey"])["publicKeyPem"])),
	}
	if err := a.db.Create(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (a *accounts) FindAdminAccount() (*Account, error) {
	var account Account
	if err := a.db.Where("username = ? AND domain = ?", "dave", "cheney.net").Joins("LocalAccount").First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func sign(r *http.Request, account *Account) error {
	r.Header.Set("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")) // Date must be in GMT, not UTC 🤯
	privPem, _ := pem.Decode(account.LocalAccount.PrivateKey)
	if privPem.Type != "RSA PRIVATE KEY" {
		return errors.New("expected RSA PRIVATE KEY")
	}

	var parsedKey interface{}
	var err error
	if parsedKey, err = x509.ParsePKCS1PrivateKey(privPem.Bytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(privPem.Bytes); err != nil { // note this returns type `interface{}`
			return err
		}
	}

	var privateKey *rsa.PrivateKey
	var ok bool
	privateKey, ok = parsedKey.(*rsa.PrivateKey)
	if !ok {
		return errors.New("expected *rsa.PrivateKey")
	}
	headersToSign := []string{httpsig.RequestTarget, "date"}
	signer, _, err := httpsig.NewSigner(
		[]httpsig.Algorithm{httpsig.RSA_SHA256},
		httpsig.DigestSha256,
		headersToSign,
		httpsig.Signature,
		60,
	)
	if err != nil {
		return err
	}
	return signer.SignRequest(privateKey, account.PublicKeyID(), r, nil)
}

func splitAcct(acct string) (string, string, error) {
	url, err := url.Parse(acct)
	if err != nil {
		return "", "", fmt.Errorf("splitAcct: %w", err)
	}
	return path.Base(url.Path), url.Host, nil
}

type AccountList struct {
	gorm.Model
	AccountID     uint
	Title         string `gorm:"size:64"`
	RepliesPolicy string `gorm:"size:64"`
}

func (a *AccountList) serialize() map[string]any {
	return map[string]any{
		"id":             strconv.Itoa(int(a.ID)),
		"title":          a.Title,
		"replies_policy": a.RepliesPolicy,
	}
}

func boolFromAny(v any) bool {
	b, _ := v.(bool)
	return b
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func mapFromAny(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}