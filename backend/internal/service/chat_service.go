package service

import (
	"errors"
	"social_app/internal/db"
	"social_app/internal/models"
	pb "social_app/internal/proto/chat"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChatService struct {
	db              *gorm.DB
	relationService *RelationService
}

type groupActorContext struct {
	Conversation models.Conversation
	Participant  models.ConversationParticipant
}

// NewChatService creates a new ChatService instance
func NewChatService(relationService *RelationService) *ChatService {
	return &ChatService{
		db:              db.GetDB(),
		relationService: relationService,
	}
}

// GetUser 获取用户信息
func (s *ChatService) GetUser(userID uint64) (*models.User, error) {
	var user models.User
	if err := s.db.Where("uid = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func normalizeGroupKind(kind pb.GroupKind) string {
	switch kind {
	case pb.GroupKind_GROUP_KIND_OFFICIAL:
		return models.GroupKindOfficial
	case pb.GroupKind_GROUP_KIND_PLAYER_CREATED, pb.GroupKind_GROUP_KIND_UNSPECIFIED:
		return models.GroupKindPlayerCreated
	default:
		return models.GroupKindPlayerCreated
	}
}

func normalizeJoinMode(mode pb.GroupJoinMode) string {
	switch mode {
	case pb.GroupJoinMode_GROUP_JOIN_MODE_APPROVAL:
		return models.GroupJoinModeApproval
	case pb.GroupJoinMode_GROUP_JOIN_MODE_PUBLIC:
		return models.GroupJoinModePublic
	case pb.GroupJoinMode_GROUP_JOIN_MODE_PRIVATE, pb.GroupJoinMode_GROUP_JOIN_MODE_UNSPECIFIED:
		return models.GroupJoinModePrivate
	default:
		return models.GroupJoinModePrivate
	}
}

func toPBGroupKind(kind string) pb.GroupKind {
	switch kind {
	case models.GroupKindOfficial:
		return pb.GroupKind_GROUP_KIND_OFFICIAL
	case models.GroupKindPlayerCreated:
		return pb.GroupKind_GROUP_KIND_PLAYER_CREATED
	default:
		return pb.GroupKind_GROUP_KIND_UNSPECIFIED
	}
}

func toPBJoinMode(mode string) pb.GroupJoinMode {
	switch mode {
	case models.GroupJoinModePrivate:
		return pb.GroupJoinMode_GROUP_JOIN_MODE_PRIVATE
	case models.GroupJoinModeApproval:
		return pb.GroupJoinMode_GROUP_JOIN_MODE_APPROVAL
	case models.GroupJoinModePublic:
		return pb.GroupJoinMode_GROUP_JOIN_MODE_PUBLIC
	default:
		return pb.GroupJoinMode_GROUP_JOIN_MODE_UNSPECIFIED
	}
}

func toPBGroupStatus(status string) pb.GroupStatus {
	switch status {
	case models.GroupStatusActive:
		return pb.GroupStatus_GROUP_STATUS_ACTIVE
	case models.GroupStatusDissolved:
		return pb.GroupStatus_GROUP_STATUS_DISSOLVED
	default:
		return pb.GroupStatus_GROUP_STATUS_UNSPECIFIED
	}
}

func toPBGroupRole(role string) pb.GroupMemberRole {
	switch role {
	case models.GroupRoleOwner:
		return pb.GroupMemberRole_GROUP_MEMBER_ROLE_OWNER
	case models.GroupRoleAdmin:
		return pb.GroupMemberRole_GROUP_MEMBER_ROLE_ADMIN
	case models.GroupRoleMember:
		return pb.GroupMemberRole_GROUP_MEMBER_ROLE_MEMBER
	default:
		return pb.GroupMemberRole_GROUP_MEMBER_ROLE_UNSPECIFIED
	}
}

func normalizeTargetRole(role pb.GroupMemberRole) (string, error) {
	switch role {
	case pb.GroupMemberRole_GROUP_MEMBER_ROLE_ADMIN:
		return models.GroupRoleAdmin, nil
	case pb.GroupMemberRole_GROUP_MEMBER_ROLE_MEMBER:
		return models.GroupRoleMember, nil
	default:
		return "", errors.New("unsupported target role")
	}
}

func toPBJoinRequestStatus(status string) pb.GroupJoinRequestStatus {
	switch status {
	case models.GroupJoinRequestStatusPending:
		return pb.GroupJoinRequestStatus_GROUP_JOIN_REQUEST_STATUS_PENDING
	case models.GroupJoinRequestStatusApproved:
		return pb.GroupJoinRequestStatus_GROUP_JOIN_REQUEST_STATUS_APPROVED
	case models.GroupJoinRequestStatusRejected:
		return pb.GroupJoinRequestStatus_GROUP_JOIN_REQUEST_STATUS_REJECTED
	case models.GroupJoinRequestStatusCancelled:
		return pb.GroupJoinRequestStatus_GROUP_JOIN_REQUEST_STATUS_CANCELLED
	default:
		return pb.GroupJoinRequestStatus_GROUP_JOIN_REQUEST_STATUS_UNSPECIFIED
	}
}

func toPBInvitationStatus(status string) pb.GroupInvitationStatus {
	switch status {
	case models.GroupInvitationStatusPending:
		return pb.GroupInvitationStatus_GROUP_INVITATION_STATUS_PENDING
	case models.GroupInvitationStatusAccepted:
		return pb.GroupInvitationStatus_GROUP_INVITATION_STATUS_ACCEPTED
	case models.GroupInvitationStatusRejected:
		return pb.GroupInvitationStatus_GROUP_INVITATION_STATUS_REJECTED
	case models.GroupInvitationStatusCancelled:
		return pb.GroupInvitationStatus_GROUP_INVITATION_STATUS_CANCELLED
	case models.GroupInvitationStatusExpired:
		return pb.GroupInvitationStatus_GROUP_INVITATION_STATUS_EXPIRED
	default:
		return pb.GroupInvitationStatus_GROUP_INVITATION_STATUS_UNSPECIFIED
	}
}

func hasAnyRole(role string, roles ...string) bool {
	for _, candidate := range roles {
		if role == candidate {
			return true
		}
	}
	return false
}

func isGroupRoleManageable(actorRole, targetRole string) bool {
	if actorRole == models.GroupRoleOwner {
		return targetRole != models.GroupRoleOwner
	}
	if actorRole == models.GroupRoleAdmin {
		return targetRole == models.GroupRoleMember
	}
	return false
}

func (s *ChatService) getGroupActorContext(tx *gorm.DB, conversationID, userID uint64, lock bool) (*groupActorContext, error) {
	query := tx
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var conversation models.Conversation
	if err := query.First(&conversation, conversationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("group not found")
		}
		return nil, err
	}
	if conversation.Type != models.ConversationTypeGroup {
		return nil, errors.New("conversation is not a group")
	}

	partQuery := tx
	if lock {
		partQuery = partQuery.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var participant models.ConversationParticipant
	if err := partQuery.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&participant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("you are not a member of this group")
		}
		return nil, err
	}

	return &groupActorContext{Conversation: conversation, Participant: participant}, nil
}

