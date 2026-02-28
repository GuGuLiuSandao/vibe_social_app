package service

import (
	"social_app/internal/db"
	"social_app/internal/models"

	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService() *UserService {
	return &UserService{
		db: db.GetDB(),
	}
}

// SearchUser 搜索用户 (通过用户名或昵称模糊搜索)
func (s *UserService) SearchUser(query string) ([]models.User, error) {
	var users []models.User
	if query == "" {
		return users, nil
	}

	// 简单的模糊搜索
	likeQuery := "%" + query + "%"
	err := s.db.Where("username LIKE ? OR nickname LIKE ?", likeQuery, likeQuery).
		Limit(20). // 限制返回数量
		Find(&users).Error

	if err != nil {
		return nil, err
	}

	return users, nil
}

// GetUser gets a user by UID
func (s *UserService) GetUser(uid uint64) (*models.User, error) {
	var user models.User
	if err := s.db.Where("uid = ?", uid).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
