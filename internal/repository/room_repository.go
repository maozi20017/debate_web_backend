package repository

import (
	"debate_web/internal/repository/models"
	"debate_web/internal/storage"
)

type RoomRepository interface {
	Create(room *models.Room) error
	FindByID(id uint) (*models.Room, error)
	Update(room *models.Room) error
	Delete(id uint) error
	FindAll() ([]models.Room, error) // 簡單的列表查詢
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

func (r *roomRepository) Delete(id uint) error {
	return r.db.Delete(&models.Room{}, id).Error
}

// FindAll 查詢所有房間
func (r *roomRepository) FindAll() ([]models.Room, error) {
	var rooms []models.Room
	err := r.db.Order("created_at DESC").Find(&rooms).Error
	return rooms, err
}
