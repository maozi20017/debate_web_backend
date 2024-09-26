package models

import (
	"time"

	"gorm.io/gorm"
)

// Room 表示一個辯論房間
type Room struct {
	gorm.Model
	Name           string
	Description    string
	Status         RoomStatus
	ProponentID    uint
	OpponentID     uint
	CurrentSpeaker uint
	StartTime      time.Time
	EndTime        time.Time
	MaxDuration    int // 以分鐘為單位
	CurrentRound   int
	TotalRounds    int
}

// RoomStatus 定義房間狀態的類型
type RoomStatus string

const (
	RoomStatusWaiting  RoomStatus = "waiting"
	RoomStatusReady    RoomStatus = "ready"
	RoomStatusOngoing  RoomStatus = "ongoing"
	RoomStatusFinished RoomStatus = "finished"
)
