package m

import (
	"time"
)

type Webfinger struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	ActorID   uint64
	Webfinger struct {
		Subject string   `json:"subject"`
		Aliases []string `json:"aliases"`
		Links   []struct {
			Rel      string `json:"rel"`
			Type     string `json:"type"`
			Href     string `json:"href"`
			Template string `json:"template"`
		} `json:"links"`
	} `gorm:"serializer:json"`
}
