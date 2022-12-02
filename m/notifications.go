package m

import (
	"net/http"
	"strconv"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Notification struct {
	gorm.Model
	AccountID uint
	Account   *Account
	StatusID  *uint
	Status    *Status
	Type      string `gorm:"size:64"`
}

func (n *Notification) serialize() map[string]any {
	return map[string]any{
		"id":         strconv.Itoa(int(n.ID)),
		"type":       n.Type,
		"created_at": n.CreatedAt.UTC().Format("2006-01-02T15:04:05.006Z"),
		"account":    n.Account.serialize(),
		"status":     n.Status.serialize(),
	}
}

type Notifications struct {
	db      *gorm.DB
	service *Service
}

func (n *Notifications) Index(w http.ResponseWriter, r *http.Request) {
	user, err := n.service.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var notifications []Notification
	if err := n.db.Where("account_id = ?", user.ID).Preload("Status").Preload("Status.Account").Find(&notifications).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []any
	for _, notification := range notifications {
		resp = append(resp, notification.serialize())
	}

	w.Header().Set("Content-Type", "application/json")
	json.MarshalFull(w, resp)
}
