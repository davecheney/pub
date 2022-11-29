package m

import (
	"fmt"
	"net/http"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type NodeInfo struct {
	db     *gorm.DB
	domain string
}

func (ni *NodeInfo) Index(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	json.MarshalFull(rw, map[string]any{
		"links": []map[string]any{
			{
				"rel":  "http://nodeinfo.diaspora.software/ns/schema/2.0",
				"href": fmt.Sprintf("https://%s/nodeinfo/2.0", ni.domain),
			},
		},
	})
}

func (ni *NodeInfo) Show(w http.ResponseWriter, r *http.Request) {
	var instance Instance
	if err := ni.db.Where("domain = ?", ni.domain).First(&instance).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, instance.serializeNodeInfo())
}
