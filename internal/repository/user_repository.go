package repository

import (
	"debate_web/internal/models"
	"debate_web/internal/storage"
)

type UserRepository interface {
	BaseRepository
	FindByUsername(username string) (*models.User, error)
}

type userRepository struct {
	BaseRepository
	db *storage.PostgresDB
}

func NewUserRepository(db *storage.PostgresDB) UserRepository {
	return &userRepository{
		BaseRepository: NewBaseRepository(db),
		db:             db,
	}
}

func (r *userRepository) FindByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
