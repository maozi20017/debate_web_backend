package service

import "debate_web/internal/repository"

type Services struct {
	User      *UserService
	Room      *RoomService
	WebSocket *WebSocketService
}

func NewServices(repos *repository.Repositories) *Services {
	ws := NewWebSocketService()

	return &Services{
		User:      NewUserService(repos.User),
		Room:      NewRoomService(repos.Room, ws),
		WebSocket: ws,
	}
}
