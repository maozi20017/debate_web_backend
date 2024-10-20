package models

import (
	"time"

	"gorm.io/gorm"
)

// Message 代表一個統一的消息結構，同時滿足 WebSocket 和數據庫存儲需求
type Message struct {
	gorm.Model
	Type      string    `json:"type" gorm:"type:varchar(50)"`
	Content   string    `json:"content" gorm:"type:text"`
	UserID    uint      `json:"user_id"`
	RoomID    uint      `json:"room_id"`
	Role      string    `json:"role" gorm:"type:varchar(20)"`
	Timestamp time.Time `json:"timestamp"`
	// 可選字段，用於存儲額外的 JSON 數據
	ExtraData string `json:"extra_data,omitempty" gorm:"type:jsonb"`
}

// NewDebateMessage 創建一個新的辯論消息
func NewDebateMessage(userID, roomID uint, content, role string) Message {
	return Message{
		Type:      "debate_message",
		Content:   content,
		UserID:    userID,
		RoomID:    roomID,
		Role:      role,
		Timestamp: time.Now(),
	}
}

// NewSystemMessage 創建一個新的系統消息
func NewSystemMessage(roomID uint, content string) Message {
	return Message{
		Type:      "system_message",
		Content:   content,
		RoomID:    roomID,
		Timestamp: time.Now(),
	}
}
