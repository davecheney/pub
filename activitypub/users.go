package activitypub

import (
	"fmt"
	"net/http"
	"time"

	"github.com/davecheney/m/mastodon"
	"github.com/go-json-experiment/json"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type Users struct {
	db       *gorm.DB
	instance *mastodon.Instance
}

func NewUsers(db *gorm.DB, instance *mastodon.Instance) *Users {
	return &Users{
		db:       db,
		instance: instance,
	}
}

func (u *Users) accounts() *mastodon.Accounts {
	return mastodon.NewAccounts(u.db, u.instance)
}

func (u *Users) Show(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	var account mastodon.Account
	if err := u.db.Where("username = ? and domain = ?", username, u.instance.Domain).First(&account).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/activity+json")
	json.MarshalFull(w, map[string]any{
		"@context": []any{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
			map[string]any{
				"manuallyApprovesFollowers": "as:manuallyApprovesFollowers",
				"toot":                      "http://joinmastodon.org/ns#",
				"featured": map[string]any{
					"@id":   "toot:featured",
					"@type": "@id",
				},
				"featuredTags": map[string]any{
					"@id":   "toot:featuredTags",
					"@type": "@id",
				},
				"alsoKnownAs": map[string]any{
					"@id":   "as:alsoKnownAs",
					"@type": "@id",
				},
				"movedTo": map[string]any{
					"@id":   "as:movedTo",
					"@type": "@id",
				},
				"schema":           "http://schema.org#",
				"PropertyValue":    "schema:PropertyValue",
				"value":            "schema:value",
				"discoverable":     "toot:discoverable",
				"Device":           "toot:Device",
				"Ed25519Signature": "toot:Ed25519Signature",
				"Ed25519Key":       "toot:Ed25519Key",
				"Curve25519Key":    "toot:Curve25519Key",
				"EncryptedMessage": "toot:EncryptedMessage",
				"publicKeyBase64":  "toot:publicKeyBase64",
				"deviceId":         "toot:deviceId",
				"claim": map[string]any{
					"@type": "@id",
					"@id":   "toot:claim",
				},
				"fingerprintKey": map[string]any{
					"@type": "@id",
					"@id":   "toot:fingerprintKey",
				},
				"identityKey": map[string]any{
					"@type": "@id",
					"@id":   "toot:identityKey",
				},
				"devices": map[string]any{
					"@type": "@id",
					"@id":   "toot:devices",
				},
				"messageFranking": "toot:messageFranking",
				"messageType":     "toot:messageType",
				"cipherText":      "toot:cipherText",
				"suspended":       "toot:suspended",
				"focalPoint": map[string]any{
					"@container": "@list",
					"@id":        "toot:focalPoint",
				},
			},
		},
		"id":                        fmt.Sprintf("https://%s/users/%s", account.Domain, account.Username),
		"type":                      "Person",
		"following":                 fmt.Sprintf("https://%s/users/%s/following", account.Domain, account.Username),
		"followers":                 fmt.Sprintf("https://%s/users/%s/followers", account.Domain, account.Username),
		"inbox":                     fmt.Sprintf("https://%s/users/%s/inbox", account.Domain, account.Username),
		"outbox":                    fmt.Sprintf("https://%s/users/%s/outbox", account.Domain, account.Username),
		"featured":                  fmt.Sprintf("https://%s/users/%s/collections/featured", account.Domain, account.Username),
		"featuredTags":              fmt.Sprintf("https://%s/users/%s/collections/featuredTags", account.Domain, account.Username),
		"preferredUsername":         account.Username,
		"name":                      account.DisplayName,
		"summary":                   account.Note,
		"url":                       fmt.Sprintf("https://%s/@%s", account.Domain, account.Username),
		"manuallyApprovesFollowers": account.Locked,
		"discoverable":              true,
		"published":                 account.CreatedAt.UTC().Format(time.RFC3339),
		"devices":                   fmt.Sprintf("https://%s/users/%s/devices", account.Domain, account.Username),
		"publicKey": map[string]any{
			"id":           account.PublicKeyID(),
			"owner":        fmt.Sprintf("https://%s/users/%s", account.Domain, account.Username),
			"publicKeyPem": string(account.PublicKey),
		},
		"tag":        []any{},
		"attachment": []any{},
		"endpoints": map[string]any{
			"sharedInbox": fmt.Sprintf("https://%s/inbox", account.Domain),
		},
		"icon": map[string]any{
			"type":      "Image",
			"mediaType": "image/jpeg",
			"url":       account.AvatarStatic,
		},
		"image": map[string]any{
			"type":      "Image",
			"mediaType": "image/jpeg",
			"url":       account.HeaderStatic,
		},
	})
}

func (u *Users) InboxCreate(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.UnmarshalFull(r.Body, &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	actor := stringFromAny(body["actor"])
	account, err := u.accounts().FindOrCreateAccount(actor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	object, _ := body["object"].(map[string]interface{})
	objectType, _ := object["type"].(string)

	activity := &Activity{
		Account:      account,
		Activity:     body,
		ActivityType: stringFromAny(body["type"]),
		ObjectType:   objectType,
	}
	if err := u.db.Create(activity).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}