func (s *ChatService) requireActiveGroup(conversation models.Conversation) error {
	if conversation.Status == models.GroupStatusDissolved {
		return errors.New("group has been dissolved")
	}
	return nil
}

func (s *ChatService) countGroupMembersTx(tx *gorm.DB, conversationID uint64) (uint32, error) {
	var count int64
	if err := tx.Model(&models.ConversationParticipant{}).Where("conversation_id = ?", conversationID).Count(&count).Error; err != nil {
		return 0, err
	}
	return uint32(count), nil
}

func (s *ChatService) buildPBConversation(tx *gorm.DB, conv models.Conversation, participant *models.ConversationParticipant) (*pb.Conversation, error) {
	memberCount, err := s.countGroupMembersTx(tx, conv.ID)
	if err != nil {
		return nil, err
	}
	pbConv := &pb.Conversation{
		Id:           conv.ID,
		Type:         pb.ConversationType(conv.Type),
		Name:         conv.Name,
		Avatar:       conv.Avatar,
		UpdatedAt:    timestamppb.New(conv.UpdatedAt),
		Description:  conv.Description,
		Announcement: conv.Announcement,
		GroupKind:    toPBGroupKind(conv.GroupKind),
		JoinMode:     toPBJoinMode(conv.JoinMode),
		Status:       toPBGroupStatus(conv.Status),
		OwnerId:      conv.OwnerID,
		MemberCount:  memberCount,
	}
	if participant != nil {
		if conv.LastLocalID > participant.LastReadLocalID {
			pbConv.UnreadCount = uint32(conv.LastLocalID - participant.LastReadLocalID)
		}
		pbConv.MyRole = toPBGroupRole(participant.Role)
	}
	if conv.LastMessageID != nil {
		var lastMessage models.Message
		if err := tx.First(&lastMessage, *conv.LastMessageID).Error; err == nil {
			pbConv.LastMessage = &pb.Message{
				Id:             lastMessage.ID,
				LocalId:        lastMessage.LocalID,
				ConversationId: lastMessage.ConversationID,
				SenderId:       lastMessage.SenderID,
				Content:        lastMessage.Content,
				Type:           pb.MessageType(lastMessage.Type),
				CreatedAt:      timestamppb.New(lastMessage.CreatedAt),
			}
		}
	}
	return pbConv, nil
}

func (s *ChatService) buildPBGroupMember(participant models.ConversationParticipant) *pb.GroupMember {
	return &pb.GroupMember{
		UserId:   participant.UserID,
		Username: participant.User.Username,
		Nickname: participant.User.Nickname,
		Avatar:   participant.User.Avatar,
		Role:     toPBGroupRole(participant.Role),
		JoinedAt: timestamppb.New(participant.JoinedAt),
	}
}

func (s *ChatService) addParticipantTx(tx *gorm.DB, conversationID, userID uint64, role string) error {
	participant := models.ConversationParticipant{
		ConversationID: conversationID,
		UserID:         userID,
		Role:           role,
		JoinedAt:       time.Now(),
	}
	return tx.Where("conversation_id = ? AND user_id = ?", conversationID, userID).FirstOrCreate(&participant).Error
}

func (s *ChatService) getGroupMembersTx(tx *gorm.DB, conversationID uint64) ([]*pb.GroupMember, error) {
	var participants []models.ConversationParticipant
	if err := tx.Where("conversation_id = ?", conversationID).Preload("User").Order("joined_at ASC").Find(&participants).Error; err != nil {
		return nil, err
	}
	members := make([]*pb.GroupMember, 0, len(participants))
	for _, participant := range participants {
		members = append(members, s.buildPBGroupMember(participant))
	}
	return members, nil
}

func (s *ChatService) getGroupJoinRequestTx(tx *gorm.DB, requestID uint64, lock bool) (*models.GroupJoinRequest, error) {
	query := tx
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var joinRequest models.GroupJoinRequest
	if err := query.First(&joinRequest, requestID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("join request not found")
		}
		return nil, err
	}
	return &joinRequest, nil
}

func (s *ChatService) buildPBJoinRequest(tx *gorm.DB, request models.GroupJoinRequest) (*pb.GroupJoinRequest, error) {
	var user models.User
	if err := tx.Where("uid = ?", request.ApplicantID).First(&user).Error; err != nil {
		return nil, err
	}
	resp := &pb.GroupJoinRequest{
		Id:                request.ID,
		ConversationId:    request.ConversationID,
		ApplicantId:       request.ApplicantID,
		ApplicantUsername: user.Username,
		ApplicantNickname: user.Nickname,
		ApplicantAvatar:   user.Avatar,
		Message:           request.Message,
		Status:            toPBJoinRequestStatus(request.Status),
		CreatedAt:         timestamppb.New(request.CreatedAt),
	}
	if request.ReviewedBy != nil {
		resp.ReviewedBy = *request.ReviewedBy
	}
	if request.ReviewedAt != nil {
		resp.ReviewedAt = timestamppb.New(*request.ReviewedAt)
	}
	return resp, nil
}

func (s *ChatService) buildPBInvitation(tx *gorm.DB, invitation models.GroupInvitation) (*pb.GroupInvitation, error) {
	var inviter models.User
	if err := tx.Where("uid = ?", invitation.InviterID).First(&inviter).Error; err != nil {
		return nil, err
	}
	var conversation models.Conversation
	if err := tx.First(&conversation, invitation.ConversationID).Error; err != nil {
		return nil, err
	}
	return &pb.GroupInvitation{
		Id:              invitation.ID,
		ConversationId:  invitation.ConversationID,
		InviterId:       invitation.InviterID,
		InviterUsername: inviter.Username,
		InviterNickname: inviter.Nickname,
		InviterAvatar:   inviter.Avatar,
		Status:          toPBInvitationStatus(invitation.Status),
		CreatedAt:       timestamppb.New(invitation.CreatedAt),
		GroupName:       conversation.Name,
		GroupAvatar:     conversation.Avatar,
	}, nil
}

