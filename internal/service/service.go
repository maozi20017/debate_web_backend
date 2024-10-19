package service

import (
	"debate_web/internal/repository"
)

type Services struct {
	UserService      *UserService
	RoomService      *RoomService
	WebSocketManager *WebSocketManager
}

func NewServices(repos *repository.Repositories) *Services {
	// 初始化 WebSocketManager
	wsManager := NewWebSocketManager()

	// 初始化 UserService
	userService := NewUserService(repos.User)

	// 初始化 RoomService，傳入 WebSocketManager
	roomService := NewRoomService(repos.Room, wsManager)

	return &Services{
		UserService:      userService,
		RoomService:      roomService,
		WebSocketManager: wsManager,
	}
}
