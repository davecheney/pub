package activitypub

import (
	"net/http"

	"github.com/davecheney/m/internal/webfinger"
	"github.com/davecheney/m/m"
	"github.com/go-chi/chi/v5"
)

type Users struct {
	service *Service
}

func (u *Users) Show(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	var account m.Account
	if err := u.service.db.Where("username = ? and domain = ?", username, r.Host).First(&account).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Cache-Control", "max-age=180, public")
	acct := webfinger.Acct{
		User: account.Username,
		Host: account.Domain,
	}
	toJSON(w, map[string]any{
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
		"id":                        acct.ID(),
		"type":                      "Person",
		"following":                 acct.Following(),
		"followers":                 acct.Followers(),
		"inbox":                     acct.Inbox(),
		"outbox":                    acct.Outbox(),
		"featured":                  acct.ID() + "/collections/featured",
		"featuredTags":              acct.ID() + "/collections/tags",
		"preferredUsername":         account.Username,
		"name":                      account.DisplayName,
		"summary":                   account.Note,
		"url":                       account.URL(),
		"manuallyApprovesFollowers": account.Locked,
		"discoverable":              false,                                                  // mastodon sets this to false
		"published":                 account.CreatedAt.UTC().Format("2006-01-02T00:00:00Z"), // spec says round created_at to nearest day
		"devices":                   acct.ID() + "/collections/devices",
		"publicKey": map[string]any{
			"id":           account.PublicKeyID(),
			"owner":        acct.ID(),
			"publicKeyPem": string(account.PublicKey),
		},
		"tag":        []any{},
		"attachment": []any{},
		"endpoints": map[string]any{
			"sharedInbox": acct.SharedInbox(),
		},
		"icon": map[string]any{
			"type":      "Image",
			"mediaType": "image/jpeg",
			"url":       account.Avatar,
		},
	})
}
