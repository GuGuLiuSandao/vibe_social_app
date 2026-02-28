package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint64         `json:"id" gorm:"primaryKey;autoIncrement:false"`
	UID       uint64         `json:"uid" gorm:"uniqueIndex;not null"`
	Username  string         `json:"username" gorm:"uniqueIndex;size:50;not null"`
	Email     string         `json:"email" gorm:"uniqueIndex;size:100;not null"`
	Password  string         `json:"-" gorm:"not null"`
	Nickname  string         `json:"nickname" gorm:"size:50"`
	Avatar    string         `json:"avatar"`
	Bio       string         `json:"bio" gorm:"size:200"`
	Gender    string         `json:"gender" gorm:"size:10"`
	Birthday  *time.Time     `json:"birthday"`
	Location  string         `json:"location" gorm:"size:100"`
	Status    int            `json:"status" gorm:"default:1"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