// SendMessage 发送消息
// 使用数据库事务和悲观锁保证 LocalID 的单调递增
func (s *ChatService) SendMessage(senderID uint64, req *pb.SendMessageRequest) (*models.Message, error) {
	var message *models.Message

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 检查会话是否存在，并加锁 (悲观锁)
		var conversation models.Conversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&conversation, req.ConversationId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("conversation not found")
			}
			return err
		}

		// 2. 检查发送者是否在会话中
		var participant models.ConversationParticipant
		if err := tx.Where("conversation_id = ? AND user_id = ?", req.ConversationId, senderID).First(&participant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("you are not a member of this conversation")
			}
			return err
		}

		// 2.1 检查私聊权限 (关注关系和消息限制)
		if conversation.Type == models.ConversationTypePrivate {
			// 获取对方ID
			var otherUserID uint64
			if err := tx.Model(&models.ConversationParticipant{}).
				Select("user_id").
				Where("conversation_id = ? AND user_id != ?", req.ConversationId, senderID).
				Limit(1).
				Scan(&otherUserID).Error; err != nil {
				return err
			}

			// 检查我是否关注了对方
			hasBlockingRelation, err := s.relationService.HasBlockingRelation(senderID, otherUserID)
			if err != nil {
				return err
			}
			if hasBlockingRelation {
				return errors.New("private chat is blocked by blacklist settings")
			}

			isFollowing, err := s.relationService.IsFollowing(senderID, otherUserID)
			if err != nil {
				return err
			}
			if !isFollowing {
				return errors.New("you must follow the user to send messages")
			}

			// 检查对方是否关注了我 (回关)
			isFollowedBy, err := s.relationService.IsFollowing(otherUserID, senderID)
			if err != nil {
				return err
			}

			// 如果对方没有回关，限制只能发送3条消息
			if !isFollowedBy {
				var sentCount int64
				if err := tx.Model(&models.Message{}).
					Where("conversation_id = ? AND sender_id = ?", req.ConversationId, senderID).
					Count(&sentCount).Error; err != nil {
					return err
				}
				if sentCount >= 3 {
					return errors.New("message limit reached (3). wait for follow back.")
				}
			}
		}

		if conversation.Type == models.ConversationTypeGroup && conversation.Status == models.GroupStatusDissolved {
			return errors.New("group has been dissolved")
		}

		// 3. 生成新的 LocalID
		newLocalID := conversation.LastLocalID + 1

		// 4. 创建消息
		message = &models.Message{
			ConversationID: req.ConversationId,
			LocalID:        newLocalID,
			SenderID:       senderID,
			Content:        req.Content,
			Type:           int(req.Type),
			CreatedAt:      time.Now(),
		}

		if err := tx.Create(message).Error; err != nil {
			return err
		}

		// 5. 更新会话状态 (LastLocalID, LastMessageID, UpdatedAt)
		updates := map[string]interface{}{
			"last_local_id":   newLocalID,
			"last_message_id": message.ID,
			"updated_at":      message.CreatedAt,
		}
		if err := tx.Model(&conversation).Updates(updates).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return message, nil
}

// GetConversationList 获取会话列表 (支持分页)
func (s *ChatService) GetConversationList(userID uint64, pageSize int, pageToken string) ([]*pb.Conversation, string, error) {
	// 1. 查询用户参与的所有会话ID
	var participants []models.ConversationParticipant

	// 分页查询逻辑:
	// 使用 updated_at 作为游标 (cursor)，因为我们要按更新时间倒序排列
	// pageToken 假设传入的是上一页最后一条数据的 UpdatedAt (Unix milli string)

	query := s.db.Model(&models.ConversationParticipant{}).
		Joins("JOIN conversations ON conversations.id = conversation_participants.conversation_id").
		Where("conversation_participants.user_id = ?", userID)

	if pageToken != "" {
		cursorTimeMs, err := strconv.ParseInt(pageToken, 10, 64)
		if err == nil {
			cursorTime := time.UnixMilli(cursorTimeMs)
			query = query.Where("conversations.updated_at < ?", cursorTime)
		}
	}

	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	err := query.
		Order("conversations.updated_at DESC").
		Limit(pageSize).
		Preload("Conversation").
		Preload("Conversation.LastMessage"). // 预加载最后一条消息
		Find(&participants).Error

	if err != nil {
		return nil, "", err
	}

	// 获取私聊对象的详细信息
	var privateConvIDs []uint64
	for _, p := range participants {
		if p.Conversation.Type == models.ConversationTypePrivate {
			privateConvIDs = append(privateConvIDs, p.ConversationID)
		}
	}

	otherParticipantsMap := make(map[uint64]models.User)
	if len(privateConvIDs) > 0 {
		var otherParts []models.ConversationParticipant
		// 查询这些会话中，非当前用户的成员信息
		if err := s.db.Where("conversation_id IN ? AND user_id != ?", privateConvIDs, userID).
			Preload("User").
			Find(&otherParts).Error; err == nil {
			for _, op := range otherParts {
				otherParticipantsMap[op.ConversationID] = op.User
			}
		}
	}

	var pbConversations []*pb.Conversation
	var lastUpdatedAt int64

	for _, p := range participants {
		conv := p.Conversation
		unreadCount := 0
		if conv.LastLocalID > p.LastReadLocalID {
			unreadCount = int(conv.LastLocalID - p.LastReadLocalID)
		}

		pbConv := &pb.Conversation{
			Id:           conv.ID,
			Type:         pb.ConversationType(conv.Type),
			Name:         conv.Name,
			Avatar:       conv.Avatar,
			UnreadCount:  uint32(unreadCount),
			UpdatedAt:    timestamppb.New(conv.UpdatedAt),
			Description:  conv.Description,
			Announcement: conv.Announcement,
			GroupKind:    toPBGroupKind(conv.GroupKind),
			JoinMode:     toPBJoinMode(conv.JoinMode),
			Status:       toPBGroupStatus(conv.Status),
			OwnerId:      conv.OwnerID,
			MyRole:       toPBGroupRole(p.Role),
		}
		if conv.Type == models.ConversationTypeGroup {
			var memberCount int64
			s.db.Model(&models.ConversationParticipant{}).Where("conversation_id = ?", conv.ID).Count(&memberCount)
			pbConv.MemberCount = uint32(memberCount)
		}

		// 如果是私聊，使用对方的信息覆盖
		if conv.Type == models.ConversationTypePrivate {
			if otherUser, ok := otherParticipantsMap[conv.ID]; ok {
				pbConv.Name = otherUser.Nickname
				if pbConv.Name == "" {
					pbConv.Name = otherUser.Username
				}
				pbConv.Avatar = otherUser.Avatar
			}
		}

		if conv.LastMessage != nil {
			pbConv.LastMessage = &pb.Message{
				Id:             conv.LastMessage.ID,
				LocalId:        conv.LastMessage.LocalID,
				ConversationId: conv.LastMessage.ConversationID,
				SenderId:       conv.LastMessage.SenderID,
				Content:        conv.LastMessage.Content,
				Type:           pb.MessageType(conv.LastMessage.Type),
				CreatedAt:      timestamppb.New(conv.LastMessage.CreatedAt),
			}
		}

		pbConversations = append(pbConversations, pbConv)
		lastUpdatedAt = conv.UpdatedAt.UnixMilli()
	}

	nextPageToken := ""
	if len(participants) == pageSize {
		nextPageToken = strconv.FormatInt(lastUpdatedAt, 10)
	}

	return pbConversations, nextPageToken, nil
}

