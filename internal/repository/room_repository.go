package repository

import (
	"debate_web/internal/models"
	"debate_web/internal/storage"
)

type RoomRepository interface {
	Create(room *models.Room) error
	FindByID(id uint) (*models.Room, error)
	Update(room *models.Room) error
	// 可以根據需要添加其他方法
}

type roomRepository struct {
	db *storage.PostgresDB
}

func NewRoomRepository(db *storage.PostgresDB) RoomRepository {
	return &roomRepository{db: db}
}

func (r *roomRepository) Create(room *models.Room) error {
	return r.db.Create(room).Error
}

func (r *roomRepository) FindByID(id uint) (*models.Room, error) {
	var room models.Room
	err := r.db.First(&room, id).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *roomRepository) Update(room *models.Room) error {
	return r.db.Save(room).Error
}
