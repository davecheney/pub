package wellknown

import (
	"io"
	"net/http"

	"github.com/davecheney/pub/activitypub"
)

func HostMetaIndex(env *activitypub.Env, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/xrd+xml")
	_, err := io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
		<XRD xmlns="http://docs.oasis-open.org/ns/xri/xrd-1.0">
		<Subject>`+r.Host+`</Subject>
		<Link rel="lrdd" template="https://`+r.Host+`/.well-known/webfinger?resource={uri}"/>
		</XRD>`)
	return err
}
