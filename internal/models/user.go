package models

import (
	"gorm.io/gorm"
)

// User 表示系統中的用戶
type User struct {
	gorm.Model        // 內嵌 gorm.Model，提供 ID、CreatedAt、UpdatedAt 和 DeletedAt 字段
	Username   string `gorm:"uniqueIndex;not null" json:"username"` // 用戶名，必須唯一
	Password   string `gorm:"not null" json:"-"`                    // 密碼，json 序列化時會被忽略
}
