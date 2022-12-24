package activitypub

import (
	"net/http"
	"net/http/httputil"

	"github.com/davecheney/m/internal/snowflake"
	"github.com/davecheney/m/m"
	"github.com/go-chi/chi/v5"
)

type Users struct {
	service *Service
}

func (u *Users) Show(w http.ResponseWriter, r *http.Request) {
	buf, _ := httputil.DumpRequest(r, true)
	println("users.Show", string(buf))

	username := chi.URLParam(r, "username")
	var actor m.Actor
	if err := u.service.db.First(&actor, "name = ? and domain = ?", username, r.Host).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
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
		"id": actor.URI,
		"type": func(a *m.Actor) string {
			switch a.Type {
			case "LocalPerson":
				return "Person"
			default:
				return a.Type
			}
		}(&actor),
		"following":                 actor.URI + "/following",
		"followers":                 actor.URI + "/followers",
		"inbox":                     actor.URI + "/inbox",
		"outbox":                    actor.URI + "/outbox",
		"featured":                  actor.URI + "/collections/featured",
		"featuredTags":              actor.URI + "/collections/tags",
		"preferredUsername":         actor.Name,
		"name":                      actor.DisplayName,
		"summary":                   actor.Note,
		"url":                       actor.URL(),
		"manuallyApprovesFollowers": actor.Locked,
		"discoverable":              false,                                                       // mastodon sets this to false
		"published":                 snowflake.IDToTime(actor.ID).Format("2006-01-02T00:00:00Z"), // spec says round created_at to nearest day
		"devices":                   actor.URI + "/collections/devices",
		"publicKey": map[string]any{
			"id":           actor.PublicKeyID(),
			"owner":        actor.URI,
			"publicKeyPem": string(actor.PublicKey),
		},
		"tag":        []any{},
		"attachment": []any{},
		"endpoints": map[string]any{
			"sharedInbox": "https://" + r.Host + "/inbox",
		},
		"icon": map[string]any{
			"type":      "Image",
			"mediaType": "image/jpeg",
			"url":       actor.Avatar,
		},
	})
}
