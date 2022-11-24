package mastodon

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type Status struct {
	// ID of the status in the database.
	Id int `json:"id,string" db:"id"`
	// URI of the status for federation purposes.
	Uri string `json:"uri,omitempty" db:"-"`
	// The time when this status was created.
	CreatedAt time.Time `json:"created_at,omitempty" db:"created_at"`

	// HTML-encoded status content.
	Content string `json:"content,omitempty" db:"content"`
	//  Visibility of this status.
	Visibility string `json:"visibility,omitempty" db:"visibility"`

	Account   *Account `json:"account,omitempty" db:"-"`
	AccountID int      `json:"-" db:"account_id"`
}

type statuses struct {
	db *sqlx.DB
}
