package m

import (
	"net/http"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type NodeInfo struct {
	db     *gorm.DB
	domain string
}

func NewNodeInfo(db *gorm.DB, domain string) *NodeInfo {
	return &NodeInfo{
		db:     db,
		domain: domain,
	}
}

func (ni *NodeInfo) Get(w http.ResponseWriter, r *http.Request) {
	var instance Instance
	if err := ni.db.Where("domain = ?", ni.domain).First(&instance).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, instance.serializeNodeInfo())
}
