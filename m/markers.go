package m

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Marker struct {
	gorm.Model
	AccountID  uint
	Name       string `gorm:"size:32"`
	Version    int    `gorm:"default:0"`
	LastReadId uint
}

type Markers struct {
	db      *gorm.DB
	service *Service
}

func (m *Markers) Index(w http.ResponseWriter, req *http.Request) {
	accessToken := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	account, err := m.service.tokens().FindByAccessToken(accessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var markers []Marker
	if err := m.db.Model(account).Association("Markers").Find(&markers); err != nil {
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
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, resp)
}

func (m *Markers) Create(w http.ResponseWriter, req *http.Request) {
	accessToken := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	_, err := m.service.tokens().FindByAccessToken(accessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	buf, _ := httputil.DumpRequest(req, true)
	println(string(buf))
	w.WriteHeader(http.StatusNotImplemented)
}
