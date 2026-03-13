package service

import (
	"errors"
	"fmt"
	"social_app/internal/models"
	accountpb "social_app/internal/proto/account"
	pb "social_app/internal/proto/relation"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type RelationService struct {
	db *gorm.DB
}

func NewRelationService(db *gorm.DB) *RelationService {
	return &RelationService{db: db}
}

// Follow creates a following relationship
func (s *RelationService) Follow(userID, targetID uint64) error {
	if userID == targetID {
		return errors.New("cannot follow yourself")
	}

	hasBlockingRelation, err := s.HasBlockingRelation(userID, targetID)
	if err != nil {
		return err
	}
	if hasBlockingRelation {
		return errors.New("cannot follow because one of you has blocked the other")
	}

	// Check if target user exists
	var targetUser models.User
	if err := s.db.Where("uid = ?", targetID).First(&targetUser).Error; err != nil {
		return fmt.Errorf("user not found with uid: %d, error: %v", targetID, err)
	}

	// Check if already following
	var rel models.Relation
	err = s.db.Where("user_id = ? AND target_id = ?", userID, targetID).First(&rel).Error
	if err == nil {
		return errors.New("already following")
	}

	// Create new relationship
	rel = models.Relation{
		UserID:   userID,
		TargetID: targetID,
	}
	return s.db.Create(&rel).Error
}

// Unfollow removes a following relationship
func (s *RelationService) Unfollow(userID, targetID uint64) error {
	result := s.db.Where("user_id = ? AND target_id = ?", userID, targetID).Delete(&models.Relation{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("relationship not found")
	}
	return nil
}

func (s *RelationService) Block(userID, targetID uint64) error {
	if userID == targetID {
		return errors.New("cannot block yourself")
	}

	var targetUser models.User
	if err := s.db.Where("uid = ?", targetID).First(&targetUser).Error; err != nil {
		return fmt.Errorf("user not found with uid: %d, error: %v", targetID, err)
	}

	var rel models.BlockRelation
	err := s.db.Where("user_id = ? AND target_id = ?", userID, targetID).First(&rel).Error
	if err == nil {
		return errors.New("already blocked")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		blockRel := models.BlockRelation{UserID: userID, TargetID: targetID}
		if err := tx.Create(&blockRel).Error; err != nil {
			return err
		}

		if err := tx.Where("(user_id = ? AND target_id = ?) OR (user_id = ? AND target_id = ?)", userID, targetID, targetID, userID).
			Delete(&models.Relation{}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *RelationService) Unblock(userID, targetID uint64) error {
	result := s.db.Where("user_id = ? AND target_id = ?", userID, targetID).Delete(&models.BlockRelation{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("block relation not found")
	}
	return nil
}

func (s *RelationService) GetBlocked(userID uint64) ([]*pb.Relation, error) {
	var relations []models.BlockRelation
	if err := s.db.Preload("Target").Where("user_id = ?", userID).Find(&relations).Error; err != nil {
		return nil, err
	}

	var pbRelations []*pb.Relation
	for _, r := range relations {
		pbRelations = append(pbRelations, &pb.Relation{
			User: &accountpb.User{
				Id:       r.Target.UID,
				Username: r.Target.Username,
				Nickname: r.Target.Nickname,
				Avatar:   r.Target.Avatar,
				Email:    r.Target.Email,
				Bio:      r.Target.Bio,
			},
			CreatedAt: r.CreatedAt.UnixMilli(),
		})
	}
	return pbRelations, nil
}

func (s *RelationService) IsBlocked(userID, targetID uint64) (bool, error) {
	var count int64
	err := s.db.Model(&models.BlockRelation{}).
		Where("user_id = ? AND target_id = ?", userID, targetID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *RelationService) HasBlockingRelation(userID, targetID uint64) (bool, error) {
	var count int64
	err := s.db.Model(&models.BlockRelation{}).
		Where("(user_id = ? AND target_id = ?) OR (user_id = ? AND target_id = ?)", userID, targetID, targetID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetFollowing returns list of users I follow
func (s *RelationService) GetFollowing(userID uint64) ([]*pb.Relation, error) {
	var relations []models.Relation
	// Preload Target info
	if err := s.db.Preload("Target").Where("user_id = ?", userID).Find(&relations).Error; err != nil {
		return nil, err
	}

	var pbRelations []*pb.Relation
	for _, r := range relations {
		pbRelations = append(pbRelations, &pb.Relation{
			User: &accountpb.User{
				Id:       r.Target.UID,
				Username: r.Target.Username,
				Nickname: r.Target.Nickname,
				Avatar:   r.Target.Avatar,
				Email:    r.Target.Email,
				Bio:      r.Target.Bio,
			},
			CreatedAt: r.CreatedAt.UnixMilli(),
		})
	}
	return pbRelations, nil
}

// GetFollowers returns list of users following me
func (s *RelationService) GetFollowers(userID uint64) ([]*pb.Relation, error) {
	var relations []models.Relation
	// Preload User (Follower) info
	if err := s.db.Preload("User").Where("target_id = ?", userID).Find(&relations).Error; err != nil {
		return nil, err
	}

	var pbRelations []*pb.Relation
	for _, r := range relations {
		pbRelations = append(pbRelations, &pb.Relation{
			User: &accountpb.User{
				Id:       r.User.UID,
				Username: r.User.Username,
				Nickname: r.User.Nickname,
				Avatar:   r.User.Avatar,
				Email:    r.User.Email,
				Bio:      r.User.Bio,
			},
			CreatedAt: r.CreatedAt.UnixMilli(),
		})
	}
	return pbRelations, nil
}

// IsFollowing checks if userID follows targetID
func (s *RelationService) IsFollowing(userID, targetID uint64) (bool, error) {
	var count int64
	err := s.db.Model(&models.Relation{}).
		Where("user_id = ? AND target_id = ?", userID, targetID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetFriends returns list of mutual friends (bidirectional follow)
func (s *RelationService) GetFriends(userID uint64, limit int32, cursor string) ([]*pb.Relation, string, error) {
	var relations []models.Relation

	// Subquery: Users who follow me
	followersSubQuery := s.db.Model(&models.Relation{}).Select("user_id").Where("target_id = ?", userID)

	// Main query: Users I follow AND who follow me
	query := s.db.Preload("Target").
		Where("user_id = ?", userID).
		Where("target_id IN (?)", followersSubQuery).
		Order("created_at DESC") // Sort by follow time

	if limit > 0 {
		query = query.Limit(int(limit))
	} else {
		query = query.Limit(20) // Default limit
	}

	if cursor != "" {
		// Cursor is the timestamp (milliseconds)
		ts, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil {
			query = query.Where("created_at < ?", time.UnixMilli(ts))
		}
	}

	if err := query.Find(&relations).Error; err != nil {
		return nil, "", err
	}

	var pbRelations []*pb.Relation
	var nextCursor string

	for _, r := range relations {
		pbRelations = append(pbRelations, &pb.Relation{
			User: &accountpb.User{
				Id:       r.Target.UID,
				Username: r.Target.Username,
				Nickname: r.Target.Nickname,
				Avatar:   r.Target.Avatar,
				Email:    r.Target.Email,
				Bio:      r.Target.Bio,
			},
			CreatedAt: r.CreatedAt.UnixMilli(),
		})
		// Update nextCursor to the last item's timestamp
		nextCursor = fmt.Sprintf("%d", r.CreatedAt.UnixMilli())
	}

	return pbRelations, nextCursor, nil
}
