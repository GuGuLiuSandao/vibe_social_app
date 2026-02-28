package websocket

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"social_app/internal/auth"
	"social_app/internal/config"
	"social_app/internal/db"
	"social_app/internal/logger"
	"social_app/internal/models"
	proto "social_app/internal/proto"
	accountpb "social_app/internal/proto/account"
	chatpb "social_app/internal/proto/chat"
	commonpb "social_app/internal/proto/common"
	relationpb "social_app/internal/proto/relation"

	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- HTTP Handler for WebSocket Upgrade ---

type Handler struct {
	server *Server
	cfg    *config.Config
}

func NewHandler(server *Server, cfg *config.Config) *Handler {
	return &Handler{
		server: server,
		cfg:    cfg,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (h *Handler) HandleWebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		// Try header
		token = c.GetHeader("Sec-WebSocket-Protocol")
	}

	if token == "" {
		logger.Warn("Missing token in WebSocket connection")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	claims, err := auth.ParseToken(token, h.cfg)
	if err != nil {
		logger.Warn("Invalid token in WebSocket connection: %v", err)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		ID:     uint(claims.UserID),
		Server: h.server,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Host:   c.Request.Host,
		Scheme: wsRequestScheme(c.Request),
	}

	client.Server.Register <- client

	go client.WritePump()
	go client.ReadPump()
}

// --- WebSocket Message Handlers ---

func (s *Server) HandleMessage(client *Client, message []byte) {
	var msg proto.WsMessage
	if err := goproto.Unmarshal(message, &msg); err != nil {
		logger.Error("Failed to unmarshal message: %v", err)
		return
	}

	var response *proto.WsMessage

	switch msg.Type {
	case proto.WsMessageType_WS_TYPE_PING:
		response = &proto.WsMessage{
			RequestId: msg.RequestId,
			Type:      proto.WsMessageType_WS_TYPE_PONG,
			Timestamp: time.Now().UnixMilli(),
		}
	case proto.WsMessageType_WS_TYPE_CHAT_SEND_MESSAGE,
		proto.WsMessageType_WS_TYPE_CHAT_GET_CONVERSATION_LIST,
		proto.WsMessageType_WS_TYPE_CHAT_GET_MESSAGE_LIST,
		proto.WsMessageType_WS_TYPE_CHAT_MARK_AS_READ,
		proto.WsMessageType_WS_TYPE_CHAT_CREATE_CONVERSATION:
		if msg.GetChat() != nil {
			response = s.handleChatMessage(client, &msg, msg.GetChat())
		}
	case proto.WsMessageType_WS_TYPE_RELATION_FOLLOW,
		proto.WsMessageType_WS_TYPE_RELATION_UNFOLLOW,
		proto.WsMessageType_WS_TYPE_RELATION_GET_FOLLOWING,
		proto.WsMessageType_WS_TYPE_RELATION_GET_FOLLOWERS,
		proto.WsMessageType_WS_TYPE_RELATION_GET_FRIENDS:
		if msg.GetRelation() != nil {
			response = s.handleRelationMessage(client, &msg, msg.GetRelation())
		}
	case proto.WsMessageType_WS_TYPE_AUTH,
		proto.WsMessageType_WS_TYPE_ACCOUNT_SEARCH_USER,
		proto.WsMessageType_WS_TYPE_ACCOUNT_UPDATE_PROFILE,
		proto.WsMessageType_WS_TYPE_ACCOUNT_UPLOAD_AVATAR:
		if msg.GetAccount() != nil {
			response = s.handleAccountMessage(client, &msg, msg.GetAccount())
		}
	default:
		logger.Warn("Unknown message type: %v", msg.Type)
		response = s.createErrorResponse(msg.RequestId, "unknown message type")
	}

	if response != nil {
		data, err := goproto.Marshal(response)
		if err == nil {
			client.Send <- data
		} else {
			logger.Error("Failed to marshal response: %v", err)
		}
	}
}

func (s *Server) createErrorResponse(requestID int64, message string) *proto.WsMessage {
	return &proto.WsMessage{
		RequestId: requestID,
		Type:      proto.WsMessageType_WS_TYPE_ERROR,
		Timestamp: time.Now().UnixMilli(),
		Payload: &proto.WsMessage_Error{
			Error: &proto.WsError{
				Code:    int32(commonpb.ErrorCode_ERROR_CODE_INTERNAL),
				Message: message,
			},
		},
	}
}

func (s *Server) handleAccountMessage(client *Client, msg *proto.WsMessage, payload *accountpb.AccountPayload) *proto.WsMessage {
	if payload.GetAuth() != nil {
		user, err := s.userService.GetUser(uint64(client.ID))
		if err != nil {
			return s.createAccountAuthResponse(msg.RequestId, commonpb.ErrorCode_ERROR_CODE_NOT_FOUND, "user not found", nil)
		}
		return s.createAccountAuthResponse(msg.RequestId, commonpb.ErrorCode_ERROR_CODE_OK, "ok", user)
	}

	if req := payload.GetSearchUser(); req != nil {
		users, err := s.userService.SearchUser(req.Query)
		if err != nil {
			return s.createErrorResponse(msg.RequestId, err.Error())
		}

		var pbUsers []*accountpb.User
		for _, u := range users {
			pbUsers = append(pbUsers, &accountpb.User{
				Id:       u.UID,
				Username: u.Username,
				Nickname: u.Nickname,
				Avatar:   u.Avatar,
				Bio:      u.Bio,
			})
		}

		return &proto.WsMessage{
			RequestId: msg.RequestId,
			Type:      proto.WsMessageType_WS_TYPE_ACCOUNT_SEARCH_USER_RESPONSE,
			Timestamp: time.Now().UnixMilli(),
			Payload: &proto.WsMessage_Account{
				Account: &accountpb.AccountPayload{
					Payload: &accountpb.AccountPayload_SearchUserResponse{
						SearchUserResponse: &accountpb.SearchUserResponse{
							Users: pbUsers,
						},
					},
				},
			},
		}
	}

	if req := payload.GetUpdateProfile(); req != nil {
		updates := map[string]interface{}{}
		if req.Nickname != "" {
			updates["nickname"] = req.Nickname
		}
		if req.Avatar != "" {
			updates["avatar"] = req.Avatar
		}
		if req.Bio != "" {
			updates["bio"] = req.Bio
		}

		if len(updates) > 0 {
			if err := db.DB.Model(&models.User{}).Where("uid = ?", uint64(client.ID)).Updates(updates).Error; err != nil {
				return s.createUpdateProfileResponse(msg.RequestId, commonpb.ErrorCode_ERROR_CODE_INTERNAL, "failed to update profile", nil)
			}
		}

		user, err := s.userService.GetUser(uint64(client.ID))
		if err != nil {
			return s.createUpdateProfileResponse(msg.RequestId, commonpb.ErrorCode_ERROR_CODE_NOT_FOUND, "user not found", nil)
		}

		return s.createUpdateProfileResponse(msg.RequestId, commonpb.ErrorCode_ERROR_CODE_OK, "ok", user)
	}

	if req := payload.GetUploadAvatar(); req != nil {
		if len(req.Data) == 0 {
			return s.createUploadAvatarResponse(msg.RequestId, commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT, "file is required", "")
		}
		if len(req.Data) > 5*1024*1024 {
			return s.createUploadAvatarResponse(msg.RequestId, commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT, "file size too large (max 5MB)", "")
		}

		ext := strings.ToLower(filepath.Ext(req.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
			return s.createUploadAvatarResponse(msg.RequestId, commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT, "only image files are allowed", "")
		}

		uploadDir := "./uploads/avatars"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			logger.Error("Failed to create upload directory: %v", err)
			return s.createUploadAvatarResponse(msg.RequestId, commonpb.ErrorCode_ERROR_CODE_INTERNAL, "internal server error", "")
		}

		filename := fmt.Sprintf("%d_u%d%s", time.Now().UnixNano(), client.ID, ext)
		dst := filepath.Join(uploadDir, filename)
		if err := os.WriteFile(dst, req.Data, 0o644); err != nil {
			logger.Error("Failed to save uploaded file: %v", err)
			return s.createUploadAvatarResponse(msg.RequestId, commonpb.ErrorCode_ERROR_CODE_INTERNAL, "failed to save file", "")
		}

		url := "/uploads/avatars/" + filename
		if client.Host != "" {
			url = fmt.Sprintf("%s://%s/uploads/avatars/%s", client.Scheme, client.Host, filename)
		}

		return s.createUploadAvatarResponse(msg.RequestId, commonpb.ErrorCode_ERROR_CODE_OK, "upload success", url)
	}

	return s.createErrorResponse(msg.RequestId, "unhandled account payload")
}

func (s *Server) createAccountAuthResponse(requestID int64, code commonpb.ErrorCode, message string, user *models.User) *proto.WsMessage {
	resp := &accountpb.AuthResponse{
		ErrorCode: code,
		Message:   message,
	}
	if user != nil {
		resp.User = toAccountUser(user)
	}

	return &proto.WsMessage{
		RequestId: requestID,
		Type:      proto.WsMessageType_WS_TYPE_AUTH_RESPONSE,
		Timestamp: time.Now().UnixMilli(),
		Payload: &proto.WsMessage_Account{
			Account: &accountpb.AccountPayload{
				Payload: &accountpb.AccountPayload_AuthResponse{
					AuthResponse: resp,
				},
			},
		},
	}
}

func (s *Server) createUpdateProfileResponse(requestID int64, code commonpb.ErrorCode, message string, user *models.User) *proto.WsMessage {
	resp := &accountpb.UpdateProfileResponse{
		ErrorCode: code,
		Message:   message,
	}
	if user != nil {
		resp.User = toAccountUser(user)
	}

	return &proto.WsMessage{
		RequestId: requestID,
		Type:      proto.WsMessageType_WS_TYPE_ACCOUNT_UPDATE_PROFILE_RESPONSE,
		Timestamp: time.Now().UnixMilli(),
		Payload: &proto.WsMessage_Account{
			Account: &accountpb.AccountPayload{
				Payload: &accountpb.AccountPayload_UpdateProfileResponse{
					UpdateProfileResponse: resp,
				},
			},
		},
	}
}

func (s *Server) createUploadAvatarResponse(requestID int64, code commonpb.ErrorCode, message, url string) *proto.WsMessage {
	return &proto.WsMessage{
		RequestId: requestID,
		Type:      proto.WsMessageType_WS_TYPE_ACCOUNT_UPLOAD_AVATAR_RESPONSE,
		Timestamp: time.Now().UnixMilli(),
		Payload: &proto.WsMessage_Account{
			Account: &accountpb.AccountPayload{
				Payload: &accountpb.AccountPayload_UploadAvatarResponse{
					UploadAvatarResponse: &accountpb.UploadAvatarResponse{
						ErrorCode: code,
						Message:   message,
						Url:       url,
					},
				},
			},
		},
	}
}

func toAccountUser(user *models.User) *accountpb.User {
	if user == nil {
		return nil
	}

	return &accountpb.User{
		Id:       user.UID,
		Username: user.Username,
		Email:    user.Email,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
		Bio:      user.Bio,
	}
}

func wsRequestScheme(req *http.Request) string {
	if req == nil {
		return "http"
	}
	if req.TLS != nil {
		return "https"
	}
	if forwarded := req.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		return strings.ToLower(forwarded)
	}
	return "http"
}

