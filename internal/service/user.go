package service

import (
	"debate_web/internal/models"
	"debate_web/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) CreateUser(user *models.User) error {
	return s.userRepo.Create(user)
}

func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	return s.userRepo.FindByUsername(username)
}
