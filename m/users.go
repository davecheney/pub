package m

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-json-experiment/json"

	"gorm.io/gorm"
)

type Users struct {
	db      *gorm.DB
	service *Service
}

func (u *Users) Show(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	var account Account
	if err := u.db.Where("username = ? and domain = ?", username, u.service.Domain()).First(&account).Error; err != nil {
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
