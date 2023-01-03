package mastodon

import (
	"net/http"
	"net/http/httputil"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/internal/models"
	"github.com/davecheney/pub/internal/to"
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

	return to.JSON(w, algorithms.Map(markers, seraliseMarker))
}

func MarkersCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	_, err := env.authenticate(r)
	if err != nil {
		return err
	}
	buf, _ := httputil.DumpRequest(r, true)
	println(string(buf))
	w.WriteHeader(http.StatusInternalServerError)
	return nil
}