// GetMessageList 获取消息列表 (支持分页)
func (s *ChatService) GetMessageList(userID uint64, req *pb.GetMessageListRequest) ([]*pb.Message, string, error) {
	// 1. 检查用户是否在会话中 (为了性能，如果是公开群聊可能不需要这一步，但私密社交必须)
	var count int64
	s.db.Model(&models.ConversationParticipant{}).
		Where("conversation_id = ? AND user_id = ?", req.ConversationId, userID).
		Count(&count)

	if count == 0 {
		return nil, "", errors.New("you are not a member of this conversation")
	}

	var conversationMeta models.Conversation
	if err := s.db.Select("id", "type", "status").First(&conversationMeta, req.ConversationId).Error; err == nil {
		if conversationMeta.Type == models.ConversationTypeGroup && conversationMeta.Status == models.GroupStatusDissolved {
			return nil, "", errors.New("group has been dissolved")
		}
	}

	// 2. 查询消息
	var messages []models.Message
	query := s.db.Where("conversation_id = ?", req.ConversationId)

	// 分页逻辑: 使用 local_id 作为游标 (cursor)
	// 如果传入 page_token，说明是向上翻页（获取更旧的消息），local_id < page_token
	if req.PageToken != "" {
		cursorLocalID, err := strconv.ParseUint(req.PageToken, 10, 64)
		if err == nil {
			query = query.Where("local_id < ?", cursorLocalID)
		}
	}

	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 默认按 local_id 倒序 (最新的在最前)
	err := query.Order("local_id DESC").Limit(pageSize).Find(&messages).Error
	if err != nil {
		return nil, "", err
	}

	// 获取消息发送者的用户信息
	var senderIDs []uint64
	seenSenderIDs := make(map[uint64]bool)
	for _, m := range messages {
		if !seenSenderIDs[m.SenderID] {
			senderIDs = append(senderIDs, m.SenderID)
			seenSenderIDs[m.SenderID] = true
		}
	}

	userMap := make(map[uint64]models.User)
	if len(senderIDs) > 0 {
		var users []models.User
		if err := s.db.Where("uid IN ?", senderIDs).Find(&users).Error; err == nil {
			for _, u := range users {
				userMap[u.UID] = u
			}
		}
	}

	var pbMessages []*pb.Message
	var lastLocalID uint64

	for _, m := range messages {
		pbMsg := &pb.Message{
			Id:             m.ID,
			LocalId:        m.LocalID,
			ConversationId: m.ConversationID,
			SenderId:       m.SenderID,
			Content:        m.Content,
			Type:           pb.MessageType(m.Type),
			CreatedAt:      timestamppb.New(m.CreatedAt),
		}

		if u, ok := userMap[m.SenderID]; ok {
			pbMsg.Sender = &pb.SenderInfo{
				Id:       u.UID,
				Username: u.Username,
				Nickname: u.Nickname,
				Avatar:   u.Avatar,
			}
		}

		pbMessages = append(pbMessages, pbMsg)
		lastLocalID = m.LocalID
	}

	nextPageToken := ""
	if len(messages) == pageSize {
		nextPageToken = strconv.FormatUint(lastLocalID, 10)
	}

	return pbMessages, nextPageToken, nil
}

