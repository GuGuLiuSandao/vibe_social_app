package models

import (
	"time"
)

// ConversationType 定义会话类型 (与 Proto 定义对应)
type ConversationType int

const (
	ConversationTypePrivate ConversationType = 1
	ConversationTypeGroup   ConversationType = 2
)

const (
	GroupKindOfficial      = "official"
	GroupKindPlayerCreated = "player_created"

	GroupJoinModePrivate  = "private"
	GroupJoinModeApproval = "approval"
	GroupJoinModePublic   = "public"

	GroupStatusActive    = "active"
	GroupStatusDissolved = "dissolved"

	GroupRoleOwner  = "owner"
	GroupRoleAdmin  = "admin"
	GroupRoleMember = "member"

	GroupJoinRequestStatusPending   = "pending"
	GroupJoinRequestStatusApproved  = "approved"
	GroupJoinRequestStatusRejected  = "rejected"
	GroupJoinRequestStatusCancelled = "cancelled"

	GroupInvitationStatusPending   = "pending"
	GroupInvitationStatusAccepted  = "accepted"
	GroupInvitationStatusRejected  = "rejected"
	GroupInvitationStatusCancelled = "cancelled"
	GroupInvitationStatusExpired   = "expired"
)

// Conversation 会话基础表
// 存储会话的公共属性，不包含特定成员的状态（如未读数）
type Conversation struct {
	ID                    uint64           `gorm:"primaryKey;autoIncrement"`
	Type                  ConversationType `gorm:"not null;index"` // 1:单聊, 2:群聊
	Name                  string           `gorm:"size:100"`       // 群聊名称，单聊通常为空
	Avatar                string           `gorm:"size:255"`       // 群聊头像
	OwnerID               uint64           `gorm:"index"`          // 群主ID (仅群聊有效)
	Description           string           `gorm:"type:text"`
	Announcement          string           `gorm:"type:text"`
	AnnouncementUpdatedBy *uint64          `gorm:"index"`
	AnnouncementUpdatedAt *time.Time
	GroupKind             string     `gorm:"size:32;default:'player_created';index"`
	JoinMode              string     `gorm:"size:32;default:'private';index"`
	Status                string     `gorm:"size:32;default:'active';index"`
	DissolvedAt           *time.Time `gorm:"index"`
	DissolvedBy           *uint64    `gorm:"index"`
	CreatedAt             time.Time  `gorm:"not null"`
	UpdatedAt             time.Time  `gorm:"not null;index"` // 用于会话列表排序

	// 关联
	LastMessageID *uint64  `gorm:"index"`     // 最新一条消息的全局ID
	LastLocalID   uint64   `gorm:"default:0"` // 该会话当前的 LocalID 最大值
	LastMessage   *Message `gorm:"foreignKey:LastMessageID"`
}

// ConversationParticipant 会话成员表 (关键)
// 解决了多对多关系，同时存储每个成员在会话中的个性化状态
type ConversationParticipant struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement"`
	ConversationID uint64 `gorm:"not null;uniqueIndex:idx_conv_user"`
	UserID         uint64 `gorm:"not null;uniqueIndex:idx_conv_user"`

	// 个性化设置
	DisplayName string `gorm:"size:50"`                  // 在群里的昵称
	Role        string `gorm:"size:20;default:'member'"` // owner, admin, member
	Muted       bool   `gorm:"default:false"`            // 是否免打扰

	// 消息状态
	UnreadCount     int       `gorm:"default:0"` // 计算方式: Conversation.LastLocalID - LastReadLocalID
	LastReadLocalID uint64    `gorm:"default:0"` // 用户在该会话中已读到的最后一条消息的 LocalID
	JoinedAt        time.Time `gorm:"autoCreateTime"`

	// 关联
	Conversation Conversation `gorm:"foreignKey:ConversationID"`
	User         User         `gorm:"foreignKey:UserID;references:UID"`
}

// Message 消息表
type Message struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement"`            // 全局唯一ID
	ConversationID uint64    `gorm:"not null;uniqueIndex:idx_conv_local"` // 组合唯一索引
	LocalID        uint64    `gorm:"not null;uniqueIndex:idx_conv_local"` // 会话内自增ID，从1开始
	SenderID       uint64    `gorm:"not null;index"`
	Content        string    `gorm:"type:text;not null"`
	Type           int       `gorm:"default:1"` // 1:text, 2:image, 3:system (与 Proto 对应)
	CreatedAt      time.Time `gorm:"not null;index:idx_created"`

	// 关联
	Sender       User         `gorm:"foreignKey:SenderID;references:UID"`
	Conversation Conversation `gorm:"foreignKey:ConversationID"`
}

type GroupJoinRequest struct {
	ID             uint64  `gorm:"primaryKey;autoIncrement"`
	ConversationID uint64  `gorm:"not null;index:idx_group_join_request_conv_user_status"`
	ApplicantID    uint64  `gorm:"not null;index:idx_group_join_request_conv_user_status"`
	Status         string  `gorm:"size:32;not null;default:'pending';index:idx_group_join_request_conv_user_status"`
	Message        string  `gorm:"type:text"`
	ReviewedBy     *uint64 `gorm:"index"`
	ReviewedAt     *time.Time
	CreatedAt      time.Time `gorm:"not null;index"`
	UpdatedAt      time.Time `gorm:"not null"`
}

type GroupInvitation struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement"`
	ConversationID uint64 `gorm:"not null;index:idx_group_invitation_conv_user_status"`
	InviterID      uint64 `gorm:"not null;index"`
	InviteeID      uint64 `gorm:"not null;index:idx_group_invitation_conv_user_status"`
	Status         string `gorm:"size:32;not null;default:'pending';index:idx_group_invitation_conv_user_status"`
	RespondedAt    *time.Time
	CreatedAt      time.Time `gorm:"not null;index"`
	UpdatedAt      time.Time `gorm:"not null"`
}
