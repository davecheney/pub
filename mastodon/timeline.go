package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func TimelinesHome(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	following := env.DB.Select("target_id").Where(&models.Relationship{ActorID: user.Actor.ID, Following: true}).Table("relationships")

	var statuses []*models.Status
	// TODO stop copying and pasting this query
	scope := env.DB.Joins("Actor").Scopes(models.PaginateStatuses(r), models.PreloadStatus).
		Where("(actor_id IN (?) AND in_reply_to_actor_id is null) or (actor_id in (?) and in_reply_to_actor_id IN (?))", following, following, following)
	query := scope.Preload("Reaction", "actor_id = ?", user.Actor.ID) // reactions
	query = query.Preload("Reblog.Reaction", "actor_id = ?", user.Actor.ID)
	if err := query.Find(&statuses).Error; err != nil {
		return httpx.Error(http.StatusInternalServerError, err)
	}

	sortStatuses(statuses) // PaginateStatuses doesn't sort, so we have to do it ourselves.

	if len(statuses) > 0 {
		linkHeader(w, r, statuses[0].ID, statuses[len(statuses)-1].ID)
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(statuses, serialise.Status))
}

func TimelinesPublic(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	authenticated := err == nil

	var statuses []*models.Status
	query := env.DB.Scopes(models.PaginateStatuses(r), publicStatuses, localOnly(r), models.PreloadStatus)
	// localOnly handles the join to the actors table
	if authenticated {
		query = query.Preload("Reaction", "actor_id = ?", user.Actor.ID) // reactions
		query = query.Preload("Reblog.Reaction", "actor_id = ?", user.Actor.ID)
	}
	if err := query.Find(&statuses).Error; err != nil {
		return httpx.Error(http.StatusInternalServerError, err)
	}

	sortStatuses(statuses) // PaginateStatuses doesn't sort, so we have to do it ourselves.

	if len(statuses) > 0 {
		linkHeader(w, r, statuses[0].ID, statuses[len(statuses)-1].ID)
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(statuses, serialise.Status))
}

// localOnly returns a scope that filters statuses to those that are local
// to the instance.
func localOnly(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch r.URL.Query().Get("local") {
		case "true":
			return db.Joins("Actor").Where("Actor.domain = ?", r.Host)
		default:
			return db.Joins("Actor")
		}
	}
}

func TimelinesListShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var listMembers []int64
	if err := env.DB.Model(&models.AccountListMember{}).Where("account_list_id = ? ", chi.URLParam(r, "id")).Pluck("member_id", &listMembers).Error; err != nil {
		return httpx.Error(http.StatusInternalServerError, err)
	}

	var statuses []*models.Status
	scope := env.DB.Scopes(models.PaginateStatuses(r), models.PreloadStatus).Where("(actor_id IN (?) AND in_reply_to_actor_id is null) or (actor_id in (?) and in_reply_to_actor_id IN (?))", listMembers, listMembers, listMembers)
	query := scope.Joins("Actor")
	query = query.Preload("Reaction", "actor_id = ?", user.Actor.ID) // reactions
	query = query.Preload("Reblog.Reaction", "actor_id = ?", user.Actor.ID)
	if err := query.Find(&statuses).Error; err != nil {
		return httpx.Error(http.StatusInternalServerError, err)
	}

	sortStatuses(statuses) // PaginateStatuses doesn't sort, so we have to do it ourselves.

	if len(statuses) > 0 {
		linkHeader(w, r, statuses[0].ID, statuses[len(statuses)-1].ID)
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(statuses, serialise.Status))
}

func TimelinesTagShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	user, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var statuses []*models.Status
	scope := env.DB.Scopes(models.PaginateStatuses(r))
	// use Joins("JOIN status_tags ...") as Joins("Tags") -- joining on an association -- causes a reflect panic in gorm.
	// no biggie, just write the JOIN manually.
	tag := env.DB.Select("id").Where("name = ?", chi.URLParam(r, "tag")).Table("tags")
	query := scope.Joins("JOIN status_tags ON status_tags.status_id = statuses.id").Where("status_tags.tag_id = (?)", tag)
	query = query.Preload("Actor").Scopes(models.PreloadStatus)
	query = query.Preload("Reaction", "actor_id = ?", user.Actor.ID) // reactions
	query = query.Preload("Reblog.Reaction", "actor_id = ?", user.Actor.ID)
	if err := query.Find(&statuses).Error; err != nil {
		return httpx.Error(http.StatusInternalServerError, err)
	}

	sortStatuses(statuses) // PaginateStatuses doesn't sort, so we have to do it ourselves.

	if len(statuses) > 0 {
		linkHeader(w, r, statuses[0].ID, statuses[len(statuses)-1].ID)
	}
	serialise := Serialiser{req: r}
	return to.JSON(w, algorithms.Map(statuses, serialise.Status))
}

// publicStatuses returns a scope that only returns public statuses which are not replies or reblogs.
func publicStatuses(db *gorm.DB) *gorm.DB {
	return db.Where("visibility = ? and reblog_id is null and in_reply_to_id is null", "public")
}
