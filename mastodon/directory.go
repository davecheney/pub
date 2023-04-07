package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

func DirectoryIndex(env *Env, w http.ResponseWriter, r *http.Request) error {
	var actors []*models.Actor
	query := env.DB.Scopes(models.PaginateActors(r), isLocal(r))
	if err := query.Find(&actors).Error; err != nil {
		return err
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(actors, serialise.Account))
}

func isLocal(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if r.URL.Query().Get("local") != "" {
			return db.Where("domain = ?", r.Host)
		}
		return db
	}
}
