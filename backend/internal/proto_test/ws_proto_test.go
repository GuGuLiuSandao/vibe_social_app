package proto_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWsProtoIsEnvelopeOnly(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	data, err := os.ReadFile(filepath.Join(root, "proto", "ws.pb"))
	if err != nil {
		t.Fatalf("read ws.pb: %v", err)
	}
	source := string(data)
	disallowed := []string{
		"message Message",
		"message SendMessageRequest",
		"message GetMessagesRequest",
		"message MarkAsReadRequest",
		"message MessagePush",
		"message CommonResponse",
		"enum MessageStatus",
		"enum Status",
	}
	for _, token := range disallowed {
		if strings.Contains(source, token) {
			t.Fatalf("ws.pb contains business type: %s", token)
		}
	}
}

func TestWsProtoUsesAccountAndChatPayloads(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	wsBytes, err := os.ReadFile(filepath.Join(root, "proto", "ws.pb"))
	if err != nil {
		t.Fatalf("read ws.pb: %v", err)
	}
	accountBytes, err := os.ReadFile(filepath.Join(root, "proto", "account", "account.pb"))
	if err != nil {
		t.Fatalf("read account.pb: %v", err)
	}
	chatBytes, err := os.ReadFile(filepath.Join(root, "proto", "chat", "chat.pb"))
	if err != nil {
		t.Fatalf("read chat.pb: %v", err)
	}

	wsSource := string(wsBytes)
	accountSource := string(accountBytes)
	chatSource := string(chatBytes)

	requiredInWs := []string{
		"AccountPayload account",
		"ChatPayload chat",
	}
	for _, token := range requiredInWs {
		if !strings.Contains(wsSource, token) {
			t.Fatalf("ws.pb missing token: %s", token)
		}
	}

	requiredInAccount := []string{
		"message AccountPayload",
		"message Ping",
		"message Pong",
	}
	for _, token := range requiredInAccount {
		if !strings.Contains(accountSource, token) {
			t.Fatalf("account.pb missing token: %s", token)
		}
	}

	requiredInChat := []string{
		"message ChatPayload",
		"message GetConversationsRequest",
	}
	for _, token := range requiredInChat {
		if !strings.Contains(chatSource, token) {
			t.Fatalf("chat.pb missing token: %s", token)
		}
	}
}
