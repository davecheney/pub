package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/snowflake"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func MarkersIndex(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	names := r.URL.Query()["timeline[]"]
	var markers []*models.AccountMarker
	if err := env.DB.Model(user).Association("Markers").Find(&markers, "name in (?)", names); err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(markers, serialise.Marker))
}

func MarkersCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}
	m := make(map[string]struct {
		LastReadID snowflake.ID `json:"last_read_id,string"`
	})
	if err := json.UnmarshalFull(r.Body, &m); err != nil {
		return httpx.Error(http.StatusBadRequest, err)
	}

	markers := make(map[string]*models.AccountMarker)
	for name, v := range m {
		marker := &models.AccountMarker{
			AccountID:  user.ID,
			Name:       name,
			LastReadID: v.LastReadID,
		}
		markers[name] = marker

		// this elaborate upsert avoids the need to load the existing marker
		// from the database before overwriting it.
		if err := env.DB.Omit("Version").Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "account_id"}, {Name: "name"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"last_read_id": marker.LastReadID,
				"updated_at":   gorm.Expr("VALUES(updated_at)"),
				"version":      gorm.Expr("version + 1"),
			}),
		}).Save(&marker).Error; err != nil {
			return err
		}
	}
	resp := make(map[string]any)
	serialise := Serialiser{req: r}
	for name, marker := range markers {
		resp[name] = serialise.Marker(marker)
	}

	return to.JSON(w, resp)
}
