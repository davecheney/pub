package models

type Peer struct {
	Domain string `gorm:"primary_key;size:64"`
}
