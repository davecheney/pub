package wellknown

import (
	"io"
	"net/http"

	"github.com/davecheney/m/m"
	"github.com/go-json-experiment/json"
)

type Service struct {
	*m.Service
}

func NewService(service *m.Service) *Service {
	return &Service{
		Service: service,
	}
}

// NodeInfo returns a NodeInfo REST resource.
func (s *Service) NodeInfo() *NodeInfo {
	return &NodeInfo{
		service: s,
	}
}

func (s *Service) Webfinger() *Webfinger {
	return &Webfinger{
		service: s,
	}
}

func (w *Service) HostMeta(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/xrd+xml")
	io.WriteString(rw, `<?xml version="1.0" encoding="UTF-8"?>
		<XRD xmlns="http://docs.oasis-open.org/ns/xri/xrd-1.0">
		<Subject>`+r.Host+`</Subject>
		<Link rel="lrdd" template="https://`+r.Host+`/.well-known/webfinger?resource={uri}"/>
		</XRD>`)
}

// toJSON writes the given object to the response body as JSON.
func toJSON(w http.ResponseWriter, obj interface{}) error {
	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	return json.MarshalFull(w, obj)
}
