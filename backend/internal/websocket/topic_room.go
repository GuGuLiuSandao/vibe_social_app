package websocket

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"social_app/internal/db"
	"social_app/internal/logger"
	"social_app/internal/models"
	proto "social_app/internal/proto"
	chatpb "social_app/internal/proto/chat"

	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const topicRoomHistoryLimit = 120

type topicRoomDefinition struct {
	ID          string
	Name        string
	Description string
	Icon        string
}

type topicRoomState struct {
	definition topicRoomDefinition
	members    map[uint]struct{}
	messages   []*chatpb.TopicRoomMessage
}

var defaultTopicRooms = []topicRoomDefinition{
	{
		ID:          "official-news",
		Name:        "Official News",
		Description: "Official announcements and platform updates.",
		Icon:        "https://img.icons8.com/fluency/96/megaphone.png",
	},
	{
		ID:          "product-feedback",
		Name:        "Product Feedback",
		Description: "Discuss feature ideas and product improvements.",
		Icon:        "https://img.icons8.com/fluency/96/idea.png",
	},
	{
		ID:          "tech-share",
		Name:        "Tech Share",
		Description: "Engineering topics, architecture and coding discussion.",
		Icon:        "https://img.icons8.com/fluency/96/source-code.png",
	},
	{
		ID:          "career-growth",
		Name:        "Career Growth",
		Description: "Career, learning paths, DevOps and interview talk.",
		Icon:        "https://img.icons8.com/fluency/96/rocket--v2.png",
	},
}

func (s *Server) initTopicRooms() {
	for _, def := range defaultTopicRooms {
		s.topicRooms[def.ID] = &topicRoomState{
			definition: def,
			members:    make(map[uint]struct{}),
			messages:   make([]*chatpb.TopicRoomMessage, 0, topicRoomHistoryLimit),
		}
	}
}

func (s *Server) GetTopicRoomList(userID uint) ([]*chatpb.TopicRoom, string) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	rooms := make([]*chatpb.TopicRoom, 0, len(defaultTopicRooms))
	for _, def := range defaultTopicRooms {
		room, ok := s.topicRooms[def.ID]
		if !ok {
			continue
		}
		rooms = append(rooms, topicRoomToPB(room))
	}

	return rooms, s.topicUserRoom[userID]
}

func (s *Server) JoinTopicRoom(userID uint, roomID string) (*chatpb.TopicRoom, []*chatpb.TopicRoomMessage, []*chatpb.TopicRoomMember, error) {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return nil, nil, nil, errors.New("room_id is required")
	}

	var (
		previousRoomID string
		roomSnapshot   *chatpb.TopicRoom
		recentMessages []*chatpb.TopicRoomMessage
		memberIDs      []uint
	)

	s.Mutex.Lock()
	room, ok := s.topicRooms[roomID]
	if !ok {
		s.Mutex.Unlock()
		return nil, nil, nil, errors.New("topic room not found")
	}

	previousRoomID = s.topicUserRoom[userID]
	if previousRoomID != "" && previousRoomID != roomID {
		if oldRoom, exists := s.topicRooms[previousRoomID]; exists {
			delete(oldRoom.members, userID)
		}
	}

	room.members[userID] = struct{}{}
	s.topicUserRoom[userID] = roomID

	roomSnapshot = topicRoomToPB(room)
	recentMessages = cloneTopicMessages(room.messages)
	memberIDs = snapshotMemberIDs(room.members)
	s.Mutex.Unlock()

	members, err := s.buildTopicRoomMembers(memberIDs)
	if err != nil {
		return nil, nil, nil, err
	}

	if previousRoomID != "" && previousRoomID != roomID {
		s.broadcastTopicRoomMembersPush(previousRoomID)
	}
	s.broadcastTopicRoomMembersPush(roomID)

	return roomSnapshot, recentMessages, members, nil
}

