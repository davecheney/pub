package m

import "time"

type Token struct {
	ID                uint `gorm:"primarykey"`
	CreatedAt         time.Time
	AccountID         uint
	Account           *Account
	ApplicationID     uint
	AccessToken       string
	TokenType         string
	Scope             string
	AuthorizationCode string
}
