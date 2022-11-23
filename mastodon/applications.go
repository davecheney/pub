package mastodon

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type Application struct {
	ID           int       `json:"id,string" db:"id"`
	CreatedAt    time.Time `json:"-" db:"created_at"`
	Name         string    `json:"name" db:"name"`
	Website      *string   `json:"website" db:"website"`
	RedirectURI  string    `json:"redirect_uri" db:"redirect_uri"`
	ClientID     string    `json:"client_id" db:"client_id"`
	ClientSecret string    `json:"client_secret" db:"client_secret"`
	VapidKey     string    `json:"vapid_key" db:"vapid_key"`
}

type applications struct {
	db *sqlx.DB
}

func (a *applications) create(app *Application) error {
	result, err := a.db.NamedExec(`INSERT INTO applications (name, website, redirect_uri, client_id, client_secret, vapid_key) VALUES (:name, :website, :redirect_uri, :client_id, :client_secret, :vapid_key)`, app)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	app.ID = int(id)
	return nil
}

func (a *applications) findByClientID(clientID string) (*Application, error) {
	app := &Application{}
	err := a.db.QueryRowx(`SELECT * FROM applications WHERE client_id = ?`, clientID).StructScan(app)
	return app, err
}