// GetConversationParticipantIDs 获取会话的所有成员ID
func (s *ChatService) GetConversationParticipantIDs(conversationID uint64) ([]uint64, error) {
	var userIDs []uint64
	err := s.db.Model(&models.ConversationParticipant{}).
		Where("conversation_id = ?", conversationID).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

// MarkAsRead 标记消息已读
func (s *ChatService) MarkAsRead(userID uint64, req *pb.MarkAsReadRequest) (uint32, error) {
	// 1. 获取最后一条已读消息的 LocalID
	var lastReadLocalID uint64
	if req.LastReadMessageId > 0 {
		var message models.Message
		// 只查询需要的字段
		if err := s.db.Select("local_id", "conversation_id").First(&message, req.LastReadMessageId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, errors.New("message not found")
			}
			return 0, err
		}
		if message.ConversationID != req.ConversationId {
			return 0, errors.New("message does not belong to this conversation")
		}
		lastReadLocalID = message.LocalID
	} else {
		// 必须要指定读到了哪条消息
		return 0, errors.New("last_read_message_id is required")
	}

	var unreadCount uint32

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 2. 更新 LastReadLocalID
		// 仅当新的 LastReadLocalID 比旧的大时才更新 (防止乱序请求导致已读回退)
		if err := tx.Model(&models.ConversationParticipant{}).
			Where("user_id = ? AND conversation_id = ? AND last_read_local_id < ?", userID, req.ConversationId, lastReadLocalID).
			Update("last_read_local_id", lastReadLocalID).Error; err != nil {
			return err
		}

		// 3. 计算最新的未读数
		var conversation models.Conversation
		if err := tx.Select("last_local_id").First(&conversation, req.ConversationId).Error; err != nil {
			return err
		}

		// 重新查询 participant 以获取确认后的 LastReadLocalID (因为上面可能更新了也可能没更新)
		var participant models.ConversationParticipant
		if err := tx.Select("last_read_local_id").
			Where("user_id = ? AND conversation_id = ?", userID, req.ConversationId).
			First(&participant).Error; err != nil {
			return err
		}

		if conversation.LastLocalID > participant.LastReadLocalID {
			unreadCount = uint32(conversation.LastLocalID - participant.LastReadLocalID)
		} else {
			unreadCount = 0
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return unreadCount, nil
}

// CreateConversation 创建会话
func (s *ChatService) CreateConversation(creatorID uint64, req *pb.CreateConversationRequest) (*models.Conversation, error) {
	// 1. 验证参数
	if len(req.ParticipantIds) == 0 {
		return nil, errors.New("participants required")
	}

	// 去重并包含创建者
	userIDs := make(map[uint64]bool)
	userIDs[creatorID] = true
	for _, uid := range req.ParticipantIds {
		userIDs[uid] = true
	}

	allParticipantIDs := make([]uint64, 0, len(userIDs))
	for uid := range userIDs {
		allParticipantIDs = append(allParticipantIDs, uid)
	}

	var existingUserCount int64
	if err := s.db.Model(&models.User{}).
		Where("uid IN ?", allParticipantIDs).
		Count(&existingUserCount).Error; err != nil {
		return nil, err
	}
	if existingUserCount != int64(len(allParticipantIDs)) {
		return nil, errors.New("some participants do not exist")
	}

	var conversation *models.Conversation
	var reqType = req.Type

	if reqType == pb.ConversationType_CONVERSATION_TYPE_PRIVATE {
		if len(userIDs) != 2 {
			return nil, errors.New("private conversation must have exactly 2 participants")
		}
		// 提取对方ID
		var otherID uint64
		for uid := range userIDs {
			if uid != creatorID {
				otherID = uid
				break
			}
		}

		// 检查关注关系
		// 必须关注对方才能发起私聊
		hasBlockingRelation, err := s.relationService.HasBlockingRelation(creatorID, otherID)
		if err != nil {
			return nil, err
		}
		if hasBlockingRelation {
			return nil, errors.New("cannot start private chat because one of you has blocked the other")
		}

		isFollowing, err := s.relationService.IsFollowing(creatorID, otherID)
		if err != nil {
			return nil, err
		}
		if !isFollowing {
			return nil, errors.New("you must follow the user to start a private chat")
		}

		// 检查是否已存在私聊
		var existingConv models.Conversation
		err = s.db.Table("conversations").
			Joins("JOIN conversation_participants p1 ON conversations.id = p1.conversation_id").
			Joins("JOIN conversation_participants p2 ON conversations.id = p2.conversation_id").
			Where("conversations.type = ? AND p1.user_id = ? AND p2.user_id = ?", models.ConversationTypePrivate, creatorID, otherID).
			First(&existingConv).Error

		if err == nil {
			// 填充对方信息
			s.fillPrivateConversationInfo(&existingConv, otherID)
			return &existingConv, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		// 不存在，创建新的
		conversation = &models.Conversation{
			Type: models.ConversationTypePrivate,
		}
	} else if reqType == pb.ConversationType_CONVERSATION_TYPE_GROUP {
		groupName := strings.TrimSpace(req.Name)
		if groupName == "" {
			return nil, errors.New("group name is required")
		}
		// 包含创建者在内，群聊至少 3 人，避免与私聊语义冲突。
		if len(userIDs) < 3 {
			return nil, errors.New("group conversation must have at least 3 participants")
		}

		conversation = &models.Conversation{
			Type:        models.ConversationTypeGroup,
			Name:        groupName,
			Avatar:      strings.TrimSpace(req.Avatar),
			OwnerID:     creatorID,
			Description: strings.TrimSpace(req.Description),
			GroupKind:   normalizeGroupKind(req.GroupKind),
			JoinMode:    normalizeJoinMode(req.JoinMode),
			Status:      models.GroupStatusActive,
		}

		for uid := range userIDs {
			if uid == creatorID {
				continue
			}
			hasBlockingRelation, err := s.relationService.HasBlockingRelation(creatorID, uid)
			if err != nil {
				return nil, err
			}
			if hasBlockingRelation {
				return nil, errors.New("cannot invite blocked users into group conversation")
			}
		}
	} else {
		return nil, errors.New("invalid conversation type")
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(conversation).Error; err != nil {
			return err
		}

		var participants []models.ConversationParticipant
		for uid := range userIDs {
			role := models.GroupRoleMember
			if uid == creatorID && reqType == pb.ConversationType_CONVERSATION_TYPE_GROUP {
				role = models.GroupRoleOwner
			}
			participants = append(participants, models.ConversationParticipant{
				ConversationID: conversation.ID,
				UserID:         uid,
				Role:           role,
				JoinedAt:       time.Now(),
			})
		}

		if err := tx.Create(&participants).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 填充私聊对象的显示信息 (仅用于返回，不存库)
	if conversation.Type == models.ConversationTypePrivate {
		// 提取对方ID (再次提取，确保准确)
		var otherID uint64
		for uid := range userIDs {
			if uid != creatorID {
				otherID = uid
				break
			}
		}
		s.fillPrivateConversationInfo(conversation, otherID)
	}

	return conversation, nil
}

// fillPrivateConversationInfo 填充私聊会话的对方信息
func (s *ChatService) fillPrivateConversationInfo(conversation *models.Conversation, otherUserID uint64) {
	var otherUser models.User
	if err := s.db.Where("uid = ?", otherUserID).First(&otherUser).Error; err == nil {
		conversation.Name = otherUser.Nickname
		if conversation.Name == "" {
			conversation.Name = otherUser.Username
		}
		conversation.Avatar = otherUser.Avatar
	}
}

// GetConversationForUser 获取指定用户视角的会话信息 (处理私聊名称/头像)
func (s *ChatService) GetConversationForUser(conversationID uint64, userID uint64) (*pb.Conversation, error) {
	var conversation models.Conversation
	if err := s.db.Preload("LastMessage").First(&conversation, conversationID).Error; err != nil {
		return nil, err
	}

	// 检查用户是否在会话中
	var participant models.ConversationParticipant
	if err := s.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&participant).Error; err != nil {
		return nil, errors.New("user not in conversation")
	}

	unreadCount := 0
	if conversation.LastLocalID > participant.LastReadLocalID {
		unreadCount = int(conversation.LastLocalID - participant.LastReadLocalID)
	}

	pbConv := &pb.Conversation{
		Id:           conversation.ID,
		Type:         pb.ConversationType(conversation.Type),
		Name:         conversation.Name,
		Avatar:       conversation.Avatar,
		UnreadCount:  uint32(unreadCount),
		UpdatedAt:    timestamppb.New(conversation.UpdatedAt),
		Description:  conversation.Description,
		Announcement: conversation.Announcement,
		GroupKind:    toPBGroupKind(conversation.GroupKind),
		JoinMode:     toPBJoinMode(conversation.JoinMode),
		Status:       toPBGroupStatus(conversation.Status),
		OwnerId:      conversation.OwnerID,
		MyRole:       toPBGroupRole(participant.Role),
	}
	if conversation.Type == models.ConversationTypeGroup {
		var memberCount int64
		s.db.Model(&models.ConversationParticipant{}).Where("conversation_id = ?", conversation.ID).Count(&memberCount)
		pbConv.MemberCount = uint32(memberCount)
	}

	if conversation.Type == models.ConversationTypePrivate {
		// 获取对方信息
		var otherParticipant models.ConversationParticipant
		if err := s.db.Where("conversation_id = ? AND user_id != ?", conversationID, userID).Preload("User").First(&otherParticipant).Error; err == nil {
			pbConv.Name = otherParticipant.User.Nickname
			if pbConv.Name == "" {
				pbConv.Name = otherParticipant.User.Username
			}
			pbConv.Avatar = otherParticipant.User.Avatar
		}
	}

	if conversation.LastMessage != nil {
		pbConv.LastMessage = &pb.Message{
			Id:             conversation.LastMessage.ID,
			LocalId:        conversation.LastMessage.LocalID,
			ConversationId: conversation.LastMessage.ConversationID,
			SenderId:       conversation.LastMessage.SenderID,
			Content:        conversation.LastMessage.Content,
			Type:           pb.MessageType(conversation.LastMessage.Type),
			CreatedAt:      timestamppb.New(conversation.LastMessage.CreatedAt),
		}
	}

	return pbConv, nil
}

func (s *ChatService) GetGroupDetail(userID uint64, conversationID uint64) (*pb.Conversation, error) {
	ctx, err := s.getGroupActorContext(s.db, conversationID, userID, false)
	if err != nil {
		return nil, err
	}
	if err := s.requireActiveGroup(ctx.Conversation); err != nil {
		return nil, err
	}
	return s.buildPBConversation(s.db, ctx.Conversation, &ctx.Participant)
}

func (s *ChatService) UpdateGroupProfile(userID uint64, req *pb.UpdateGroupProfileRequest) (*pb.Conversation, error) {
	var result *pb.Conversation
	err := s.db.Transaction(func(tx *gorm.DB) error {
		ctx, err := s.getGroupActorContext(tx, req.ConversationId, userID, true)
		if err != nil {
			return err
		}
		if err := s.requireActiveGroup(ctx.Conversation); err != nil {
			return err
		}
		if !hasAnyRole(ctx.Participant.Role, models.GroupRoleOwner, models.GroupRoleAdmin) {
			return errors.New("permission denied")
		}
		updates := map[string]interface{}{"updated_at": time.Now()}
		if name := strings.TrimSpace(req.Name); name != "" {
			updates["name"] = name
		}
		updates["avatar"] = strings.TrimSpace(req.Avatar)
		updates["description"] = strings.TrimSpace(req.Description)
		if req.JoinMode != pb.GroupJoinMode_GROUP_JOIN_MODE_UNSPECIFIED {
			updates["join_mode"] = normalizeJoinMode(req.JoinMode)
		}
		if err := tx.Model(&models.Conversation{}).Where("id = ?", req.ConversationId).Updates(updates).Error; err != nil {
			return err
		}
		var conv models.Conversation
		if err := tx.First(&conv, req.ConversationId).Error; err != nil {
			return err
		}
		result, err = s.buildPBConversation(tx, conv, &ctx.Participant)
		return err
	})
	return result, err
}

func (s *ChatService) UpdateGroupAnnouncement(userID uint64, req *pb.UpdateGroupAnnouncementRequest) (*pb.Conversation, error) {
	var result *pb.Conversation
	err := s.db.Transaction(func(tx *gorm.DB) error {
		ctx, err := s.getGroupActorContext(tx, req.ConversationId, userID, true)
		if err != nil {
			return err
		}
		if err := s.requireActiveGroup(ctx.Conversation); err != nil {
			return err
		}
		if !hasAnyRole(ctx.Participant.Role, models.GroupRoleOwner, models.GroupRoleAdmin) {
			return errors.New("permission denied")
		}
		now := time.Now()
		if err := tx.Model(&models.Conversation{}).Where("id = ?", req.ConversationId).Updates(map[string]interface{}{
			"announcement":            strings.TrimSpace(req.Announcement),
			"announcement_updated_by": userID,
			"announcement_updated_at": now,
			"updated_at":              now,
		}).Error; err != nil {
			return err
		}
		var conv models.Conversation
		if err := tx.First(&conv, req.ConversationId).Error; err != nil {
			return err
		}
		result, err = s.buildPBConversation(tx, conv, &ctx.Participant)
		return err
	})
	return result, err
}

func (s *ChatService) GetGroupMembers(userID uint64, conversationID uint64) ([]*pb.GroupMember, error) {
	ctx, err := s.getGroupActorContext(s.db, conversationID, userID, false)
	if err != nil {
		return nil, err
	}
	if err := s.requireActiveGroup(ctx.Conversation); err != nil {
		return nil, err
	}
	return s.getGroupMembersTx(s.db, conversationID)
}

func (s *ChatService) UpdateGroupMemberRole(userID uint64, req *pb.UpdateGroupMemberRoleRequest) (*pb.GroupMember, error) {
	var result *pb.GroupMember
	targetRole, err := normalizeTargetRole(req.Role)
	if err != nil {
		return nil, err
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		ctx, err := s.getGroupActorContext(tx, req.ConversationId, userID, true)
		if err != nil {
			return err
		}
		if err := s.requireActiveGroup(ctx.Conversation); err != nil {
			return err
		}
		if ctx.Participant.Role != models.GroupRoleOwner {
			return errors.New("only owner can change admin role")
		}
		var target models.ConversationParticipant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("conversation_id = ? AND user_id = ?", req.ConversationId, req.TargetUserId).Preload("User").First(&target).Error; err != nil {
			return errors.New("target member not found")
		}
		if target.Role == models.GroupRoleOwner {
			return errors.New("cannot change owner role")
		}
		if err := tx.Model(&models.ConversationParticipant{}).Where("id = ?", target.ID).Update("role", targetRole).Error; err != nil {
			return err
		}
		target.Role = targetRole
		result = s.buildPBGroupMember(target)
		return nil
	})
	return result, err
}

func (s *ChatService) TransferGroupOwnership(userID uint64, req *pb.TransferGroupOwnershipRequest) (*pb.Conversation, error) {
	var result *pb.Conversation
	err := s.db.Transaction(func(tx *gorm.DB) error {
		ctx, err := s.getGroupActorContext(tx, req.ConversationId, userID, true)
		if err != nil {
			return err
		}
		if err := s.requireActiveGroup(ctx.Conversation); err != nil {
			return err
		}
		if ctx.Participant.Role != models.GroupRoleOwner {
			return errors.New("only owner can transfer ownership")
		}
		if req.TargetUserId == userID {
			return errors.New("target user must be another group member")
		}
		var target models.ConversationParticipant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("conversation_id = ? AND user_id = ?", req.ConversationId, req.TargetUserId).First(&target).Error; err != nil {
			return errors.New("target member not found")
		}
		if err := tx.Model(&models.ConversationParticipant{}).Where("conversation_id = ? AND user_id = ?", req.ConversationId, userID).Update("role", models.GroupRoleMember).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.ConversationParticipant{}).Where("conversation_id = ? AND user_id = ?", req.ConversationId, req.TargetUserId).Update("role", models.GroupRoleOwner).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Conversation{}).Where("id = ?", req.ConversationId).Updates(map[string]interface{}{"owner_id": req.TargetUserId, "updated_at": time.Now()}).Error; err != nil {
			return err
		}
		var conv models.Conversation
		if err := tx.First(&conv, req.ConversationId).Error; err != nil {
			return err
		}
		ctx.Participant.Role = models.GroupRoleMember
		result, err = s.buildPBConversation(tx, conv, &ctx.Participant)
		return err
	})
	return result, err
}

