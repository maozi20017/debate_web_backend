package repository

import (
	"debate_web/internal/storage"
)

type RoomRepository interface {
	BaseRepository
	// 可以在這裡添加特定於 Room 的方法
}

type roomRepository struct {
	BaseRepository
	db *storage.PostgresDB
}

func NewRoomRepository(db *storage.PostgresDB) RoomRepository {
	return &roomRepository{
		BaseRepository: NewBaseRepository(db),
		db:             db,
	}
}
