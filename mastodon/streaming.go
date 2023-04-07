package mastodon

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/davecheney/pub/internal/streaming"
	"github.com/davecheney/pub/models"
	"github.com/go-json-experiment/json"
)

func StreamingHealth(env *Env, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, "OK")
	return err
}

func StreamingPublic(env *Env, w http.ResponseWriter, r *http.Request) error {
	sub := env.Subscribe()
	defer sub.Cancel()
	return stream(r.Context(), w, r, sub)
}

// stream writes a stream of SSE events to w. If
func stream(ctx context.Context, w http.ResponseWriter, r *http.Request, sub *streaming.Subscription) error {
	rc := http.NewResponseController(w)
	rc.SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	thumpTime := 30 * time.Second
	thump := time.NewTicker(thumpTime)
	defer thump.Stop()

	if _, err := io.WriteString(w, ":)\n\n"); err != nil {
		return err
	}
	if err := rc.Flush(); err != nil {
		return err
	}

	for {
		select {
		case <-thump.C:
			_, err := io.WriteString(w, ":thump\n\n")
			if err != nil {
				return err
			}
			if err := rc.Flush(); err != nil {
				return err
			}
		case payload, ok := <-sub.C:
			if !ok {
				return fmt.Errorf("subscription cancelled")
			}
			_, err := io.WriteString(w, "event: "+payload.Event+"\ndata: ")
			if err != nil {
				return err
			}
			serialise := Serialiser{req: r}
			switch data := payload.Data.(type) {
			case *models.Status:
				if err := json.MarshalFull(w, serialise.Status(data)); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unhandled payload type %T", payload.Data)
			}
			_, err = io.WriteString(w, "\n\n")
			if err != nil {
				return err
			}
			if err := rc.Flush(); err != nil {
				return err
			}
			thump.Reset(thumpTime)
		case <-ctx.Done():
			return fmt.Errorf("streaming: context done: %w", ctx.Err())
		}
	}
}
