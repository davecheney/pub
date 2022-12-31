package mastodon

import (
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/davecheney/m/internal/models"
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

	var markers []models.Marker
	if err := ms.service.DB().Model(user).Association("Markers").Find(&markers); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]any{}
	for _, marker := range markers {
		resp[marker.Name] = map[string]any{
			"last_read_id": utoa(marker.LastReadId),
			"version":      marker.Version,
			"updated_at":   marker.UpdatedAt.Format("2006-01-02T15:04:05.006Z"),
		}
	}
	toJSON(w, resp)
}

func (ms *Markers) Create(w http.ResponseWriter, r *http.Request) {
	_, err := ms.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	buf, _ := httputil.DumpRequest(r, true)
	println(string(buf))
	w.WriteHeader(http.StatusNotImplemented)
}

func utoa(u uint) string {
	return strconv.FormatUint(uint64(u), 10)
}
