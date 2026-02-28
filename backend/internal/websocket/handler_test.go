package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"social_app/internal/config"
	"social_app/internal/logger"
	accountpb "social_app/internal/proto/account"
	commonpb "social_app/internal/proto/common"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

func TestWebSocketHandshakeRejectsMissingUID(t *testing.T) {
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
	if ct := recorder.Header().Get("Content-Type"); !strings.Contains(ct, "application/x-protobuf") {
		t.Fatalf("unexpected content-type: %s", ct)
	}

	var resp accountpb.AuthResponse
	if err := proto.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ErrorCode != commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT {
		t.Fatalf("unexpected error code: %v", resp.ErrorCode)
	}
	if resp.Message == "" {
		t.Fatalf("missing message")
	}
}
