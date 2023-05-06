package activities

import (
	"github.com/davecheney/pub/models"
)

const (
	FOLLOW = "Follow"
	LIKE   = "Like"
	UNDO   = "Undo"
)

func Follow(actor, object *models.Actor) map[string]any {
	return map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"type":     FOLLOW,
		"actor":    actor.URI,
		"object":   object.URI,
	}
}

func Like(actor *models.Actor, object string) map[string]any {
	return map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"type":     LIKE,
		"actor":    actor.URI,
		"object":   object,
	}
}

func Unfollow(actor, object *models.Actor) map[string]any {
	return map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"type":     UNDO,
		"actor":    actor.URI,
		"object":   Follow(actor, object),
	}
}

func Unlike(actor *models.Actor, object string) map[string]any {
	return map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"type":     UNDO,
		"actor":    actor.URI,
		"object":   Like(actor, object),
	}
}
