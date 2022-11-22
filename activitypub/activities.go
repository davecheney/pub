package activitypub

import (
	"encoding/json"
	"log"

	"github.com/jmoiron/sqlx"
)

type activities struct {
	db *sqlx.DB
}

func (a *activities) create(activity map[string]interface{}) error {
	b, err := json.Marshal(activity)
	if err != nil {
		return err
	}
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	object, _ := activity["object"].(map[string]interface{})
	objectType, _ := object["type"].(string)
	if _, err := tx.Exec("INSERT INTO activitypub_inbox (activity_type, object_type, activity) VALUES (?,?,?)", activity["type"], objectType, b); err != nil {
		log.Println("storeActivity:", err)
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
