package repository

import (
	"debate_web/internal/models"
	"debate_web/internal/storage"
)

type DebateMessageRepository interface {
	Create(message *models.DebateMessage) error
	FindByRoomID(roomID uint) ([]models.DebateMessage, error)
}

type debateMessageRepository struct {
	db *storage.PostgresDB
}

func NewDebateMessageRepository(db *storage.PostgresDB) DebateMessageRepository {
	return &debateMessageRepository{db: db}
}

func (r *debateMessageRepository) Create(message *models.DebateMessage) error {
	return r.db.Create(message).Error
}

func (r *debateMessageRepository) FindByRoomID(roomID uint) ([]models.DebateMessage, error) {
	var messages []models.DebateMessage
	err := r.db.Where("room_id = ?", roomID).Order("timestamp asc").Find(&messages).Error
	return messages, err
}
