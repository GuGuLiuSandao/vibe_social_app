package main

import (
	"flag"
	"fmt"
	"log"
	"social_app/internal/config"
	"social_app/internal/db"
	"social_app/internal/logger"
	"social_app/internal/models"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func main() {
	startUID := flag.Uint64("start", 10000000, "seed range start uid (inclusive)")
	endUID := flag.Uint64("end", 10000100, "seed range end uid (inclusive)")
	followSpan := flag.Int("follow-span", 4, "each user follows next N users in ring")
	messagesPerConversation := flag.Int("messages-per-conv", 6, "messages generated for each private conversation")
	flag.Parse()

	if *endUID < *startUID {
		log.Fatalf("invalid range: end (%d) < start (%d)", *endUID, *startUID)
	}
	if *followSpan < 1 {
		log.Fatalf("follow-span must be >= 1")
	}
	if *messagesPerConversation < 1 {
		log.Fatalf("messages-per-conv must be >= 1")
	}

	logger.Init()
	cfg := config.Load()
	if err := db.Init(cfg); err != nil {
		log.Fatalf("failed to init db: %v", err)
	}
	d := db.GetDB()

	uids := make([]uint64, 0, int(*endUID-*startUID+1))
	for uid := *startUID; uid <= *endUID; uid++ {
		uids = append(uids, uid)
	}

	now := time.Now().UTC().Truncate(time.Second)
	if err := upsertUsers(d, uids, now); err != nil {
		log.Fatalf("failed to seed users: %v", err)
	}

	relations, err := seedRelations(d, *startUID, *endUID, uids, *followSpan, now)
	if err != nil {
		log.Fatalf("failed to seed relations: %v", err)
	}

	conversations, messages, err := seedPrivateConversations(d, uids, *messagesPerConversation, now)
	if err != nil {
		log.Fatalf("failed to seed private conversations: %v", err)
	}

	fmt.Printf(
		"seed done: users=%d, relations=%d, private_conversations=%d, messages=%d, range=%d-%d\n",
		len(uids),
		relations,
		conversations,
		messages,
		*startUID,
		*endUID,
	)
}

func upsertUsers(d *gorm.DB, uids []uint64, baseTime time.Time) error {
	for i, uid := range uids {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(strconv.FormatUint(uid, 10)), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		createdAt := baseTime.Add(time.Duration(i) * time.Minute)
		user := models.User{
			ID:        uid,
			UID:       uid,
			Username:  fmt.Sprintf("seed_%d", uid),
			Email:     fmt.Sprintf("seed_%d@test.local", uid),
			Password:  string(passwordHash),
			Nickname:  buildNickname(i, uid),
			Avatar:    fmt.Sprintf("https://api.dicebear.com/8.x/identicon/svg?seed=%d", uid),
			Bio:       fmt.Sprintf("Seed account %d for social graph and private chat testing", uid),
			Gender:    "unknown",
			Location:  "Neo City",
			Status:    1,
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		}

		err = d.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "uid"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"username":   user.Username,
				"email":      user.Email,
				"password":   user.Password,
				"nickname":   user.Nickname,
				"avatar":     user.Avatar,
				"bio":        user.Bio,
				"gender":     user.Gender,
				"location":   user.Location,
				"status":     user.Status,
				"updated_at": time.Now().UTC(),
			}),
		}).Create(&user).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func seedRelations(d *gorm.DB, startUID, endUID uint64, uids []uint64, followSpan int, baseTime time.Time) (int, error) {
	if len(uids) <= 1 {
		return 0, nil
	}

	if err := d.Where("user_id BETWEEN ? AND ? AND target_id BETWEEN ? AND ?", startUID, endUID, startUID, endUID).
		Delete(&models.Relation{}).Error; err != nil {
		return 0, err
	}

	maxSpan := followSpan
	if maxSpan >= len(uids) {
		maxSpan = len(uids) - 1
	}

	relations := make([]models.Relation, 0, len(uids)*maxSpan)
	for i, uid := range uids {
		for step := 1; step <= maxSpan; step++ {
			target := uids[(i+step)%len(uids)]
			if target == uid {
				continue
			}
			relations = append(relations, models.Relation{
				UserID:    uid,
				TargetID:  target,
				CreatedAt: baseTime.Add(time.Duration(i+step) * time.Second),
			})
		}
	}

	if len(relations) == 0 {
		return 0, nil
	}

	if err := d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "target_id"}},
		DoNothing: true,
	}).CreateInBatches(relations, 1000).Error; err != nil {
		return 0, err
	}

	return len(relations), nil
}

