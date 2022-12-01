package m

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-json-experiment/json"
	"gorm.io/gorm"
)

type Conversation struct {
	gorm.Model
	Visibility string `gorm:"type:enum('public', 'unlisted', 'private', 'direct')"`
	Statuses   []Status
}

type Conversations struct {
	db      *gorm.DB
	service *Service
}

func (c *Conversations) Index(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	_, err := c.service.tokens().FindByAccessToken(accessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var statuses []Status
	scope := c.db.Scopes(c.paginate(r)).Preload("Account").Where("visibility = ?", "direct")
	switch r.URL.Query().Get("local") {
	case "":
		scope = scope.Joins("Account")
	default:
		scope = scope.Joins("Account").Where("Account.instance_id = ?", c.service.instance.ID)
	}

	if err := scope.Order("statuses.id desc").Find(&statuses).Error; err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var resp []any
	for _, status := range statuses {
		resp = append(resp, status.serialize())
	}

	w.Header().Set("Content-Type", "application/json")
	if len(statuses) > 0 {
		w.Header().Set("Link", fmt.Sprintf("<https://%s/api/v1/timelines/public?max_id=%d>; rel=\"next\", <https://%s/api/v1/timelines/public?min_id=%d>; rel=\"prev\"", c.service.Domain(), statuses[len(statuses)-1].ID, c.service.Domain(), statuses[0].ID))
	}
	json.MarshalFull(w, resp)
}

func (c *Conversations) paginate(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		q := r.URL.Query()

		limit, _ := strconv.Atoi(q.Get("limit"))
		switch {
		case limit > 40:
			limit = 40
		case limit <= 0:
			limit = 20
		}
		db = db.Limit(limit)

		sinceID, _ := strconv.Atoi(r.URL.Query().Get("since_id"))
		if sinceID > 0 {
			db = db.Where("statuses.id > ?", sinceID)
		}
		minID, _ := strconv.Atoi(r.URL.Query().Get("min_id"))
		if minID > 0 {
			db = db.Where("statuses.id > ?", minID)
		}
		maxID, _ := strconv.Atoi(r.URL.Query().Get("max_id"))
		if maxID > 0 {
			db = db.Where("statuses.id < ?", maxID)
		}
		return db
	}
}

type conversations struct {
	db      *gorm.DB
	service *Service
}

func (c *conversations) FindConversationByURI(uri string) (*Conversation, error) {
	var conversation Conversation
	if err := c.db.Where("status.uri = ?", uri).Joins("Statuses").First(&conversation).Error; err != nil {
		return nil, err
	}
	return &conversation, nil
}

func (c *conversations) FindConversationByStatusID(id uint) (*Conversation, error) {
	var status Status
	if err := c.db.Where("id = ?", id).First(&status).Error; err != nil {
		return nil, err
	}
	var conversation Conversation
	if err := c.db.Preload("Statuses").First(&conversation, status.ConversationID).Error; err != nil {
		return &conversation, nil
	}
	return &conversation, nil
}
