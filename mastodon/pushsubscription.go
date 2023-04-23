package mastodon

import (
	"net/http"

	"github.com/davecheney/pub/internal/httpx"
	"github.com/davecheney/pub/internal/to"
	"github.com/davecheney/pub/models"
)

func PushSubscriptionCreate(env *Env, w http.ResponseWriter, r *http.Request) error {
	account, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var body struct {
		Data struct {
			Policy string `json:"policy" schema:"data[policy]"`
			Alerts struct {
				Status        bool `json:"status" schema:"data[alerts][status]"`
				Reblog        bool `json:"reblog" schema:"data[alerts][reblog]"`
				Follow        bool `json:"follow" schema:"data[alerts][follow]"`
				FollowRequest bool `json:"follow_request" schema:"data[alerts][follow_request]"`
				Favourite     bool `json:"favourite" schema:"data[alerts][favourite]"`
				Poll          bool `json:"poll" schema:"data[alerts][poll]"`
				Update        bool `json:"update" schema:"data[alerts][update]"`
				Mention       bool `json:"mention" schema:"data[alerts][mention]"`
			} `json:"alerts"`
		} `json:"data" schema:"data"`
		Subscription struct {
			Endpoint string `json:"endpoint" schema:"subscription[endpoint]"`
			Keys     struct {
				P256DH string `json:"p256dh" schema:"subscription[keys][p256dh]"`
				Auth   string `json:"auth" schema:"subscription[keys][auth]"`
			} `json:"keys"`
		} `json:"subscription" schema:"subscription"`
	}
	if err := httpx.Params(r, &body); err != nil {
		return err
	}
	sub := models.PushSubscription{
		AccountID:     account.ID,
		Endpoint:      body.Subscription.Endpoint,
		Mention:       false,
		Status:        body.Data.Alerts.Status,
		Reblog:        body.Data.Alerts.Reblog,
		Follow:        body.Data.Alerts.Follow,
		FollowRequest: body.Data.Alerts.FollowRequest,
		Favourite:     body.Data.Alerts.Favourite,
		Poll:          body.Data.Alerts.Poll,
		Update:        body.Data.Alerts.Update,
		Policy:        models.PushSubscriptionPolicy(body.Data.Policy),
	}
	if err := env.DB.Create(&sub).Error; err != nil {
		return err
	}
	ser := Serialiser{req: r}
	return to.JSON(w, ser.WebPushSubscription(&sub))
}

func PushSubscriptionUpdate(env *Env, w http.ResponseWriter, r *http.Request) error {
	account, err := env.authenticate(r)
	if err != nil {
		return err
	}
	var body struct {
		Data struct {
			Policy string `json:"policy" schema:"data[policy]"`
			Alerts struct {
				Status        bool `json:"status" schema:"data[alerts][status]"`
				Reblog        bool `json:"reblog" schema:"data[alerts][reblog]"`
				Follow        bool `json:"follow" schema:"data[alerts][follow]"`
				FollowRequest bool `json:"follow_request" schema:"data[alerts][follow_request]"`
				Favourite     bool `json:"favourite" schema:"data[alerts][favourite]"`
				Poll          bool `json:"poll" schema:"data[alerts][poll]"`
				Update        bool `json:"update" schema:"data[alerts][update]"`
				Mention       bool `json:"mention" schema:"data[alerts][mention]"`
			} `json:"alerts"`
		} `json:"data" schema:"data"`
	}
	if err := httpx.Params(r, &body); err != nil {
		return err
	}
	var sub models.PushSubscription
	if err := env.DB.First(&sub, models.PushSubscription{AccountID: account.ID}).Error; err != nil {
		return err
	}
	sub.Status = body.Data.Alerts.Status
	sub.Reblog = body.Data.Alerts.Reblog
	sub.Follow = body.Data.Alerts.Follow
	sub.FollowRequest = body.Data.Alerts.FollowRequest
	sub.Favourite = body.Data.Alerts.Favourite
	sub.Poll = body.Data.Alerts.Poll
	sub.Update = body.Data.Alerts.Update
	sub.Mention = body.Data.Alerts.Mention
	if body.Data.Policy != "" {
		sub.Policy = models.PushSubscriptionPolicy(body.Data.Policy)
	}
	if err := env.DB.Save(&sub).Error; err != nil {
		return err
	}
	ser := Serialiser{req: r}
	return to.JSON(w, ser.WebPushSubscription(&sub))
}

func PushSubscriptionShow(env *Env, w http.ResponseWriter, r *http.Request) error {
	account, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var sub models.PushSubscription
	if err := env.DB.FirstOrInit(&sub, models.PushSubscription{AccountID: account.ID}).Error; err != nil {
		return err
	}

	ser := Serialiser{req: r}
	return to.JSON(w, ser.WebPushSubscription(&sub))
}

func PushSubscriptionDestroy(env *Env, w http.ResponseWriter, r *http.Request) error {
	account, err := env.authenticate(r)
	if err != nil {
		return err
	}

	var sub models.PushSubscription
	if err := env.DB.Delete(&sub, models.PushSubscription{AccountID: account.ID}).Error; err != nil {
		return err
	}

	if err := env.DB.Delete(&sub).Error; err != nil {
		return err
	}

	return to.JSON(w, make(map[string]interface{}))
}
