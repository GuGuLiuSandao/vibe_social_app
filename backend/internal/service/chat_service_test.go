package service

import (
	"fmt"
	"strings"
	"testing"

	"social_app/internal/db"
	"social_app/internal/logger"
	"social_app/internal/models"
	pb "social_app/internal/proto/chat"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupChatServiceTestDB(t *testing.T) *ChatService {
	t.Helper()
	logger.Init()

	gormDB, err := gorm.Open(sqlite.Open("file:chat_service_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := gormDB.AutoMigrate(
		&models.User{},
		&models.Message{},
		&models.Conversation{},
		&models.ConversationParticipant{},
		&models.Relation{},
		&models.BlockRelation{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	db.DB = gormDB
	relationService := NewRelationService(gormDB)
	return NewChatService(relationService)
}

func mustCreateChatTestUser(t *testing.T, uid uint64) models.User {
	t.Helper()
	user := models.User{
		ID:       uid + 2000000,
		UID:      uid,
		Username: fmt.Sprintf("u%d", uid),
		Email:    fmt.Sprintf("u%d@example.com", uid),
		Password: "secret12",
		Nickname: fmt.Sprintf("N%d", uid),
	}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user %d: %v", uid, err)
	}
	return user
}

func TestSendMessagePrivateLimitWithoutFollowBack(t *testing.T) {
	chatService := setupChatServiceTestDB(t)

	sender := mustCreateChatTestUser(t, 10000001)
	receiver := mustCreateChatTestUser(t, 10000002)

	if err := db.DB.Create(&models.Relation{
		UserID:   sender.UID,
		TargetID: receiver.UID,
	}).Error; err != nil {
		t.Fatalf("create one-way follow relation: %v", err)
	}

	conv, err := chatService.CreateConversation(sender.UID, &pb.CreateConversationRequest{
		Type:           pb.ConversationType_CONVERSATION_TYPE_PRIVATE,
		ParticipantIds: []uint64{receiver.UID},
	})
	if err != nil {
		t.Fatalf("create private conversation: %v", err)
	}

	for i := 1; i <= 3; i++ {
		_, err := chatService.SendMessage(sender.UID, &pb.SendMessageRequest{
			ConversationId: conv.ID,
			Content:        fmt.Sprintf("msg-%d", i),
			Type:           pb.MessageType_MESSAGE_TYPE_TEXT,
		})
		if err != nil {
			t.Fatalf("send message %d failed: %v", i, err)
		}
	}

	_, err = chatService.SendMessage(sender.UID, &pb.SendMessageRequest{
		ConversationId: conv.ID,
		Content:        "msg-4",
		Type:           pb.MessageType_MESSAGE_TYPE_TEXT,
	})
	if err == nil {
		t.Fatalf("expected message limit error on 4th message")
	}
	if !strings.Contains(err.Error(), "message limit reached") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBlocklistBlocksPrivateConversationCreation(t *testing.T) {
	chatService := setupChatServiceTestDB(t)

	creator := mustCreateChatTestUser(t, 10000011)
	target := mustCreateChatTestUser(t, 10000012)

	if err := db.DB.Create(&models.Relation{UserID: creator.UID, TargetID: target.UID}).Error; err != nil {
		t.Fatalf("create follow relation: %v", err)
	}
	if err := db.DB.Create(&models.BlockRelation{UserID: target.UID, TargetID: creator.UID}).Error; err != nil {
		t.Fatalf("create block relation: %v", err)
	}

	_, err := chatService.CreateConversation(creator.UID, &pb.CreateConversationRequest{
		Type:           pb.ConversationType_CONVERSATION_TYPE_PRIVATE,
		ParticipantIds: []uint64{target.UID},
	})
	if err == nil {
		t.Fatalf("expected blocklist error when creating private conversation")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBlocklistBlocksPrivateMessageSending(t *testing.T) {
	chatService := setupChatServiceTestDB(t)

	sender := mustCreateChatTestUser(t, 10000021)
	receiver := mustCreateChatTestUser(t, 10000022)

	if err := db.DB.Create(&models.Relation{UserID: sender.UID, TargetID: receiver.UID}).Error; err != nil {
		t.Fatalf("create follow relation: %v", err)
	}

	conv, err := chatService.CreateConversation(sender.UID, &pb.CreateConversationRequest{
		Type:           pb.ConversationType_CONVERSATION_TYPE_PRIVATE,
		ParticipantIds: []uint64{receiver.UID},
	})
	if err != nil {
		t.Fatalf("create private conversation: %v", err)
	}

	if err := db.DB.Create(&models.BlockRelation{UserID: receiver.UID, TargetID: sender.UID}).Error; err != nil {
		t.Fatalf("create block relation: %v", err)
	}

	_, err = chatService.SendMessage(sender.UID, &pb.SendMessageRequest{
		ConversationId: conv.ID,
		Content:        "blocked-msg",
		Type:           pb.MessageType_MESSAGE_TYPE_TEXT,
	})
	if err == nil {
		t.Fatalf("expected blocklist error when sending private message")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBlocklistBlocksGroupInvite(t *testing.T) {
	chatService := setupChatServiceTestDB(t)

	creator := mustCreateChatTestUser(t, 10000031)
	friendA := mustCreateChatTestUser(t, 10000032)
	friendB := mustCreateChatTestUser(t, 10000033)

	if err := db.DB.Create(&models.BlockRelation{UserID: friendB.UID, TargetID: creator.UID}).Error; err != nil {
		t.Fatalf("create block relation: %v", err)
	}

	_, err := chatService.CreateConversation(creator.UID, &pb.CreateConversationRequest{
		Type:           pb.ConversationType_CONVERSATION_TYPE_GROUP,
		ParticipantIds: []uint64{friendA.UID, friendB.UID},
		Name:           "blocked-group",
	})
	if err == nil {
		t.Fatalf("expected blocklist error when creating group conversation")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBlocklistBlocksFollowWhileBlocked(t *testing.T) {
	chatService := setupChatServiceTestDB(t)
	_ = chatService

	relationService := NewRelationService(db.DB)
	userA := mustCreateChatTestUser(t, 10000041)
	userB := mustCreateChatTestUser(t, 10000042)

	if err := relationService.Block(userA.UID, userB.UID); err != nil {
		t.Fatalf("block user: %v", err)
	}

	err := relationService.Follow(userB.UID, userA.UID)
	if err == nil {
		t.Fatalf("expected follow to fail while blocked")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("unexpected error: %v", err)
	}
}
