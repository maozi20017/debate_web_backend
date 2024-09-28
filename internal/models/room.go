package models

import (
	"time"

	"gorm.io/gorm"
)

// Room 表示一個辯論房間
type Room struct {
	gorm.Model
	Name            string
	Description     string
	Status          RoomStatus
	ProponentID     uint
	OpponentID      uint
	CurrentSpeaker  uint
	StartTime       time.Time
	EndTime         time.Time
	MaxDuration     int // 以分鐘為單位
	CurrentRound    int
	TotalRounds     int
	RoundDuration   int       // 每回合的持續時間（秒）
	CurrentRoundEnd time.Time // 當前回合的結束時間
	Spectators      []uint    `gorm:"type:integer[]"` // 觀眾的用戶 ID 列表
}

// RoomStatus 定義房間狀態的類型
type RoomStatus string

const (
	RoomStatusWaiting  RoomStatus = "waiting"
	RoomStatusReady    RoomStatus = "ready"
	RoomStatusOngoing  RoomStatus = "ongoing"
	RoomStatusFinished RoomStatus = "finished"
)