func (s *Server) LeaveTopicRoom(userID uint, roomID string) (string, error) {
	requestedRoomID := strings.TrimSpace(roomID)

	s.Mutex.Lock()
	currentRoomID := s.topicUserRoom[userID]
	if currentRoomID == "" {
		s.Mutex.Unlock()
		return "", nil
	}
	if requestedRoomID != "" && requestedRoomID != currentRoomID {
		s.Mutex.Unlock()
		return "", errors.New("you are not in this topic room")
	}

	if room, ok := s.topicRooms[currentRoomID]; ok {
		delete(room.members, userID)
	}
	delete(s.topicUserRoom, userID)
	s.Mutex.Unlock()

	s.broadcastTopicRoomMembersPush(currentRoomID)
	return currentRoomID, nil
}

func (s *Server) SendTopicRoomMessage(userID uint, roomID, content string) (*chatpb.TopicRoomMessage, []uint, error) {
	normalizedRoomID := strings.TrimSpace(roomID)
	if normalizedRoomID == "" {
		return nil, nil, errors.New("room_id is required")
	}

	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		return nil, nil, errors.New("message content is required")
	}

	var senderInfo *chatpb.SenderInfo
	sender, err := s.userService.GetUser(uint64(userID))
	if err == nil {
		senderInfo = &chatpb.SenderInfo{
			Id:       sender.UID,
			Username: sender.Username,
			Nickname: sender.Nickname,
			Avatar:   sender.Avatar,
		}
	} else {
		senderInfo = &chatpb.SenderInfo{Id: uint64(userID)}
	}

	var (
		message      *chatpb.TopicRoomMessage
		recipientIDs []uint
	)

	now := time.Now()

	s.Mutex.Lock()
	room, ok := s.topicRooms[normalizedRoomID]
	if !ok {
		s.Mutex.Unlock()
		return nil, nil, errors.New("topic room not found")
	}

	joinedRoomID := s.topicUserRoom[userID]
	if joinedRoomID != normalizedRoomID {
		s.Mutex.Unlock()
		return nil, nil, errors.New("join this topic room before sending messages")
	}

	s.topicMessageSeq++
	message = &chatpb.TopicRoomMessage{
		Id:        fmt.Sprintf("%d-%d", now.UnixMilli(), s.topicMessageSeq),
		RoomId:    normalizedRoomID,
		SenderId:  uint64(userID),
		Content:   trimmedContent,
		CreatedAt: timestamppb.New(now),
		Sender:    senderInfo,
	}

	room.messages = append(room.messages, message)
	if len(room.messages) > topicRoomHistoryLimit {
		room.messages = append([]*chatpb.TopicRoomMessage(nil), room.messages[len(room.messages)-topicRoomHistoryLimit:]...)
	}
	recipientIDs = snapshotMemberIDs(room.members)
	s.Mutex.Unlock()

	return message, recipientIDs, nil
}

func (s *Server) GetTopicRoomMembers(roomID string) ([]*chatpb.TopicRoomMember, error) {
	normalizedRoomID := strings.TrimSpace(roomID)
	if normalizedRoomID == "" {
		return nil, errors.New("room_id is required")
	}

	s.Mutex.RLock()
	room, ok := s.topicRooms[normalizedRoomID]
	if !ok {
		s.Mutex.RUnlock()
		return nil, errors.New("topic room not found")
	}
	memberIDs := snapshotMemberIDs(room.members)
	s.Mutex.RUnlock()

	return s.buildTopicRoomMembers(memberIDs)
}

func (s *Server) PushTopicRoomMessage(message *chatpb.TopicRoomMessage, recipients []uint) {
	if message == nil || len(recipients) == 0 {
		return
	}

	pushMsg := &proto.WsMessage{
		Type:      proto.WsMessageType_WS_TYPE_CHAT_TOPIC_ROOM_MESSAGE_PUSH,
		Timestamp: time.Now().UnixMilli(),
		Payload: &proto.WsMessage_Chat{
			Chat: &chatpb.ChatPayload{
				Payload: &chatpb.ChatPayload_TopicRoomMessagePush{
					TopicRoomMessagePush: &chatpb.TopicRoomMessagePush{
						Message: message,
					},
				},
			},
		},
	}

	data, err := goproto.Marshal(pushMsg)
	if err != nil {
		logger.Error("Failed to marshal topic room message push: %v", err)
		return
	}

	for _, uid := range recipients {
		s.SendToUser(uid, data)
	}
}

