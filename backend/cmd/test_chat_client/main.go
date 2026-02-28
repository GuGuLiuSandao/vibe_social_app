package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	ws "social_app/internal/proto"
	pb "social_app/internal/proto/chat"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var uid = flag.Uint64("uid", 20000001, "user id")
var targetUid = flag.Uint64("target", 20000002, "target user id")

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws", RawQuery: fmt.Sprintf("uid=%d", *uid)}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	// Variables to store state
	var conversationID uint64

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
				log.Printf("unmarshal error: %v", err)
				continue
			}

			// Log minimal info
			// log.Printf("recv: type=%v reqId=%d", wsMsg.Type, wsMsg.RequestId)

			if chat := wsMsg.GetChat(); chat != nil {
				if resp := chat.GetCreateConversationResponse(); resp != nil {
					log.Printf("✅ CreateConversationResponse: id=%d type=%v name=%s",
						resp.Conversation.Id, resp.Conversation.Type, resp.Conversation.Name)
					conversationID = resp.Conversation.Id

					// Now send message
					go func() {
						time.Sleep(500 * time.Millisecond)
						sendMsg := &ws.WsMessage{
							RequestId: time.Now().UnixNano(),
							Type:      ws.WsMessageType_WS_TYPE_CHAT_SEND_MESSAGE,
							Timestamp: time.Now().UnixMilli(),
							Payload: &ws.WsMessage_Chat{
								Chat: &pb.ChatPayload{
									Payload: &pb.ChatPayload_SendMessage{
										SendMessage: &pb.SendMessageRequest{
											ConversationId: conversationID,
											Type:           pb.MessageType_MESSAGE_TYPE_TEXT,
											Content:        "Hello from test client!",
										},
									},
								},
							},
						}
						log.Printf("📤 Sending Message to conv %d...", conversationID)
						send(c, sendMsg)
					}()
				}

				if resp := chat.GetSendMessageResponse(); resp != nil {
					log.Printf("✅ SendMessageResponse: id=%d localId=%d content=%s",
						resp.Message.Id, resp.Message.LocalId, resp.Message.Content)

					// Now list conversations
					go func() {
						time.Sleep(500 * time.Millisecond)
						listMsg := &ws.WsMessage{
							RequestId: time.Now().UnixNano(),
							Type:      ws.WsMessageType_WS_TYPE_CHAT_GET_CONVERSATION_LIST,
							Timestamp: time.Now().UnixMilli(),
							Payload: &ws.WsMessage_Chat{
								Chat: &pb.ChatPayload{
									Payload: &pb.ChatPayload_GetConversationList{
										GetConversationList: &pb.GetConversationListRequest{
											PageSize: 10,
										},
									},
								},
							},
						}
						log.Printf("📤 Requesting Conversation List...")
						send(c, listMsg)
					}()
				}

				if list := chat.GetGetConversationListResponse(); list != nil {
					log.Printf("✅ ConversationList: count=%d", len(list.Conversations))
					for _, conv := range list.Conversations {
						log.Printf("   - Conv %d: unread=%d lastMsg=%s", conv.Id, conv.UnreadCount, conv.LastMessage.GetContent())
						if conv.Id == conversationID {
							// Mark as read (although unread should be 0 for sender)
							// Just to test the API
							go func() {
								time.Sleep(500 * time.Millisecond)
								markMsg := &ws.WsMessage{
									RequestId: time.Now().UnixNano(),
									Type:      ws.WsMessageType_WS_TYPE_CHAT_MARK_AS_READ,
									Timestamp: time.Now().UnixMilli(),
									Payload: &ws.WsMessage_Chat{
										Chat: &pb.ChatPayload{
											Payload: &pb.ChatPayload_MarkAsRead{
												MarkAsRead: &pb.MarkAsReadRequest{
													ConversationId:    conversationID,
													LastReadMessageId: conv.LastMessage.Id,
												},
											},
										},
									},
								}
								log.Printf("📤 Marking as Read (conv %d, msg %d)...", conversationID, conv.LastMessage.Id)
								send(c, markMsg)
							}()
						}
					}
				}

				if mark := chat.GetMarkAsReadResponse(); mark != nil {
					log.Printf("✅ MarkAsReadResponse: conv=%d unread=%d", mark.ConversationId, mark.UnreadCount)
					log.Println("🎉 All tests completed!")
					// os.Exit(0) // Optional: exit after success
				}

				if push := chat.GetMessagePush(); push != nil {
					log.Printf("🔔 MessagePush: sender=%d content=%s", push.Message.SenderId, push.Message.Content)
				}
			} else if wsMsg.Type == ws.WsMessageType_WS_TYPE_ERROR {
				if errPayload := wsMsg.GetError(); errPayload != nil {
					log.Printf("❌ Error: %s (code=%d)", errPayload.Message, errPayload.Code)
				} else {
					log.Printf("❌ Error: (unknown payload)")
				}
			}
		}
	}()

	// 1. Create Conversation
	time.Sleep(time.Second) // Wait for connection
	reqId := time.Now().UnixNano()
	createMsg := &ws.WsMessage{
		RequestId: reqId,
		Type:      ws.WsMessageType_WS_TYPE_CHAT_CREATE_CONVERSATION,
		Timestamp: time.Now().UnixMilli(),
		Payload: &ws.WsMessage_Chat{
			Chat: &pb.ChatPayload{
				Payload: &pb.ChatPayload_CreateConversation{
					CreateConversation: &pb.CreateConversationRequest{
						Type:           pb.ConversationType_CONVERSATION_TYPE_PRIVATE,
						ParticipantIds: []uint64{*targetUid},
					},
				},
			},
		},
	}
	log.Printf("📤 Creating Conversation with %d...", *targetUid)
	send(c, createMsg)

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func send(c *websocket.Conn, msg *ws.WsMessage) {
	data, _ := proto.Marshal(msg)
	err := c.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		log.Println("write:", err)
	}
}
