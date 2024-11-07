package service

import (
	"debate_web/internal/repository"
	"debate_web/internal/repository/models"
	"debate_web/internal/utils"
	"errors"

	"gorm.io/gorm"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// CheckUserExists 檢查用戶是否存在
func (s *UserService) CheckUserExists(username string) (bool, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		// 如果是找不到記錄的錯誤，返回 false 而不是錯誤
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return user != nil, nil
}

func (s *UserService) CreateUser(user *models.User) error {
	// 檢查用戶名是否存在
	exists, err := s.CheckUserExists(user.Username)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("用戶名已被使用")
	}

	return s.repo.Create(user)
}

func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	return s.repo.FindByUsername(username)
}

func (s *UserService) GenerateToken(userID uint) (string, error) {
	return utils.GenerateToken(userID)
}
