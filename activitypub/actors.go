package activitypub

import (
	"encoding/json"

	"github.com/jmoiron/sqlx"
)

type actors struct {
	db *sqlx.DB
}

func (a *actors) findById(id string) (map[string]any, error) {
	var b []byte
	err := a.db.QueryRowx("SELECT object FROM activitypub_actors WHERE actor_id = ? ORDER BY created_at desc LIMIT 1", id).Scan(&b)
	if err != nil {
		return nil, err
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func (a *actors) create(actor map[string]interface{}) error {
	b, err := json.Marshal(actor)
	if err != nil {
		return err
	}
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("INSERT INTO activitypub_actors (actor_id, type, object, publickey) VALUES (?,?,?,?)", actor["id"], actor["type"], b, actor["publicKey"].(map[string]interface{})["publicKeyPem"].(string)); err != nil {
		return err
	}
	return tx.Commit()
}
