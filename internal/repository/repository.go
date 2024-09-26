package repository

import "debate_web/internal/storage"

type Repositories struct {
	User          UserRepository
	Room          RoomRepository
	DebateMessage DebateMessageRepository
}

func NewRepositories(db *storage.PostgresDB) *Repositories {
	return &Repositories{
		User:          NewUserRepository(db),
		Room:          NewRoomRepository(db),
		DebateMessage: NewDebateMessageRepository(db),
	}
}
