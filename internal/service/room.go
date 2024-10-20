package service

import (
	"debate_web/internal/models"
	"debate_web/internal/repository"
	"errors"
	"fmt"
	"sync"
	"time"
)

// RoomService 處理與辯論房間相關的業務邏輯
type RoomService struct {
	roomRepo        repository.RoomRepository
	wsManager       *WebSocketManager
	messageBuffer   map[uint][]models.Message
	messageBufferMu sync.Mutex
}

// NewRoomService 創建一個新的 RoomService 實例
func NewRoomService(roomRepo repository.RoomRepository, wsManager *WebSocketManager) *RoomService {
	return &RoomService{
		roomRepo:      roomRepo,
		wsManager:     wsManager,
		messageBuffer: make(map[uint][]models.Message),
	}
}

// CreateRoom 創建一個新的辯論房間
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

// GetRoom 根據 ID 獲取房間信息
func (s *RoomService) GetRoom(roomID uint) (*models.Room, error) {
	var room models.Room
	err := s.roomRepo.FindByID(roomID, &room)
	if err != nil {
		return nil, fmt.Errorf("獲取房間失敗: %w", err)
	}
	return &room, nil
}

// JoinRoom 處理用戶加入房間的邏輯
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
	if err := s.AddMessage(roomID, 0, systemMsg.Content, "system"); err != nil {
		return fmt.Errorf("保存系統消息失敗: %w", err)
	}
	s.wsManager.BroadcastToRoom(roomID, systemMsg)
	return nil
}

// LeaveRoom 處理用戶離開房間的邏輯
func (s *RoomService) LeaveRoom(roomID, userID uint) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	isDebater := false
	switch userID {
	case room.ProponentID:
		room.ProponentID = 0
		isDebater = true
	case room.OpponentID:
		room.OpponentID = 0
		isDebater = true
	default:
		for i, spectatorID := range room.Spectators {
			if spectatorID == userID {
				room.Spectators = append(room.Spectators[:i], room.Spectators[i+1:]...)
				break
			}
		}
	}

	// 如果是辯論進行中，且離開的是辯論者，則結束辯論
	if room.Status == models.RoomStatusOngoing && isDebater {
		room.Status = models.RoomStatusFinished
		room.EndTime = time.Now()

		// 保存辯論結束狀態到數據庫
		if err := s.roomRepo.Update(room); err != nil {
			return fmt.Errorf("更新辯論結束狀態失敗: %w", err)
		}

		// 發送系統消息
		systemMsg := models.NewSystemMessage(roomID, "辯論結束：一名辯論者離開了房間")
		if err := s.AddMessage(roomID, 0, systemMsg.Content, "system"); err != nil {
			return fmt.Errorf("添加系統消息失敗: %w", err)
		}

		s.wsManager.BroadcastToRoom(roomID, systemMsg)
	} else if room.ProponentID == 0 && room.OpponentID == 0 && len(room.Spectators) == 0 {
		// 如果房間沒有人了，將狀態設置為等待中
		room.Status = models.RoomStatusWaiting
	}

	// 再次更新房間狀態，以確保所有變更都被保存
	if err := s.roomRepo.Update(room); err != nil {
		return fmt.Errorf("更新房間狀態失敗: %w", err)
	}

	// 斷開用戶的 WebSocket 連接
	s.wsManager.DisconnectUser(roomID, userID)

	// 發送用戶離開的系統消息
	leaveMsg := models.NewSystemMessage(roomID, fmt.Sprintf("使用者 %d 離開了房間", userID))
	if err := s.AddMessage(roomID, 0, leaveMsg.Content, "system"); err != nil {
		return fmt.Errorf("添加用戶離開消息失敗: %w", err)
	}

	s.wsManager.BroadcastToRoom(roomID, leaveMsg)

	return nil
}

// StartDebate 開始辯論
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

	systemMsg := models.NewSystemMessage(roomID, "辯論開始")
	if err := s.AddMessage(roomID, 0, systemMsg.Content, "system"); err != nil {
		return fmt.Errorf("保存開始辯論系統消息失敗: %w", err)
	}
	s.wsManager.BroadcastToRoom(roomID, systemMsg)
	return nil
}

// NextRound 進入下一輪辯論
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

	systemMsg := models.NewSystemMessage(roomID, fmt.Sprintf("第 %d 回合開始", room.CurrentRound))
	if err := s.AddMessage(roomID, 0, systemMsg.Content, "system"); err != nil {
		return fmt.Errorf("保存下一回合系統消息失敗: %w", err)
	}
	s.wsManager.BroadcastToRoom(roomID, systemMsg)
	return nil
}

// EndDebate 結束辯論並更新房間狀態
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
		return fmt.Errorf("更新房間狀態失敗: %w", err)
	}

	systemMsg := models.NewSystemMessage(roomID, "辯論結束")
	// 將系統消息保存到數據庫
	if err := s.AddMessage(roomID, 0, systemMsg.Content, "system"); err != nil {
		return fmt.Errorf("保存辯論結束系統消息失敗: %w", err)
	}
	s.wsManager.BroadcastToRoom(roomID, systemMsg)
	return nil
}

// AddMessage 添加新消息到房間並廣播
func (s *RoomService) AddMessage(roomID, userID uint, content, role string) error {
	message := models.NewDebateMessage(userID, roomID, content, role)

	// 將消息添加到緩存
	s.messageBufferMu.Lock()
	s.messageBuffer[roomID] = append(s.messageBuffer[roomID], message)
	s.messageBufferMu.Unlock()

	// 廣播消息到 WebSocket
	s.wsManager.BroadcastToRoom(roomID, message)

	return nil
}

// GetDebateMessages 獲取辯論的所有消息
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