func (s *Server) handleChatMessage(client *Client, msg *proto.WsMessage, payload *chatpb.ChatPayload) *proto.WsMessage {
	if req := payload.GetSendMessage(); req != nil {
		message, err := s.chatService.SendMessage(uint64(client.ID), req)
		if err != nil {
			return s.createErrorResponse(msg.RequestId, err.Error())
		}

		// Push to other participants (except sender)
		participantIDs, _ := s.chatService.GetConversationParticipantIDs(message.ConversationID)
		pushMsg := &proto.WsMessage{
			Type:      proto.WsMessageType_WS_TYPE_CHAT_MESSAGE_PUSH,
			Timestamp: time.Now().UnixMilli(),
			Payload: &proto.WsMessage_Chat{
				Chat: &chatpb.ChatPayload{
					Payload: &chatpb.ChatPayload_MessagePush{
						MessagePush: &chatpb.MessagePush{
							Message: &chatpb.Message{
								Id:             message.ID,
								LocalId:        message.LocalID,
								ConversationId: message.ConversationID,
								SenderId:       message.SenderID,
								Content:        message.Content,
								Type:           chatpb.MessageType(message.Type),
								CreatedAt:      timestamppb.New(message.CreatedAt),
								Sender: &chatpb.SenderInfo{ // Simple sender info
									Id: uint64(client.ID),
									// Ideally populate other fields if available in context or fetch them
								},
							},
						},
					},
				},
			},
		}

		// Fill sender info fully
		sender, err := s.chatService.GetUser(message.SenderID)
		if err == nil {
			pushMsg.GetChat().GetMessagePush().Message.Sender = &chatpb.SenderInfo{
				Id:       sender.UID,
				Username: sender.Username,
				Nickname: sender.Nickname,
				Avatar:   sender.Avatar,
			}
		}

		data, _ := goproto.Marshal(pushMsg)
		for _, uid := range participantIDs {
			if uid != uint64(client.ID) {
				s.SendToUser(uint(uid), data)
			}
		}

		return &proto.WsMessage{
			RequestId: msg.RequestId,
			Type:      proto.WsMessageType_WS_TYPE_CHAT_SEND_MESSAGE_RESPONSE,
			Timestamp: time.Now().UnixMilli(),
			Payload: &proto.WsMessage_Chat{
				Chat: &chatpb.ChatPayload{
					Payload: &chatpb.ChatPayload_SendMessageResponse{
						SendMessageResponse: &chatpb.SendMessageResponse{
							Message: pushMsg.GetChat().GetMessagePush().Message,
						},
					},
				},
			},
		}
	}

	if req := payload.GetGetConversationList(); req != nil {
		conversations, nextPageToken, err := s.chatService.GetConversationList(uint64(client.ID), int(req.PageSize), req.PageToken)
		if err != nil {
			return s.createErrorResponse(msg.RequestId, err.Error())
		}
		return &proto.WsMessage{
			RequestId: msg.RequestId,
			Type:      proto.WsMessageType_WS_TYPE_CHAT_GET_CONVERSATION_LIST_RESPONSE,
			Timestamp: time.Now().UnixMilli(),
			Payload: &proto.WsMessage_Chat{
				Chat: &chatpb.ChatPayload{
					Payload: &chatpb.ChatPayload_GetConversationListResponse{
						GetConversationListResponse: &chatpb.GetConversationListResponse{
							Conversations: conversations,
							NextPageToken: nextPageToken,
						},
					},
				},
			},
		}
	}

	if req := payload.GetGetMessageList(); req != nil {
		messages, nextPageToken, err := s.chatService.GetMessageList(uint64(client.ID), req)
		if err != nil {
			return s.createErrorResponse(msg.RequestId, err.Error())
		}
		return &proto.WsMessage{
			RequestId: msg.RequestId,
			Type:      proto.WsMessageType_WS_TYPE_CHAT_GET_MESSAGE_LIST_RESPONSE,
			Timestamp: time.Now().UnixMilli(),
			Payload: &proto.WsMessage_Chat{
				Chat: &chatpb.ChatPayload{
					Payload: &chatpb.ChatPayload_GetMessageListResponse{
						GetMessageListResponse: &chatpb.GetMessageListResponse{
							Messages:      messages,
							NextPageToken: nextPageToken,
						},
					},
				},
			},
		}
	}

	if req := payload.GetCreateConversation(); req != nil {
		conv, err := s.chatService.CreateConversation(uint64(client.ID), req)
		if err != nil {
			return s.createErrorResponse(msg.RequestId, err.Error())
		}

		// Convert to proto
		pbConv := &chatpb.Conversation{
			Id:        conv.ID,
			Type:      chatpb.ConversationType(conv.Type),
			Name:      conv.Name,
			Avatar:    conv.Avatar,
			UpdatedAt: timestamppb.New(conv.UpdatedAt),
		}
		// Note: CreateConversation returns model, we might need to populate extra fields or just return basic info
		// For private chat, the name/avatar might need to be customized for the viewer, but here we return raw or let client handle.
		// Actually, let's try to match GetConversationList logic for consistency if possible, or just return what we have.

		// Push to other participants
		// ... logic to push WS_TYPE_CHAT_CONVERSATION_PUSH ...

		return &proto.WsMessage{
			RequestId: msg.RequestId,
			Type:      proto.WsMessageType_WS_TYPE_CHAT_CREATE_CONVERSATION_RESPONSE,
			Timestamp: time.Now().UnixMilli(),
			Payload: &proto.WsMessage_Chat{
				Chat: &chatpb.ChatPayload{
					Payload: &chatpb.ChatPayload_CreateConversationResponse{
						CreateConversationResponse: &chatpb.CreateConversationResponse{
							Conversation: pbConv,
						},
					},
				},
			},
		}
	}

	if req := payload.GetMarkAsRead(); req != nil {
		unreadCount, err := s.chatService.MarkAsRead(uint64(client.ID), req)
		if err != nil {
			return s.createErrorResponse(msg.RequestId, err.Error())
		}
		return &proto.WsMessage{
			RequestId: msg.RequestId,
			Type:      proto.WsMessageType_WS_TYPE_CHAT_MARK_AS_READ_RESPONSE,
			Timestamp: time.Now().UnixMilli(),
			Payload: &proto.WsMessage_Chat{
				Chat: &chatpb.ChatPayload{
					Payload: &chatpb.ChatPayload_MarkAsReadResponse{
						MarkAsReadResponse: &chatpb.MarkAsReadResponse{
							ConversationId: req.ConversationId,
							UnreadCount:    unreadCount,
						},
					},
				},
			},
		}
	}

	return s.createErrorResponse(msg.RequestId, "unhandled chat payload")
}

