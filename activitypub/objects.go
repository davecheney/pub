package activitypub

import (
	"encoding/json"
	"errors"
	"time"
)

// Object is an ActivityPub Object.
// https://www.w3.org/TR/activitystreams-vocabulary/#dfn-object
type Object struct {
	// Type is the type of the Object.
	Type string `json:"type"`
}

// HashTag is an ActivityStreams Hashtag
type HashTag struct {
	Type string `json:"type"`
	Href string `json:"href"`
	Name string `json:"name"`
}

// Image is an ActivityStreams Image
type Image struct {
	Type      string `json:"type"`
	MediaType string `json:"mediaType"`
	URL       string `json:"url"`
}

type Attachment struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Status struct {
	// Type is the type of the Status.
	Type string `json:"type"`
	// The Status's unique global identifier.
	ID string `json:"id"`

	AttributedTo string    `json:"attributedTo"`
	InReplyTo    string    `json:"inReplyTo"`
	Published    time.Time `json:"published"`
	Updated      time.Time `json:"updated"`

	To StringOrArray `json:"to"`
	CC StringOrArray `json:"cc"`

	Sensitive   bool          `json:"sensitive"`
	Summary     string        `json:"summary"`
	Content     string        `json:"content"`
	Attachments []interface{} `json:"attachment"`
	Tags        []HashTag     `json:"tag"`

	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	OneOf     []Option  `json:"oneOf"`
}

type StringOrArray []string

func (s *StringOrArray) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		*s = []string{str}
		return nil
	}

	var arr []string
	if err := json.Unmarshal(b, &arr); err == nil {
		*s = arr
		return nil
	}

	return errors.New("invalid string or array")
}

type Option struct {
	Type    string     `json:"type"`
	Name    string     `json:"name"`
	Replies Collection `json:"replies"`
}

type Collection struct {
	Type       string `json:"type"`
	TotalItems int    `json:"totalItems"`
}

// Activity is an ActivityStreams Activity.
// https://www.w3.org/TR/activitystreams-core/#activities
type Activity struct {
	// Type is the type of the Activity.
	Type string `json:"type"`
	// The Activity's unique global identifier.
	ID string `json:"id"`
	// Object is the Object that the Activity is acting upon.
	Object any `json:"object"`
	// Actor is the Actor that performed the Activity.
	Actor any `json:"actor"`
	// Target is the Object that the Activity is directed at.
	Target string `json:"target"`

	Published time.Time `json:"published"`
	Updated   time.Time `json:"updated"`
}
