package models

import (
	"time"

	"gorm.io/gorm"
)

type DebateMessage struct {
	gorm.Model
	RoomID    uint      `json:"room_id"`
	UserID    uint      `json:"user_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