func (s *Server) handleRelationMessage(client *Client, msg *proto.WsMessage, payload *relationpb.RelationPayload) *proto.WsMessage {
	if payload.GetFollow() != nil {
		return s.handleFollow(client, msg, payload)
	}
	if payload.GetUnfollow() != nil {
		return s.handleUnfollow(client, msg, payload)
	}
	if payload.GetGetFollowing() != nil {
		return s.handleGetFollowing(client, msg, payload)
	}
	if payload.GetGetFollowers() != nil {
		return s.handleGetFollowers(client, msg, payload)
	}
	if payload.GetGetFriends() != nil {
		return s.handleGetFriends(client, msg, payload)
	}

	logger.Warn("Unhandled relation payload from user %d", client.ID)
	return s.createErrorResponse(msg.RequestId, "unhandled relation payload")
}

func (s *Server) handleFollow(client *Client, msg *proto.WsMessage, payload *relationpb.RelationPayload) *proto.WsMessage {
	req := payload.GetFollow()
	err := s.relationService.Follow(uint64(client.ID), req.TargetUid)
	if err != nil {
		return s.createErrorResponse(msg.RequestId, err.Error())
	}

	// Send push notification to target user
	go s.sendRelationPush(req.TargetUid, uint64(client.ID), relationpb.RelationPush_TYPE_FOLLOWED)

	return &proto.WsMessage{
		RequestId: msg.RequestId,
		Type:      proto.WsMessageType_WS_TYPE_RELATION_FOLLOW_RESPONSE,
		Timestamp: time.Now().UnixMilli(),
		Payload: &proto.WsMessage_Relation{
			Relation: &relationpb.RelationPayload{
				Payload: &relationpb.RelationPayload_FollowResponse{
					FollowResponse: &relationpb.FollowResponse{
						ErrorCode: commonpb.ErrorCode_ERROR_CODE_OK,
						Message:   "followed",
					},
				},
			},
		},
	}
}

