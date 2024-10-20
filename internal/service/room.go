package service

import (
	"debate_web/internal/models"
	"debate_web/internal/repository"
	"errors"
	"fmt"
	"sync"
	"time"
)

type RoomService struct {
	roomRepo        repository.RoomRepository
	wsManager       *WebSocketManager
	messageBuffer   map[uint][]models.Message
	messageBufferMu sync.Mutex
}

func NewRoomService(roomRepo repository.RoomRepository, wsManager *WebSocketManager) *RoomService {
	return &RoomService{
		roomRepo:      roomRepo,
		wsManager:     wsManager,
		messageBuffer: make(map[uint][]models.Message),
	}
}

func (s *RoomService) CreateRoom(name, description string, maxDuration, totalRounds int) (*models.Room, error) {
	room := &models.Room{
		Name:        name,
		Description: description,
		Status:      models.RoomStatusWaiting,
		TotalRounds: totalRounds,
		MaxDuration: maxDuration,
	}

	if err := s.roomRepo.Create(room); err != nil {
		return nil, fmt.Errorf("創建房間失敗: %w", err)
	}

	return room, nil
}

func (s *RoomService) GetRoom(roomID uint) (*models.Room, error) {
	var room models.Room
	err := s.roomRepo.FindByID(roomID, &room)
	if err != nil {
		return nil, fmt.Errorf("獲取房間失敗: %w", err)
	}
	return &room, nil
}

func (s *RoomService) JoinRoom(roomID, userID uint, role string) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	if room.Status != models.RoomStatusWaiting {
		return errors.New("房間不開放加入")
	}

	switch role {
	case "proponent":
		if room.ProponentID != 0 {
			return errors.New("正方角色已被占用")
		}
		room.ProponentID = userID
	case "opponent":
		if room.OpponentID != 0 {
			return errors.New("反方角色已被占用")
		}
		room.OpponentID = userID
	case "spectator":
		room.Spectators = append(room.Spectators, userID)
	default:
		return errors.New("無效的角色")
	}

	if room.ProponentID != 0 && room.OpponentID != 0 {
		room.Status = models.RoomStatusReady
	}

	if err := s.roomRepo.Update(room); err != nil {
		return err
	}

	systemMsg := models.NewSystemMessage(roomID, fmt.Sprintf("User %d joined as %s", userID, role))
	s.wsManager.BroadcastToRoom(roomID, systemMsg.ToWebSocketMessage())
	return nil
}

func (s *RoomService) LeaveRoom(roomID, userID uint) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	switch userID {
	case room.ProponentID:
		room.ProponentID = 0
	case room.OpponentID:
		room.OpponentID = 0
	default:
		for i, spectatorID := range room.Spectators {
			if spectatorID == userID {
				room.Spectators = append(room.Spectators[:i], room.Spectators[i+1:]...)
				break
			}
		}
	}

	if room.Status == models.RoomStatusOngoing && (room.ProponentID == 0 || room.OpponentID == 0) {
		room.Status = models.RoomStatusFinished
		room.EndTime = time.Now()
	}

	if room.ProponentID == 0 && room.OpponentID == 0 && len(room.Spectators) == 0 {
		room.Status = models.RoomStatusWaiting
	}

	if err := s.roomRepo.Update(room); err != nil {
		return err
	}

	s.wsManager.DisconnectUser(roomID, userID)
	systemMsg := models.NewSystemMessage(roomID, fmt.Sprintf("User %d left the room", userID))
	s.wsManager.BroadcastToRoom(roomID, systemMsg.ToWebSocketMessage())

	return nil
}

func (s *RoomService) StartDebate(roomID uint) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	if room.Status != models.RoomStatusReady {
		return errors.New("房間尚未準備就緒")
	}

	now := time.Now()
	room.Status = models.RoomStatusOngoing
	room.StartTime = now
	room.EndTime = now.Add(time.Duration(room.MaxDuration) * time.Minute)
	room.CurrentRound = 1
	room.CurrentSpeaker = room.ProponentID
	room.CurrentRoundEnd = now.Add(time.Duration(room.RoundDuration) * time.Second)

	if err := s.roomRepo.Update(room); err != nil {
		return err
	}

	systemMsg := models.NewSystemMessage(roomID, "Debate started")
	s.wsManager.BroadcastToRoom(roomID, systemMsg.ToWebSocketMessage())
	return nil
}

func (s *RoomService) NextRound(roomID uint) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	if room.Status != models.RoomStatusOngoing {
		return errors.New("辯論尚未開始或已結束")
	}

	room.CurrentRound++
	if room.CurrentRound > room.TotalRounds {
		return s.EndDebate(roomID)
	}

	if room.CurrentSpeaker == room.ProponentID {
		room.CurrentSpeaker = room.OpponentID
	} else {
		room.CurrentSpeaker = room.ProponentID
	}

	room.CurrentRoundEnd = time.Now().Add(time.Duration(room.RoundDuration) * time.Second)

	if err := s.roomRepo.Update(room); err != nil {
		return err
	}

	systemMsg := models.NewSystemMessage(roomID, fmt.Sprintf("Round %d started", room.CurrentRound))
	s.wsManager.BroadcastToRoom(roomID, systemMsg.ToWebSocketMessage())
	return nil
}

func (s *RoomService) EndDebate(roomID uint) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	room.Status = models.RoomStatusFinished
	room.EndTime = time.Now()

	// 將緩存的消息添加到房間
	s.messageBufferMu.Lock()
	if messages, ok := s.messageBuffer[roomID]; ok {
		room.Messages = append(room.Messages, messages...)
		delete(s.messageBuffer, roomID)
	}
	s.messageBufferMu.Unlock()

	// 更新房間狀態和消息
	if err := s.roomRepo.Update(room); err != nil {
		return err
	}

	systemMsg := models.NewSystemMessage(roomID, "Debate ended")
	s.wsManager.BroadcastToRoom(roomID, systemMsg.ToWebSocketMessage())
	return nil
}

func (s *RoomService) AddMessage(roomID, userID uint, content, role string) error {
	message := models.NewDebateMessage(userID, roomID, content, role)

	s.messageBufferMu.Lock()
	s.messageBuffer[roomID] = append(s.messageBuffer[roomID], *message)
	s.messageBufferMu.Unlock()

	// 廣播消息到 WebSocket
	s.wsManager.BroadcastToRoom(roomID, message.ToWebSocketMessage())

	return nil
}

func (s *RoomService) GetDebateMessages(roomID uint) ([]models.Message, error) {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return nil, err
	}

	s.messageBufferMu.Lock()
	bufferedMessages := s.messageBuffer[roomID]
	s.messageBufferMu.Unlock()

	// 合併數據庫中的消息和緩存中的消息
	allMessages := append(room.Messages, bufferedMessages...)

	return allMessages, nil
}
