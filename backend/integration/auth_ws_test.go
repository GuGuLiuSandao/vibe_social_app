//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	redisclient "github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"

	"social_app/internal/auth"
	"social_app/internal/config"
	rootpb "social_app/internal/proto"
	accountpb "social_app/internal/proto/account"
	commonpb "social_app/internal/proto/common"
)

func postProtobuf(t *testing.T, url string, request, response proto.Message) {
	t.Helper()
	body, err := proto.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	httpResponse, err := http.Post(url, "application/x-protobuf", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer httpResponse.Body.Close()
	payload, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		t.Fatal(err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		t.Fatalf("POST %s status = %d, body = %x", url, httpResponse.StatusCode, payload)
	}
	if got := httpResponse.Header.Get("Content-Type"); got != "application/x-protobuf" {
		t.Fatalf("POST %s Content-Type = %q", url, got)
	}
	if err := proto.Unmarshal(payload, response); err != nil {
		t.Fatalf("decode %s response: %v", url, err)
	}
}

func endpoints(t *testing.T) (string, string, string) {
	t.Helper()
	baseURL := os.Getenv("INTEGRATION_BASE_URL")
	wsURL := os.Getenv("INTEGRATION_WS_URL")
	redisAddr := os.Getenv("INTEGRATION_REDIS_ADDR")
	if baseURL == "" || wsURL == "" || redisAddr == "" {
		t.Fatal("integration endpoint environment is incomplete")
	}
	return baseURL, wsURL, redisAddr
}

func registerAndLogin(t *testing.T, baseURL, label string) (*accountpb.RegisterResponse, *accountpb.LoginResponse) {
	t.Helper()
	suffix := fmt.Sprintf("%s_%d", label, time.Now().UnixNano())
	username := "dlq_" + suffix
	email := "dlq_" + suffix + "@example.test"
	password := "test-password-123"
	register := &accountpb.RegisterResponse{}
	postProtobuf(t, baseURL+"/api/v1/auth/register", &accountpb.RegisterRequest{Username: username, Email: email, Password: password}, register)
	if register.ErrorCode != commonpb.ErrorCode_ERROR_CODE_OK || register.Message != "ok" || register.User == nil || register.Token == "" {
		t.Fatalf("registration response = %#v", register)
	}
	if register.User.Username != username || register.User.Email != email || register.User.Nickname != username || register.User.Id == 0 {
		t.Fatalf("registration identity = %#v", register.User)
	}
	login := &accountpb.LoginResponse{}
	postProtobuf(t, baseURL+"/api/v1/auth/login", &accountpb.LoginRequest{Email: email, Password: password}, login)
	if login.ErrorCode != commonpb.ErrorCode_ERROR_CODE_OK || login.Message != "ok" || login.User == nil || login.Token == "" {
		t.Fatalf("login response = %#v", login)
	}
	if login.User.Id != register.User.Id || login.User.Username != username || login.User.Nickname != username || login.User.Email != email {
		t.Fatalf("login identity = %#v, want registration identity %#v", login.User, register.User)
	}
	for name, token := range map[string]string{"register": register.Token, "login": login.Token} {
		claims, err := auth.ParseToken(token, &config.Config{JWTSecret: "integration-only-jwt-secret"})
		if err != nil || claims.UserID != register.User.Id || claims.Username != username {
			t.Fatalf("%s token claims = %#v, %v", name, claims, err)
		}
	}
	return register, login
}

func waitForMembership(t *testing.T, client *redisclient.Client, uid uint64, want bool) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		got, err := client.SIsMember(ctx, "online:users", uid).Result()
		if err == nil && got == want {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("Redis membership for UID %d did not become %v", uid, want)
}

func assertWebSocketUnauthorized(t *testing.T, url string) {
	t.Helper()
	connection, response, err := websocket.DefaultDialer.Dial(url, nil)
	if connection != nil {
		connection.Close()
	}
	if response != nil {
		defer response.Body.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("WebSocket dial %s = response %#v, error %v; want HTTP 401", url, response, err)
	}
}

func TestDLQ_TC_011_016_AUTH_HTTP_001_RegisterLogin(t *testing.T) {
	baseURL, _, _ := endpoints(t)
	registerAndLogin(t, baseURL, "auth")
}

func TestDLQ_TC_017_WS_HTTP_001_AuthenticatedPingPong(t *testing.T) {
	baseURL, wsURL, redisAddr := endpoints(t)
	t.Run("missing-token", func(t *testing.T) { assertWebSocketUnauthorized(t, wsURL) })
	t.Run("invalid-token", func(t *testing.T) { assertWebSocketUnauthorized(t, wsURL+"?token=invalid") })
	_, login := registerAndLogin(t, baseURL, "ws")

	redis := redisclient.NewClient(&redisclient.Options{Addr: redisAddr})
	defer redis.Close()
	if err := redis.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("Redis ping: %v", err)
	}

	connection, response, err := websocket.DefaultDialer.Dial(wsURL+"?uid="+fmt.Sprint(login.User.Id)+"&token="+login.Token, nil)
	if err != nil {
		if response != nil {
			t.Fatalf("WebSocket upgrade status %d: %v", response.StatusCode, err)
		}
		t.Fatalf("WebSocket dial: %v", err)
	}
	waitForMembership(t, redis, login.User.Id, true)

	ping := &rootpb.WsMessage{RequestId: 9001, Type: rootpb.WsMessageType_WS_TYPE_PING, Timestamp: time.Now().UnixMilli()}
	payload, err := proto.Marshal(ping)
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.WriteMessage(websocket.BinaryMessage, payload); err != nil {
		t.Fatal(err)
	}
	connection.SetReadDeadline(time.Now().Add(5 * time.Second))
	messageType, payload, err := connection.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("WebSocket message type = %d, want binary", messageType)
	}
	pong := &rootpb.WsMessage{}
	if err := proto.Unmarshal(payload, pong); err != nil {
		t.Fatal(err)
	}
	if pong.Type != rootpb.WsMessageType_WS_TYPE_PONG || pong.RequestId != ping.RequestId || pong.Timestamp <= 0 {
		t.Fatalf("pong = %#v", pong)
	}
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	waitForMembership(t, redis, login.User.Id, false)
}
