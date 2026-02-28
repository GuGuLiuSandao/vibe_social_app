package models

import "time"

// Relation represents a unidirectional follow relationship
// UserID follows TargetID
type Relation struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	UserID    uint64    `gorm:"not null;uniqueIndex:idx_user_target"` // Follower (Me)
	TargetID  uint64    `gorm:"not null;uniqueIndex:idx_user_target"` // Followee (Them)
	CreatedAt time.Time `gorm:"autoCreateTime"`

	// Associations
	User   User `gorm:"foreignKey:UserID;references:UID"`
	Target User `gorm:"foreignKey:TargetID;references:UID"`
}