func (s *ChatService) RemoveGroupMember(userID uint64, req *pb.RemoveGroupMemberRequest) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		ctx, err := s.getGroupActorContext(tx, req.ConversationId, userID, true)
		if err != nil {
			return err
		}
		if err := s.requireActiveGroup(ctx.Conversation); err != nil {
			return err
		}
		if req.TargetUserId == userID {
			return errors.New("use leave group instead")
		}
		var target models.ConversationParticipant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("conversation_id = ? AND user_id = ?", req.ConversationId, req.TargetUserId).First(&target).Error; err != nil {
			return errors.New("target member not found")
		}
		if !isGroupRoleManageable(ctx.Participant.Role, target.Role) {
			return errors.New("permission denied")
		}
		return tx.Where("conversation_id = ? AND user_id = ?", req.ConversationId, req.TargetUserId).Delete(&models.ConversationParticipant{}).Error
	})
}

func (s *ChatService) LeaveGroup(userID uint64, conversationID uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		ctx, err := s.getGroupActorContext(tx, conversationID, userID, true)
		if err != nil {
			return err
		}
		if err := s.requireActiveGroup(ctx.Conversation); err != nil {
			return err
		}
		if ctx.Participant.Role == models.GroupRoleOwner {
			return errors.New("owner must transfer ownership before leaving")
		}
		return tx.Where("conversation_id = ? AND user_id = ?", conversationID, userID).Delete(&models.ConversationParticipant{}).Error
	})
}

