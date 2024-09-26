package repository

import (
	"debate_web/internal/models"
	"debate_web/internal/storage"
)

type UserRepository interface {
	Create(user *models.User) error
	FindByUsername(username string) (*models.User, error)
	// 可以根據需要添加其他方法
}

type userRepository struct {
	db *storage.PostgresDB
}

func NewUserRepository(db *storage.PostgresDB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) FindByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
