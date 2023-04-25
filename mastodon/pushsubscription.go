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
				Status        BoolOrBit `json:"status" schema:"data[alerts][status]"`
				Reblog        BoolOrBit `json:"reblog" schema:"data[alerts][reblog]"`
				Follow        BoolOrBit `json:"follow" schema:"data[alerts][follow]"`
				FollowRequest BoolOrBit `json:"follow_request" schema:"data[alerts][follow_request]"`
				Favourite     BoolOrBit `json:"favourite" schema:"data[alerts][favourite]"`
				Poll          BoolOrBit `json:"poll" schema:"data[alerts][poll]"`
				Update        BoolOrBit `json:"update" schema:"data[alerts][update]"`
				Mention       BoolOrBit `json:"mention" schema:"data[alerts][mention]"`
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
		Status:        bool(body.Data.Alerts.Status),
		Reblog:        bool(body.Data.Alerts.Reblog),
		Follow:        bool(body.Data.Alerts.Follow),
		FollowRequest: bool(body.Data.Alerts.FollowRequest),
		Favourite:     bool(body.Data.Alerts.Favourite),
		Poll:          bool(body.Data.Alerts.Poll),
		Update:        bool(body.Data.Alerts.Update),
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
				Status        BoolOrBit `json:"status" schema:"data[alerts][status]"`
				Reblog        BoolOrBit `json:"reblog" schema:"data[alerts][reblog]"`
				Follow        BoolOrBit `json:"follow" schema:"data[alerts][follow]"`
				FollowRequest BoolOrBit `json:"follow_request" schema:"data[alerts][follow_request]"`
				Favourite     BoolOrBit `json:"favourite" schema:"data[alerts][favourite]"`
				Poll          BoolOrBit `json:"poll" schema:"data[alerts][poll]"`
				Update        BoolOrBit `json:"update" schema:"data[alerts][update]"`
				Mention       BoolOrBit `json:"mention" schema:"data[alerts][mention]"`
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
	sub.Status = bool(body.Data.Alerts.Status)
	sub.Reblog = bool(body.Data.Alerts.Reblog)
	sub.Follow = bool(body.Data.Alerts.Follow)
	sub.FollowRequest = bool(body.Data.Alerts.FollowRequest)
	sub.Favourite = bool(body.Data.Alerts.Favourite)
	sub.Poll = bool(body.Data.Alerts.Poll)
	sub.Update = bool(body.Data.Alerts.Update)
	sub.Mention = bool(body.Data.Alerts.Mention)
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

	if err := env.DB.Where("account_id = ?", account.ID).Delete(models.PushSubscription{}).Error; err != nil {
		return err
	}

	return to.JSON(w, make(map[string]interface{}))
}
