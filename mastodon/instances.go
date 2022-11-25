package mastodon

import (
	"net/http"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Instances struct {
	db *gorm.DB
}

func NewInstance(db *gorm.DB) *Instances {
	return &Instances{
		db: db,
	}
}

func (i *Instances) Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, &Instance{
		URI:              "https://cheney.net/",
		Title:            "Casa del Cheese",
		ShortDescription: "🧀",
		Email:            "dave@cheney.net",
		Version:          "0.1.2",
		Languages:        []string{"en"},
	})
}

func (i *Instances) Peers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, []string{})
}
