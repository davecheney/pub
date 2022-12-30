package models

import (
	"time"

	"github.com/davecheney/m/internal/snowflake"
	"gorm.io/gorm"
)

// An Instance is an ActivityPub domain managed by this server.
// An Instance has many InstanceRules.
// An Instance has one Admin Account.
type Instance struct {
	snowflake.ID     `gorm:"primarykey;autoIncrement:false"`
	UpdatedAt        time.Time
	Domain           string `gorm:"size:64;uniqueIndex"`
	AdminID          *snowflake.ID
	Admin            *Account
	SourceURL        string
	Title            string `gorm:"size:64"`
	ShortDescription string
	Description      string
	Thumbnail        string `gorm:"size:64"`
	AccountsCount    int    `gorm:"default:0;not null"`
	StatusesCount    int    `gorm:"default:0;not null"`

	DomainsCount int `gorm:"-"`

	Rules []InstanceRule
}

func (i *Instance) AfterCreate(tx *gorm.DB) error {
	return i.updateAccountsCount(tx)
}

func (i *Instance) updateAccountsCount(tx *gorm.DB) error {
	var count int64
	err := tx.Model(&Account{}).Where("instance_id = ?", i.ID).Count(&count).Error
	if err != nil {
		return err
	}
	return tx.Model(i).Update("accounts_count", count).Error
}

func (i *Instance) updateStatusesCount(tx *gorm.DB) error {
	var count int64
	err := tx.Model(&Status{}).Joins("Account").Where("instance_id = ?", i.ID).Count(&count).Error
	if err != nil {
		return err
	}
	return tx.Model(i).Update("statuses_count", count).Error
}

type InstanceRule struct {
	ID         uint32 `gorm:"primarykey"`
	InstanceID uint64
	Text       string
}