func (s *Server) broadcastTopicRoomMembersPush(roomID string) {
	normalizedRoomID := strings.TrimSpace(roomID)
	if normalizedRoomID == "" {
		return
	}

	var (
		memberIDs   []uint
		recipients  []uint
		onlineCount uint32
	)

	s.Mutex.RLock()
	room, ok := s.topicRooms[normalizedRoomID]
	if !ok {
		s.Mutex.RUnlock()
		return
	}
	memberIDs = snapshotMemberIDs(room.members)
	for uid := range s.Clients {
		recipients = append(recipients, uid)
	}
	onlineCount = uint32(len(room.members))
	s.Mutex.RUnlock()

	members, err := s.buildTopicRoomMembers(memberIDs)
	if err != nil {
		logger.Error("Failed to build topic room members for room=%s: %v", normalizedRoomID, err)
		return
	}

	pushMsg := &proto.WsMessage{
		Type:      proto.WsMessageType_WS_TYPE_CHAT_TOPIC_ROOM_MEMBERS_PUSH,
		Timestamp: time.Now().UnixMilli(),
		Payload: &proto.WsMessage_Chat{
			Chat: &chatpb.ChatPayload{
				Payload: &chatpb.ChatPayload_TopicRoomMembersPush{
					TopicRoomMembersPush: &chatpb.TopicRoomMembersPush{
						RoomId:      normalizedRoomID,
						Members:     members,
						OnlineCount: onlineCount,
					},
				},
			},
		},
	}

	data, err := goproto.Marshal(pushMsg)
	if err != nil {
		logger.Error("Failed to marshal topic room members push: %v", err)
		return
	}

	for _, uid := range recipients {
		s.SendToUser(uid, data)
	}
}

func topicRoomToPB(room *topicRoomState) *chatpb.TopicRoom {
	return &chatpb.TopicRoom{
		Id:          room.definition.ID,
		Name:        room.definition.Name,
		Description: room.definition.Description,
		Icon:        room.definition.Icon,
		OnlineCount: uint32(len(room.members)),
	}
}

func cloneTopicMessages(messages []*chatpb.TopicRoomMessage) []*chatpb.TopicRoomMessage {
	if len(messages) == 0 {
		return []*chatpb.TopicRoomMessage{}
	}

	copied := make([]*chatpb.TopicRoomMessage, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}
		cloned, ok := goproto.Clone(message).(*chatpb.TopicRoomMessage)
		if !ok {
			continue
		}
		copied = append(copied, cloned)
	}
	return copied
}

func snapshotMemberIDs(members map[uint]struct{}) []uint {
	ids := make([]uint, 0, len(members))
	for uid := range members {
		ids = append(ids, uid)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func (s *Server) buildTopicRoomMembers(memberIDs []uint) ([]*chatpb.TopicRoomMember, error) {
	if len(memberIDs) == 0 {
		return []*chatpb.TopicRoomMember{}, nil
	}

	memberUIDs := make([]uint64, 0, len(memberIDs))
	for _, uid := range memberIDs {
		memberUIDs = append(memberUIDs, uint64(uid))
	}

	var users []models.User
	if err := db.GetDB().Where("uid IN ?", memberUIDs).Find(&users).Error; err != nil {
		return nil, err
	}

	userMap := make(map[uint64]models.User, len(users))
	for _, user := range users {
		userMap[user.UID] = user
	}

	members := make([]*chatpb.TopicRoomMember, 0, len(memberIDs))
	for _, uid := range memberIDs {
		user, ok := userMap[uint64(uid)]
		if !ok {
			members = append(members, &chatpb.TopicRoomMember{
				Id: uint64(uid),
			})
			continue
		}

		members = append(members, &chatpb.TopicRoomMember{
			Id:       user.UID,
			Username: user.Username,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
		})
	}

	return members, nil
}
