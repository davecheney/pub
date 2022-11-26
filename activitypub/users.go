package activitypub

import (
	"fmt"
	"net/http"

	"github.com/go-json-experiment/json"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type Users struct {
	db *gorm.DB
}

func NewUsers(db *gorm.DB) *Users {
	return &Users{
		db: db,
	}
}

func (u *Users) Show(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	actor_id := fmt.Sprintf("https://cheney.net/users/%s", username)
	var actor Actor
	if err := u.db.Where("actor_id = ?", actor_id).First(&actor).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/activity+json")
	json.MarshalFull(w, actor)
}

func (u *Users) InboxCreate(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.UnmarshalFull(r.Body, &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	object, _ := body["object"].(map[string]interface{})
	objectType, _ := object["type"].(string)

	activity := &Activity{
		Activity:     body,
		ActivityType: body["type"].(string),
		ObjectType:   objectType,
	}
	if err := u.db.Create(activity).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
