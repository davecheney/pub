package mastodon

import (
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID                int       `db:"id"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
	Email             string    `db:"email"`
	EncryptedPassword []byte    `db:"encrypted_password"`
}

func (u *User) comparePassword(password string) bool {
	if err := bcrypt.CompareHashAndPassword(u.EncryptedPassword, []byte(password)); err != nil {
		return false
	}
	return true
}

type users struct {
	db *sqlx.DB
}

func (u *users) findByEmail(email string) (*User, error) {
	user := &User{}
	err := u.db.QueryRowx(`SELECT * FROM users WHERE email = ?`, email).StructScan(user)
	return user, err
}

func (u *users) findByID(id int) (*User, error) {
	user := &User{}
	err := u.db.QueryRowx(`SELECT * FROM users WHERE id = ?`, id).StructScan(user)
	return user, err
}
