package models

import (
	"time"

	"gorm.io/gorm"
)

// Room 表示一個辯論房間
type Room struct {
	gorm.Model
	Name        string
	Status      RoomStatus // "waiting", "ongoing", "finished"
	ProponentID uint
	OpponentID  uint
	StartTime   time.Time
	EndTime     time.Time
	Messages    []Message
	Spectators  []uint
}

// RoomStatus 定義房間狀態的類型
type RoomStatus string

const (
	RoomStatusWaiting  RoomStatus = "waiting"
	RoomStatusReady    RoomStatus = "ready"
	RoomStatusOngoing  RoomStatus = "ongoing"
	RoomStatusFinished RoomStatus = "finished"
)
