package m

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-json-experiment/json"
	"github.com/gorilla/mux"
)

type Contexts struct {
	service *Service
}

func (c *Contexts) Show(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	_, err := c.service.tokens().FindByAccessToken(accessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	conv, err := c.service.conversations().FindConversationByStatusID(uint64(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// load conversation statuses
	var statuses []Status
	if err := c.service.db.Where("conversation_id = ?", conv.ID).Joins("Account").Find(&statuses).Error; err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	ancestors, decentants := thread(uint64(id), statuses)
	w.Header().Set("Content-Type", "application/activity+json")
	json.MarshalFull(w, map[string]interface{}{
		"ancestors": func() []interface{} {
			var a []interface{}
			for _, s := range ancestors {
				a = append(a, s.serialize())
			}
			return a
		}(),
		"descendants": func() []interface{} {
			var a []interface{}
			for _, s := range decentants {
				a = append(a, s.serialize())
			}
			return a
		}(),
	})
}

// thread sorts statuses into a tree, it returns the statuses
// preceeding id, and statuses following id.
func thread(id uint64, statuses []Status) ([]*Status, []*Status) {
	type link struct {
		parent   *link
		status   *Status
		children []*link
	}
	ids := make(map[uint64]*link)
	for i := range statuses {
		ids[statuses[i].ID] = &link{status: &statuses[i]}
	}

	for _, l := range ids {
		if l.status.InReplyToID != nil {
			parent := ids[*l.status.InReplyToID]
			parent.children = append(parent.children, l)
			l.parent = parent
		}
	}

	var ancestors []*Status
	var l = ids[id].parent
	for l != nil {
		ancestors = append(ancestors, l.status)
		l = l.parent
	}
	reverse(ancestors)

	var decendants []*Status
	var walk func(*link)
	walk = func(l *link) {
		for _, c := range l.children {
			decendants = append(decendants, c.status)
			walk(c)
		}
	}
	walk(ids[id])
	return ancestors, decendants
}

func reverse[T any](a []T) {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
}
