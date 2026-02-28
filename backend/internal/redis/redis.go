package redis

import (
	"context"
	"fmt"
	"social_app/internal/config"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	Client *redis.Client
	ctx    = context.Background()
)

const (
	onlineUsersKey   = "online:users"
	userSessionsKey  = "user:conversations:%d"
	conversationSubscribersKey = "conversation:subscribers:%d"
)

func Init(cfg *config.Config) error {
	Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return nil
}

func SetUserOnline(userID uint) error {
	return Client.SAdd(ctx, onlineUsersKey, userID).Err()
}

func SetUserOffline(userID uint) error {
	return Client.SRem(ctx, onlineUsersKey, userID).Err()
}

func IsUserOnline(userID uint) (bool, error) {
	return Client.SIsMember(ctx, onlineUsersKey, userID).Result()
}

func GetOnlineUsers() ([]uint, error) {
	members, err := Client.SMembers(ctx, onlineUsersKey).Result()
	if err != nil {
		return nil, err
	}

	users := make([]uint, 0, len(members))
	for _, m := range members {
		var id uint
		fmt.Sscanf(m, "%d", &id)
		users = append(users, id)
	}
	return users, nil
}

func SubscribeToConversation(userID, conversationID uint) error {
	err := Client.SAdd(ctx, fmt.Sprintf(userSessionsKey, userID), conversationID).Err()
	if err != nil {
		return err
	}
	return Client.SAdd(ctx, fmt.Sprintf(conversationSubscribersKey, conversationID), userID).Err()
}

func UnsubscribeFromConversation(userID, conversationID uint) error {
	err := Client.SRem(ctx, fmt.Sprintf(userSessionsKey, userID), conversationID).Err()
	if err != nil {
		return err
	}
	return Client.SRem(ctx, fmt.Sprintf(conversationSubscribersKey, conversationID), userID).Err()
}

func GetConversationSubscribers(conversationID uint) ([]uint, error) {
	members, err := Client.SMembers(ctx, fmt.Sprintf(conversationSubscribersKey, conversationID)).Result()
	if err != nil {
		return nil, err
	}

	users := make([]uint, 0, len(members))
	for _, m := range members {
		var id uint
		fmt.Sscanf(m, "%d", &id)
		users = append(users, id)
	}
	return users, nil
}

func GetUserSubscriptions(userID uint) ([]uint, error) {
	members, err := Client.SMembers(ctx, fmt.Sprintf(userSessionsKey, userID)).Result()
	if err != nil {
		return nil, err
	}

	conversations := make([]uint, 0, len(members))
	for _, m := range members {
		var id uint
		fmt.Sscanf(m, "%d", &id)
		conversations = append(conversations, id)
	}
	return conversations, nil
}

func PublishMessage(channel string, message []byte) error {
	return Client.Publish(ctx, channel, message).Err()
}

func Subscribe(channel string) *redis.PubSub {
	return Client.Subscribe(ctx, channel)
}

func SetUserSessionTTL(userID uint, ttl time.Duration) error {
	return Client.Expire(ctx, fmt.Sprintf(userSessionsKey, userID), ttl).Err()
}

func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}
