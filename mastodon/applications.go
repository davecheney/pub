package mastodon

import (
	"gorm.io/gorm"
)

type Application struct {
	gorm.Model
	Name         string  `json:"name"`
	Website      *string `json:"website"`
	RedirectURI  string  `json:"redirect_uri"`
	ClientID     string  `json:"client_id"`
	ClientSecret string  `json:"client_secret"`
	VapidKey     string  `json:"vapid_key"`
}

type applications struct {
	db *gorm.DB
}

func (a *applications) create(app *Application) error {
	result := a.db.Create(app)
	return result.Error
}

func (a *applications) findByClientID(clientID string) (*Application, error) {
	app := &Application{}
	result := a.db.Where("client_id = ?", clientID).First(app)
	return app, result.Error
}
