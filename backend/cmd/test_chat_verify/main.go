package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	ws "social_app/internal/proto"
	"social_app/internal/proto/account"
	pb "social_app/internal/proto/chat"
)

const (
	apiURL = "http://localhost:8080/api/v1"
	wsURL  = "ws://localhost:8080/ws"
)

func registerUser(username string) (uint64, string) {
	req := &account.RegisterRequest{
		Username: username,
		Email:    username + "@example.com",
		Password: "password123",
	}
	data, _ := proto.Marshal(req)
	resp, err := http.Post(apiURL+"/auth/register", "application/x-protobuf", bytes.NewReader(data))
	if err != nil {
		log.Fatalf("Register failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var authResp account.RegisterResponse
	if err := proto.Unmarshal(body, &authResp); err != nil {
		log.Fatalf("Unmarshal failed: %v", err)
	}

	if authResp.User == nil {
		log.Fatalf("Register failed: %s", authResp.Message)
	}

	return authResp.User.Id, authResp.Token
}

func main() {
	log.SetFlags(0)

	// 1. Register two users
	ts := time.Now().Unix()
	userA_Name := fmt.Sprintf("userA_%d", ts)
	userB_Name := fmt.Sprintf("userB_%d", ts)

	uidA, tokenA := registerUser(userA_Name)
	uidB, _ := registerUser(userB_Name)

	log.Printf("User A: %d (%s)", uidA, userA_Name)
	log.Printf("User B: %d (%s)", uidB, userB_Name)

	if uidA > 100000000 || uidB > 100000000 {
		log.Fatalf("UIDs are too large! Expected sequence IDs, got Snowflake IDs.")
	}

	// 2. Connect WS as User A
	u, _ := url.Parse(wsURL)
	q := u.Query()
	q.Set("uid", fmt.Sprintf("%d", uidA))
	q.Set("token", tokenA)
	u.RawQuery = q.Encode()

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	// 3. Create Conversation with User B
	createConvReq := &ws.WsMessage{
		RequestId: time.Now().UnixNano(),
		Type:      ws.WsMessageType_WS_TYPE_CHAT_CREATE_CONVERSATION,
		Timestamp: time.Now().UnixMilli(),
		Payload: &ws.WsMessage_Chat{
			Chat: &pb.ChatPayload{
				Payload: &pb.ChatPayload_CreateConversation{
					CreateConversation: &pb.CreateConversationRequest{
						Type:           pb.ConversationType_CONVERSATION_TYPE_PRIVATE,
						ParticipantIds: []uint64{uidB},
					},
				},
			},
		},
	}

	send(c, createConvReq)

	// 4. Wait for CreateConversationResponse
	var convID uint64
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			var wsMsg ws.WsMessage
			if err := proto.Unmarshal(message, &wsMsg); err != nil {
				continue
			}

			if chat := wsMsg.GetChat(); chat != nil {
				// Handle CreateConversationResponse
				if resp := chat.GetCreateConversationResponse(); resp != nil {
					log.Printf("✅ Conversation Created: %d", resp.Conversation.Id)
					convID = resp.Conversation.Id

					// 5. Send Message
					sendMsg := &ws.WsMessage{
						RequestId: time.Now().UnixNano(),
						Type:      ws.WsMessageType_WS_TYPE_CHAT_SEND_MESSAGE,
						Timestamp: time.Now().UnixMilli(),
						Payload: &ws.WsMessage_Chat{
							Chat: &pb.ChatPayload{
								Payload: &pb.ChatPayload_SendMessage{
									SendMessage: &pb.SendMessageRequest{
										ConversationId: convID,
										Type:           pb.MessageType_MESSAGE_TYPE_TEXT,
										Content:        "Hello",
									},
								},
							},
						},
					}
					send(c, sendMsg)
				}

				// Handle MessagePush (or SendMessageResponse)
				// The server sends SendMessageResponse to sender, and MessagePush to others?
				// Actually handler.go sends SendMessageResponse?
				// Let's check handler.go:
				// It sends back the result of processMessage, which calls handleSendMessage.
				// handleSendMessage returns a WsMessage with WS_TYPE_CHAT_MESSAGE_PUSH.
				// Wait, handleSendMessage returns WS_TYPE_CHAT_MESSAGE_PUSH?
				// Yes: Type: proto.WsMessageType_WS_TYPE_CHAT_MESSAGE_PUSH

				if push := chat.GetMessagePush(); push != nil {
					msg := push.Message
					log.Printf("📩 Received Message: SenderID=%d, Sender.ID=%d", msg.SenderId, msg.Sender.Id)

					if msg.SenderId != uidA {
						log.Fatalf("❌ SenderID mismatch: expected %d, got %d", uidA, msg.SenderId)
					}
					if msg.Sender.Id != uidA {
						log.Fatalf("❌ Sender.ID mismatch: expected %d, got %d", uidA, msg.Sender.Id)
					}

					log.Println("✅ Verification SUCCESS!")
					os.Exit(0)
				}
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Fatal("Timeout waiting for response")
	}
}

func send(c *websocket.Conn, msg *ws.WsMessage) {
	data, err := proto.Marshal(msg)
	if err != nil {
		log.Fatal("marshal:", err)
	}
	if err := c.WriteMessage(websocket.BinaryMessage, data); err != nil {
		log.Fatal("write:", err)
	}
}
