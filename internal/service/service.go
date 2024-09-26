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
	wsManager := NewWebSocketManager(repos.DebateMessage)

	userService := NewUserService(repos.User)
	roomService := NewRoomService(repos.Room, wsManager, repos.DebateMessage)
	return &Services{
		UserService:      userService,
		RoomService:      roomService,
		WebSocketManager: wsManager,
	}
}
