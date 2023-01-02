package mastodon

import (
	"net/http"
	"net/http/httputil"

	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/to"
)

type Markers struct {
	service *Service
}

func (ms *Markers) Index(w http.ResponseWriter, r *http.Request) {
	user, err := ms.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	names := r.URL.Query()["timeline[]"]
	var markers []models.AccountMarker
	if err := ms.service.db.Model(user).Association("Markers").Find(&markers, "name in (?)", names); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]any{}
	for _, marker := range markers {
		resp[marker.Name] = seraliseMarker(&marker)
	}
	to.JSON(w, resp)
}

func (ms *Markers) Create(w http.ResponseWriter, r *http.Request) {
	_, err := ms.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	buf, _ := httputil.DumpRequest(r, true)
	println(string(buf))
	w.WriteHeader(http.StatusInternalServerError)
}