func (s *ChatService) DissolveGroup(userID uint64, conversationID uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		ctx, err := s.getGroupActorContext(tx, conversationID, userID, true)
		if err != nil {
			return err
		}
		if ctx.Participant.Role != models.GroupRoleOwner {
			return errors.New("only owner can dissolve group")
		}
		if ctx.Conversation.Status == models.GroupStatusDissolved {
			return errors.New("group has already been dissolved")
		}
		now := time.Now()
		return tx.Model(&models.Conversation{}).Where("id = ?", conversationID).Updates(map[string]interface{}{
			"status":       models.GroupStatusDissolved,
			"dissolved_at": now,
			"dissolved_by": userID,
			"updated_at":   now,
		}).Error
	})
}

func (s *ChatService) ApplyToJoinGroup(userID uint64, req *pb.ApplyToJoinGroupRequest) (*pb.GroupJoinRequest, error) {
	var result *pb.GroupJoinRequest
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var conversation models.Conversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&conversation, req.ConversationId).Error; err != nil {
			return errors.New("group not found")
		}
		if conversation.Type != models.ConversationTypeGroup {
			return errors.New("conversation is not a group")
		}
		if err := s.requireActiveGroup(conversation); err != nil {
			return err
		}
		if conversation.JoinMode == models.GroupJoinModePrivate {
			return errors.New("private group does not allow join requests")
		}
		var existingMember int64
		if err := tx.Model(&models.ConversationParticipant{}).Where("conversation_id = ? AND user_id = ?", req.ConversationId, userID).Count(&existingMember).Error; err != nil {
			return err
		}
		if existingMember > 0 {
			return errors.New("already a group member")
		}
		var existingPending int64
		if err := tx.Model(&models.GroupJoinRequest{}).Where("conversation_id = ? AND applicant_id = ? AND status = ?", req.ConversationId, userID, models.GroupJoinRequestStatusPending).Count(&existingPending).Error; err != nil {
			return err
		}
		if existingPending > 0 {
			return errors.New("join request already pending")
		}
		joinRequest := models.GroupJoinRequest{ConversationID: req.ConversationId, ApplicantID: userID, Status: models.GroupJoinRequestStatusPending, Message: strings.TrimSpace(req.Message), CreatedAt: time.Now(), UpdatedAt: time.Now()}
		if conversation.JoinMode == models.GroupJoinModePublic {
			joinRequest.Status = models.GroupJoinRequestStatusApproved
			now := time.Now()
			joinRequest.ReviewedAt = &now
			joinRequest.ReviewedBy = &conversation.OwnerID
			if err := s.addParticipantTx(tx, req.ConversationId, userID, models.GroupRoleMember); err != nil {
				return err
			}
		}
		if err := tx.Create(&joinRequest).Error; err != nil {
			return err
		}
		built, buildErr := s.buildPBJoinRequest(tx, joinRequest)
		if buildErr != nil {
			return buildErr
		}
		result = built
		return nil
	})
	return result, err
}

