package mastodon

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email             string
	EncryptedPassword []byte
	AccountID         uint
	Account           Account
}

func (u *User) comparePassword(password string) bool {
	if err := bcrypt.CompareHashAndPassword(u.EncryptedPassword, []byte(password)); err != nil {
		return false
	}
	return true
}
