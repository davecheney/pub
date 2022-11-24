package mastodon

import (
	"bytes"
	"context"
	"os"

	"github.com/go-json-experiment/json"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

func NewInboxProcessor(db *sqlx.DB) *InboxProcessor {
	return &InboxProcessor{
		db:  db,
		log: zerolog.New(os.Stderr),
	}
}

type InboxProcessor struct {
	db  *sqlx.DB
	log zerolog.Logger
}

func (ip *InboxProcessor) Run(ctx context.Context) error {
	rows, err := ip.db.Queryx(`SELECT id, activity FROM activitypub_inbox ORDER BY created_at ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var activity []byte
		if err := rows.Scan(&id, &activity); err != nil {
			return err
		}
		log := ip.log.With().Int("id", id).Logger()
		var a map[string]any
		if err := json.UnmarshalFull(bytes.NewReader(activity), &a); err != nil {
			log.Error().Err(err).Msg("unmarshal activity")
			continue
		}
		ip.processActivity(ctx, log, a)
	}
	return nil
}

func (ip *InboxProcessor) processActivity(ctx context.Context, log zerolog.Logger, a map[string]any) {
	typ := a["type"].(string)
	log = log.With().Str("type", typ).Logger()
	switch a["type"] {
	case "Create":
		ip.processCreate(ctx, log, a)
	default:
		log.Warn().Msg("unknown activity type")
	}
}

func (ip *InboxProcessor) processCreate(ctx context.Context, log zerolog.Logger, a map[string]any) {
	typ, _ := a["object"].(map[string]any)["type"].(string)
	log = log.With().Str("object_type", typ).Logger()
	switch typ {
	case "Note":
		ip.processCreateNote(ctx, log, a)
	default:
		log.Warn().Msg("unknown object type")
	}
}

func (ip *InboxProcessor) processCreateNote(ctx context.Context, log zerolog.Logger, a map[string]any) {
	log.Info().Msg("create note")
}
