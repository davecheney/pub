package mastodon

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/davecheney/pub/internal/streaming"
	"github.com/davecheney/pub/models"
	"github.com/go-json-experiment/json"
	"golang.org/x/net/websocket"
)

func StreamingWebsocket(env *Env, w http.ResponseWriter, r *http.Request) error {
	svr := websocket.Server{
		Handler: func(ws *websocket.Conn) {
			fmt.Println("StreamingHandler: connected: ", ws.LocalAddr(), ws.RemoteAddr())
			defer func() {
				fmt.Println("StreamingHandler: disconnected: ", ws.LocalAddr(), ws.RemoteAddr())
				ws.Close()
			}()

			readErr := make(chan error, 1)
			go func() {
				var val struct {
					Type   string `json:"type"`
					Stream string `json:"stream"`
					Tag    string `json:"tag"`
				}
				dec := json.DecodeOptions{}.NewDecoder(ws)
				for {
					err := json.UnmarshalOptions{}.UnmarshalNext(dec, &val)
					if err != nil {
						readErr <- err
						return
					}
					log.Printf("StreamingHandler: read: %+v", val)
				}
			}()
			ctx := ws.Request().Context()
			sub := env.Subscribe()
			defer sub.Cancel()
			for {
				select {
				case err := <-readErr:
					log.Println("StreamingHandler: read error:", err)
					return
				case <-ctx.Done():
					return
				case _, ok := <-sub.C:
					if !ok {
						return
					}
				case <-time.After(30 * time.Second):
					if _, err := ws.Write([]byte("")); err != nil {
						return
					}
				}
			}
		},
		Handshake: func(config *websocket.Config, req *http.Request) error {
			_, err := env.authenticate(req)
			return err
		},
		Config: websocket.Config{
			Origin: &url.URL{
				Host: r.RemoteAddr,
			},
		},
	}
	svr.ServeHTTP(w, r)
	return nil
}

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
