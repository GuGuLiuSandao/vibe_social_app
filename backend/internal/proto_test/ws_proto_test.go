package proto_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWsProtoIsEnvelopeOnly(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	data, err := os.ReadFile(filepath.Join(root, "proto", "ws.proto"))
	if err != nil {
		t.Fatalf("read ws.proto: %v", err)
	}
	source := string(data)
	disallowed := []string{
		"message Message",
		"message SendMessageRequest",
		"message GetMessageListRequest",
		"message MarkAsReadRequest",
		"message MessagePush",
		"message Conversation",
	}
	for _, token := range disallowed {
		if strings.Contains(source, token) {
			t.Fatalf("ws.proto contains business type: %s", token)
		}
	}
}

func TestWsProtoUsesAccountAndChatPayloads(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	wsBytes, err := os.ReadFile(filepath.Join(root, "proto", "ws.proto"))
	if err != nil {
		t.Fatalf("read ws.proto: %v", err)
	}
	accountBytes, err := os.ReadFile(filepath.Join(root, "proto", "account", "account.proto"))
	if err != nil {
		t.Fatalf("read account.proto: %v", err)
	}
	chatBytes, err := os.ReadFile(filepath.Join(root, "proto", "chat", "chat.proto"))
	if err != nil {
		t.Fatalf("read chat.proto: %v", err)
	}

	wsSource := string(wsBytes)
	accountSource := string(accountBytes)
	chatSource := string(chatBytes)

	requiredInWs := []string{
		`import "account/account.proto";`,
		`import "chat/chat.proto";`,
		"social.account.AccountPayload account = 10;",
		"social.chat.ChatPayload chat = 11;",
	}
	for _, token := range requiredInWs {
		if !strings.Contains(wsSource, token) {
			t.Fatalf("ws.pb missing token: %s", token)
		}
	}

	requiredInAccount := []string{
		"message AccountPayload",
		"Ping ping = 1;",
		"AuthRequest auth = 3;",
	}
	for _, token := range requiredInAccount {
		if !strings.Contains(accountSource, token) {
			t.Fatalf("account.pb missing token: %s", token)
		}
	}

	requiredInChat := []string{
		"message ChatPayload",
		"message GetConversationListRequest",
		"CreateConversationRequest create_conversation = 9;",
	}
	for _, token := range requiredInChat {
		if !strings.Contains(chatSource, token) {
			t.Fatalf("chat.pb missing token: %s", token)
		}
	}
}