func (s *Server) handleUnfollow(client *Client, msg *proto.WsMessage, payload *relationpb.RelationPayload) *proto.WsMessage {
	req := payload.GetUnfollow()
	err := s.relationService.Unfollow(uint64(client.ID), req.TargetUid)
	if err != nil {
		return s.createErrorResponse(msg.RequestId, err.Error())
	}

	// Send push notification to target user
	go s.sendRelationPush(req.TargetUid, uint64(client.ID), relationpb.RelationPush_TYPE_UNFOLLOWED)

	return &proto.WsMessage{
		RequestId: msg.RequestId,
		Type:      proto.WsMessageType_WS_TYPE_RELATION_UNFOLLOW_RESPONSE,
		Timestamp: time.Now().UnixMilli(),
		Payload: &proto.WsMessage_Relation{
			Relation: &relationpb.RelationPayload{
				Payload: &relationpb.RelationPayload_UnfollowResponse{
					UnfollowResponse: &relationpb.UnfollowResponse{
						ErrorCode: commonpb.ErrorCode_ERROR_CODE_OK,
						Message:   "unfollowed",
					},
				},
			},
		},
	}
}

func (s *Server) handleGetFollowing(client *Client, msg *proto.WsMessage, payload *relationpb.RelationPayload) *proto.WsMessage {
	relations, err := s.relationService.GetFollowing(uint64(client.ID))
	if err != nil {
		return s.createErrorResponse(msg.RequestId, err.Error())
	}

	return &proto.WsMessage{
		RequestId: msg.RequestId,
		Type:      proto.WsMessageType_WS_TYPE_RELATION_GET_FOLLOWING_RESPONSE,
		Timestamp: time.Now().UnixMilli(),
		Payload: &proto.WsMessage_Relation{
			Relation: &relationpb.RelationPayload{
				Payload: &relationpb.RelationPayload_GetFollowingResponse{
					GetFollowingResponse: &relationpb.GetFollowingResponse{
						FollowingList: relations,
					},
				},
			},
		},
	}
}

