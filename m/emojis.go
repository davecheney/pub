package m

import (
	"net/http"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Emojis struct {
	db *gorm.DB
}

func (e *Emojis) Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, []any{})
}
