package models

import (
	"gorm.io/gorm"
)

// Message 代表一個統一的消息結構，同時滿足 WebSocket 和數據庫存儲需求
type Message struct {
	gorm.Model
	Type    string
	RoomID  uint
	UserID  uint
	Content string
	Role    string // "proponent", "opponent", "system"
}
