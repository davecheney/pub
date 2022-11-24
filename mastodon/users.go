package mastodon

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email             string
	EncryptedPassword []byte
}

func (u *User) comparePassword(password string) bool {
	if err := bcrypt.CompareHashAndPassword(u.EncryptedPassword, []byte(password)); err != nil {
		return false
	}
	return true
}

type users struct {
	db *gorm.DB
}

func (u *users) findByEmail(email string) (*User, error) {
	user := &User{}
	result := u.db.Where("email = ?", email).First(user)
	return user, result.Error
}

func (u *users) findByID(id int) (*User, error) {
	user := &User{}
	result := u.db.First(user, id)
	return user, result.Error
}
