package websocket

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"social_app/internal/config"
	"social_app/internal/db"
	"social_app/internal/logger"
	"social_app/internal/models"
	proto "social_app/internal/proto"
	chatpb "social_app/internal/proto/chat"

	"github.com/gin-gonic/gin"
	goproto "google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWebSocketHandshakeRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Init()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req, _ := http.NewRequest(http.MethodGet, "/ws", nil)
	c.Request = req

	h := NewHandler(NewServer(), &config.Config{JWTSecret: "test-secret"})
	h.HandleWebSocket(c)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if ct := recorder.Header().Get("Content-Type"); ct != "" {
		t.Fatalf("unexpected content-type: %s", ct)
	}
	if body := recorder.Body.String(); body != "" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func setupWebSocketTestDB(t *testing.T) {
	t.Helper()
	logger.Init()

	gormDB, err := gorm.Open(sqlite.Open("file:websocket_test?mode=memory&cache=shared"), &gorm.Config{})
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
}

func mustCreateTestUser(t *testing.T, uid uint64) models.User {
	t.Helper()
	user := models.User{
		ID:       uid + 1000000,
		UID:      uid,
		Username: fmt.Sprintf("user_%d", uid),
		Email:    fmt.Sprintf("u%d@example.com", uid),
		Password: "secret12",
		Nickname: fmt.Sprintf("U%d", uid),
	}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user %d: %v", uid, err)
	}
	return user
}

func TestCreateConversationPushesToOtherParticipant(t *testing.T) {
	setupWebSocketTestDB(t)
	gin.SetMode(gin.TestMode)

	creator := mustCreateTestUser(t, 10000001)
	peer := mustCreateTestUser(t, 10000002)

	if err := db.DB.Create(&models.Relation{
		UserID:   creator.UID,
		TargetID: peer.UID,
	}).Error; err != nil {
		t.Fatalf("create relation: %v", err)
	}

	server := NewServer()
	creatorClient := &Client{
		ID:     uint(creator.UID),
		Send:   make(chan []byte, 4),
		Server: server,
	}
	peerClient := &Client{
		ID:     uint(peer.UID),
		Send:   make(chan []byte, 4),
		Server: server,
	}
	server.Clients[creatorClient.ID] = creatorClient
	server.Clients[peerClient.ID] = peerClient

	wsReq := &proto.WsMessage{
		RequestId: 1,
		Type:      proto.WsMessageType_WS_TYPE_CHAT_CREATE_CONVERSATION,
		Timestamp: time.Now().UnixMilli(),
		Payload: &proto.WsMessage_Chat{
			Chat: &chatpb.ChatPayload{
				Payload: &chatpb.ChatPayload_CreateConversation{
					CreateConversation: &chatpb.CreateConversationRequest{
						Type:           chatpb.ConversationType_CONVERSATION_TYPE_PRIVATE,
						ParticipantIds: []uint64{peer.UID},
					},
				},
			},
		},
	}
	data, err := goproto.Marshal(wsReq)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	server.HandleMessage(creatorClient, data)

	var creatorRespRaw []byte
	select {
	case creatorRespRaw = <-creatorClient.Send:
	case <-time.After(time.Second):
		t.Fatalf("creator did not receive create conversation response")
	}

	var creatorResp proto.WsMessage
	if err := goproto.Unmarshal(creatorRespRaw, &creatorResp); err != nil {
		t.Fatalf("unmarshal creator response: %v", err)
	}
	if creatorResp.Type != proto.WsMessageType_WS_TYPE_CHAT_CREATE_CONVERSATION_RESPONSE {
		t.Fatalf("unexpected creator response type: %v", creatorResp.Type)
	}
	createdConv := creatorResp.GetChat().GetCreateConversationResponse().GetConversation()
	if createdConv == nil || createdConv.Id == 0 {
		t.Fatalf("missing created conversation in creator response")
	}

	var peerPushRaw []byte
	select {
	case peerPushRaw = <-peerClient.Send:
	case <-time.After(time.Second):
		t.Fatalf("peer did not receive conversation_push")
	}

	var peerPush proto.WsMessage
	if err := goproto.Unmarshal(peerPushRaw, &peerPush); err != nil {
		t.Fatalf("unmarshal peer push: %v", err)
	}
	if peerPush.Type != proto.WsMessageType_WS_TYPE_CHAT_CONVERSATION_PUSH {
		t.Fatalf("unexpected peer push type: %v", peerPush.Type)
	}
	pushedConv := peerPush.GetChat().GetConversationPush().GetConversation()
	if pushedConv == nil || pushedConv.Id != createdConv.Id {
		t.Fatalf("conversation push mismatch, got=%v want=%v", pushedConv.GetId(), createdConv.Id)
	}
}

func TestWebSocketHandshakeRejectsInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.Init()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req, _ := http.NewRequest(http.MethodGet, "/ws?token=invalid-token", nil)
	c.Request = req

	h := NewHandler(NewServer(), &config.Config{JWTSecret: "test-secret"})
	h.HandleWebSocket(c)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if body := recorder.Body.String(); body != "" {
		t.Fatalf("unexpected body: %q", body)
	}
}
