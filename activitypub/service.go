package activitypub

import (
	"encoding/json"
	"net/http"
)

// Service implements an ActivityPub service.
type Service struct {
	StoreActivity func(activity map[string]any) error
}

func (svc *Service) Inbox(w http.ResponseWriter, r *http.Request) {
	var activity map[string]any
	if err := json.NewDecoder(r.Body).Decode(&activity); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := svc.StoreActivity(activity); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