func seedPrivateConversations(d *gorm.DB, uids []uint64, messagesPerConversation int, baseTime time.Time) (int, int, error) {
	if len(uids) <= 1 {
		return 0, 0, nil
	}

	var oldConvIDs []uint64
	if err := d.Model(&models.Conversation{}).
		Where("type = ? AND name LIKE ?", models.ConversationTypePrivate, "seed:%").
		Pluck("id", &oldConvIDs).Error; err != nil {
		return 0, 0, err
	}

	if len(oldConvIDs) > 0 {
		if err := d.Where("conversation_id IN ?", oldConvIDs).Delete(&models.Message{}).Error; err != nil {
			return 0, 0, err
		}
		if err := d.Where("conversation_id IN ?", oldConvIDs).Delete(&models.ConversationParticipant{}).Error; err != nil {
			return 0, 0, err
		}
		if err := d.Where("id IN ?", oldConvIDs).Delete(&models.Conversation{}).Error; err != nil {
			return 0, 0, err
		}
	}

	conversationCount := 0
	messageCount := 0

	for i, uidA := range uids {
		uidB := uids[(i+1)%len(uids)]
		left, right := uidA, uidB
		if left > right {
			left, right = right, left
		}

		convCreatedAt := baseTime.Add(time.Duration(i) * 5 * time.Minute)
		conv := models.Conversation{
			Type:      models.ConversationTypePrivate,
			Name:      fmt.Sprintf("seed:%d:%d", left, right),
			OwnerID:   uidA,
			CreatedAt: convCreatedAt,
			UpdatedAt: convCreatedAt,
		}
		if err := d.Create(&conv).Error; err != nil {
			return 0, 0, err
		}

		unreadForB := 2
		if messagesPerConversation < 2 {
			unreadForB = 0
		}
		lastReadB := uint64(messagesPerConversation - unreadForB)

		participants := []models.ConversationParticipant{
			{
				ConversationID:  conv.ID,
				UserID:          uidA,
				Role:            "member",
				UnreadCount:     0,
				LastReadLocalID: uint64(messagesPerConversation),
				JoinedAt:        convCreatedAt,
			},
			{
				ConversationID:  conv.ID,
				UserID:          uidB,
				Role:            "member",
				UnreadCount:     unreadForB,
				LastReadLocalID: lastReadB,
				JoinedAt:        convCreatedAt,
			},
		}
		if err := d.Create(&participants).Error; err != nil {
			return 0, 0, err
		}

		var lastMsgID uint64
		var lastMsgTime time.Time
		for m := 1; m <= messagesPerConversation; m++ {
			sender := uidA
			if m%2 == 0 {
				sender = uidB
			}

			createdAt := convCreatedAt.Add(time.Duration(m) * 40 * time.Second)
			content := fmt.Sprintf("seed chat %02d/%02d between %d and %d", m, messagesPerConversation, uidA, uidB)
			msg := models.Message{
				ConversationID: conv.ID,
				LocalID:        uint64(m),
				SenderID:       sender,
				Content:        content,
				Type:           1,
				CreatedAt:      createdAt,
			}
			if err := d.Create(&msg).Error; err != nil {
				return 0, 0, err
			}
			lastMsgID = msg.ID
			lastMsgTime = createdAt
			messageCount++
		}

		if err := d.Model(&models.Conversation{}).
			Where("id = ?", conv.ID).
			Updates(map[string]interface{}{
				"last_local_id":   uint64(messagesPerConversation),
				"last_message_id": lastMsgID,
				"updated_at":      lastMsgTime,
			}).Error; err != nil {
			return 0, 0, err
		}

		conversationCount++
	}

	return conversationCount, messageCount, nil
}

func buildNickname(index int, uid uint64) string {
	adjectives := []string{
		"冷焰", "银霜", "蓝域", "夜航", "静潮",
		"极光", "量子", "深空", "零界", "浮光",
	}
	nouns := []string{
		"引擎", "信标", "矩阵", "节点", "回响",
		"棱镜", "核心", "轨迹", "波形", "镜像",
	}
	return fmt.Sprintf("%s%s-%03d", adjectives[index%len(adjectives)], nouns[index%len(nouns)], uid%1000)
}