func (s *Server) handleGetFollowers(client *Client, msg *proto.WsMessage, payload *relationpb.RelationPayload) *proto.WsMessage {
	relations, err := s.relationService.GetFollowers(uint64(client.ID))
	if err != nil {
		return s.createErrorResponse(msg.RequestId, err.Error())
	}

	return &proto.WsMessage{
		RequestId: msg.RequestId,
		Type:      proto.WsMessageType_WS_TYPE_RELATION_GET_FOLLOWERS_RESPONSE,
		Timestamp: time.Now().UnixMilli(),
		Payload: &proto.WsMessage_Relation{
			Relation: &relationpb.RelationPayload{
				Payload: &relationpb.RelationPayload_GetFollowersResponse{
					GetFollowersResponse: &relationpb.GetFollowersResponse{
						FollowerList: relations,
					},
				},
			},
		},
	}
}

func (s *Server) handleGetFriends(client *Client, msg *proto.WsMessage, payload *relationpb.RelationPayload) *proto.WsMessage {
	req := payload.GetGetFriends()
	relations, nextCursor, err := s.relationService.GetFriends(uint64(client.ID), req.PageSize, req.Cursor)
	if err != nil {
		return s.createErrorResponse(msg.RequestId, err.Error())
	}

	return &proto.WsMessage{
		RequestId: msg.RequestId,
		Type:      proto.WsMessageType_WS_TYPE_RELATION_GET_FRIENDS_RESPONSE,
		Timestamp: time.Now().UnixMilli(),
		Payload: &proto.WsMessage_Relation{
			Relation: &relationpb.RelationPayload{
				Payload: &relationpb.RelationPayload_GetFriendsResponse{
					GetFriendsResponse: &relationpb.GetFriendsResponse{
						FriendList: relations,
						NextCursor: nextCursor,
					},
				},
			},
		},
	}
}

func (s *Server) sendRelationPush(targetUserID, fromUserID uint64, pushType relationpb.RelationPush_Type) {
	// Get fromUser info
	user, err := s.userService.GetUser(fromUserID)
	if err != nil {
		logger.Error("Failed to get user %d for push: %v", fromUserID, err)
		return
	}

	// Construct Push Message
	pushMsg := &proto.WsMessage{
		Type:      proto.WsMessageType_WS_TYPE_RELATION_PUSH,
		Timestamp: time.Now().UnixMilli(),
		Payload: &proto.WsMessage_Relation{
			Relation: &relationpb.RelationPayload{
				Payload: &relationpb.RelationPayload_RelationPush{
					RelationPush: &relationpb.RelationPush{
						Type: pushType,
						User: &accountpb.User{
							Id:       user.UID,
							Username: user.Username,
							Nickname: user.Nickname,
							Avatar:   user.Avatar,
						},
						Timestamp: time.Now().UnixMilli(),
					},
				},
			},
		},
	}

	// Marshal
	data, err := goproto.Marshal(pushMsg)
	if err != nil {
		logger.Error("Failed to marshal push message: %v", err)
		return
	}

	// Send
	s.SendToUser(uint(targetUserID), data)
}