func (s *ChatService) GetGroupJoinRequests(userID uint64, conversationID uint64) ([]*pb.GroupJoinRequest, error) {
	ctx, err := s.getGroupActorContext(s.db, conversationID, userID, false)
	if err != nil {
		return nil, err
	}
	if !hasAnyRole(ctx.Participant.Role, models.GroupRoleOwner, models.GroupRoleAdmin) {
		return nil, errors.New("permission denied")
	}
	var requests []models.GroupJoinRequest
	if err := s.db.Where("conversation_id = ?", conversationID).Order("created_at DESC").Find(&requests).Error; err != nil {
		return nil, err
	}
	result := make([]*pb.GroupJoinRequest, 0, len(requests))
	for _, item := range requests {
		pbReq, err := s.buildPBJoinRequest(s.db, item)
		if err != nil {
			return nil, err
		}
		result = append(result, pbReq)
	}
	return result, nil
}

func (s *ChatService) ReviewGroupJoinRequest(userID uint64, req *pb.ReviewGroupJoinRequestRequest) (*pb.GroupJoinRequest, error) {
	var result *pb.GroupJoinRequest
	err := s.db.Transaction(func(tx *gorm.DB) error {
		joinRequest, err := s.getGroupJoinRequestTx(tx, req.RequestId, true)
		if err != nil {
			return err
		}
		ctx, err := s.getGroupActorContext(tx, joinRequest.ConversationID, userID, true)
		if err != nil {
			return err
		}
		if err := s.requireActiveGroup(ctx.Conversation); err != nil {
			return err
		}
		if !hasAnyRole(ctx.Participant.Role, models.GroupRoleOwner, models.GroupRoleAdmin) {
			return errors.New("permission denied")
		}
		if joinRequest.Status != models.GroupJoinRequestStatusPending {
			return errors.New("join request already processed")
		}
		now := time.Now()
		status := models.GroupJoinRequestStatusRejected
		if req.Approve {
			status = models.GroupJoinRequestStatusApproved
			if err := s.addParticipantTx(tx, joinRequest.ConversationID, joinRequest.ApplicantID, models.GroupRoleMember); err != nil {
				return err
			}
		}
		if err := tx.Model(&models.GroupJoinRequest{}).Where("id = ?", joinRequest.ID).Updates(map[string]interface{}{"status": status, "reviewed_by": userID, "reviewed_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		joinRequest.Status = status
		joinRequest.ReviewedBy = &userID
		joinRequest.ReviewedAt = &now
		built, buildErr := s.buildPBJoinRequest(tx, *joinRequest)
		if buildErr != nil {
			return buildErr
		}
		result = built
		return nil
	})
	return result, err
}

func (s *ChatService) InviteToGroup(userID uint64, req *pb.InviteToGroupRequest) (*pb.GroupInvitation, error) {
	var result *pb.GroupInvitation
	err := s.db.Transaction(func(tx *gorm.DB) error {
		ctx, err := s.getGroupActorContext(tx, req.ConversationId, userID, true)
		if err != nil {
			return err
		}
		if err := s.requireActiveGroup(ctx.Conversation); err != nil {
			return err
		}
		if !hasAnyRole(ctx.Participant.Role, models.GroupRoleOwner, models.GroupRoleAdmin) {
			return errors.New("permission denied")
		}
		var memberCount int64
		if err := tx.Model(&models.ConversationParticipant{}).Where("conversation_id = ? AND user_id = ?", req.ConversationId, req.InviteeId).Count(&memberCount).Error; err != nil {
			return err
		}
		if memberCount > 0 {
			return errors.New("user is already a group member")
		}
		var pending int64
		if err := tx.Model(&models.GroupInvitation{}).Where("conversation_id = ? AND invitee_id = ? AND status = ?", req.ConversationId, req.InviteeId, models.GroupInvitationStatusPending).Count(&pending).Error; err != nil {
			return err
		}
		if pending > 0 {
			return errors.New("invitation already pending")
		}
		invitation := models.GroupInvitation{ConversationID: req.ConversationId, InviterID: userID, InviteeID: req.InviteeId, Status: models.GroupInvitationStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		if err := tx.Create(&invitation).Error; err != nil {
			return err
		}
		built, buildErr := s.buildPBInvitation(tx, invitation)
		if buildErr != nil {
			return buildErr
		}
		result = built
		return nil
	})
	return result, err
}

func (s *ChatService) GetMyGroupInvitations(userID uint64) ([]*pb.GroupInvitation, error) {
	var invitations []models.GroupInvitation
	if err := s.db.Where("invitee_id = ?", userID).Order("created_at DESC").Find(&invitations).Error; err != nil {
		return nil, err
	}
	result := make([]*pb.GroupInvitation, 0, len(invitations))
	for _, invitation := range invitations {
		pbInvitation, err := s.buildPBInvitation(s.db, invitation)
		if err != nil {
			return nil, err
		}
		result = append(result, pbInvitation)
	}
	return result, nil
}

func (s *ChatService) RespondGroupInvitation(userID uint64, invitationID uint64, accept bool) (*pb.GroupInvitation, *pb.Conversation, error) {
	var result *pb.GroupInvitation
	var convResult *pb.Conversation
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var invitation models.GroupInvitation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&invitation, invitationID).Error; err != nil {
			return errors.New("invitation not found")
		}
		if invitation.InviteeID != userID {
			return errors.New("permission denied")
		}
		if invitation.Status != models.GroupInvitationStatusPending {
			return errors.New("invitation already processed")
		}
		var conversation models.Conversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&conversation, invitation.ConversationID).Error; err != nil {
			return errors.New("group not found")
		}
		if err := s.requireActiveGroup(conversation); err != nil {
			return err
		}
		now := time.Now()
		status := models.GroupInvitationStatusRejected
		if accept {
			status = models.GroupInvitationStatusAccepted
			if err := s.addParticipantTx(tx, invitation.ConversationID, userID, models.GroupRoleMember); err != nil {
				return err
			}
			participant := models.ConversationParticipant{Role: models.GroupRoleMember}
			builtConv, buildErr := s.buildPBConversation(tx, conversation, &participant)
			if buildErr != nil {
				return buildErr
			}
			convResult = builtConv
		}
		if err := tx.Model(&models.GroupInvitation{}).Where("id = ?", invitation.ID).Updates(map[string]interface{}{"status": status, "responded_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		invitation.Status = status
		invitation.RespondedAt = &now
		builtInvitation, buildErr := s.buildPBInvitation(tx, invitation)
		if buildErr != nil {
			return buildErr
		}
		result = builtInvitation
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if convResult != nil {
		convResult.MyRole = pb.GroupMemberRole_GROUP_MEMBER_ROLE_MEMBER
	}
	return result, convResult, nil
}
