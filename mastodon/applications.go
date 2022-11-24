package mastodon

import (
	"time"

	"gorm.io/gorm"
)

type Application struct {
	ID           uint           `json:"id,string" gorm:"primarykey"`
	CreatedAt    time.Time      `json:"-"`
	UpdatedAt    time.Time      `json:"-"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	Name         string         `json:"name"`
	Website      *string        `json:"website"`
	RedirectURI  string         `json:"redirect_uri"`
	ClientID     string         `json:"client_id"`
	ClientSecret string         `json:"client_secret"`
	VapidKey     string         `json:"vapid_key"`
	Tokens       []Token        `json:"-" gorm:"foreignKey:ApplicationID;references:ID"`
}

type applications struct {
	db *gorm.DB
}

func (a *applications) findByClientID(clientID string) (*Application, error) {
	app := &Application{}
	result := a.db.Where("client_id = ?", clientID).First(app)
	return app, result.Error
}
