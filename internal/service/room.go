package service

import (
	"debate_web/internal/models"
	"debate_web/internal/repository"
	"errors"
	"strconv"
	"time"
)

type RoomService struct {
	roomRepo    repository.RoomRepository
	wsManager   *WebSocketManager
	messageRepo repository.DebateMessageRepository
}

func NewRoomService(roomRepo repository.RoomRepository, wsManager *WebSocketManager, messageRepo repository.DebateMessageRepository) *RoomService {
	return &RoomService{
		roomRepo:    roomRepo,
		wsManager:   wsManager,
		messageRepo: messageRepo,
	}
}

func (s *RoomService) GetDebateMessages(roomID uint) ([]models.DebateMessage, error) {
	return s.messageRepo.FindByRoomID(roomID)
}

func (s *RoomService) CreateRoom(name, description string, maxDuration, totalRounds int) (*models.Room, error) {
	room := &models.Room{
		Name:        name,
		Description: description,
		Status:      models.RoomStatusWaiting,
		MaxDuration: maxDuration,
		TotalRounds: totalRounds,
	}
	err := s.roomRepo.Create(room)
	if err != nil {
		return nil, err
	}
	return room, nil
}

// GetRoom 根據房間 ID 獲取房間信息
func (s *RoomService) GetRoom(roomID uint) (*models.Room, error) {
	room, err := s.roomRepo.FindByID(roomID)
	if err != nil {
		return nil, err
	}
	return room, nil
}

// JoinRoom 使用者加入指定的辯論房間
func (s *RoomService) JoinRoom(roomID, userID uint, role string) error {
	// 使用 repository 獲取房間信息
	room, err := s.roomRepo.FindByID(roomID)
	if err != nil {
		return err
	}

	if room.Status != "waiting" {
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
	default:
		return errors.New("無效的角色")
	}

	if room.ProponentID != 0 && room.OpponentID != 0 {
		room.Status = "ready"
	}

	// 使用 repository 更新房間信息
	return s.roomRepo.Update(room)
}

// StartDebate 開始指定房間的辯論
func (s *RoomService) StartDebate(roomID uint) error {
	room, err := s.roomRepo.FindByID(roomID)
	if err != nil {
		return err
	}

	if room.Status != models.RoomStatusReady {
		return errors.New("房間尚未準備就緒")
	}

	room.Status = models.RoomStatusOngoing
	room.StartTime = time.Now()
	room.EndTime = room.StartTime.Add(time.Duration(room.MaxDuration) * time.Minute)
	room.CurrentRound = 1
	room.CurrentSpeaker = room.ProponentID // 假設正方先開始

	err = s.roomRepo.Update(room)
	if err != nil {
		return err
	}

	s.wsManager.BroadcastSystemMessage(roomID, "辯論開始，第1回合，正方發言")
	return nil
}

func (s *RoomService) NextRound(roomID uint) error {
	room, err := s.roomRepo.FindByID(roomID)
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
		s.wsManager.BroadcastSystemMessage(roomID, "第"+strconv.Itoa(room.CurrentRound)+"回合，反方發言")
	} else {
		room.CurrentSpeaker = room.ProponentID
		s.wsManager.BroadcastSystemMessage(roomID, "第"+strconv.Itoa(room.CurrentRound)+"回合，正方發言")
	}

	return s.roomRepo.Update(room)
}

// EndDebate 結束指定房間的辯論
func (s *RoomService) EndDebate(roomID uint) error {
	room, err := s.roomRepo.FindByID(roomID)
	if err != nil {
		return err
	}

	if room.Status != models.RoomStatusOngoing {
		return errors.New("辯論尚未開始或已結束")
	}

	room.Status = models.RoomStatusFinished
	room.EndTime = time.Now()

	err = s.roomRepo.Update(room)
	if err != nil {
		return err
	}

	s.wsManager.BroadcastSystemMessage(roomID, "辯論結束")
	return nil
}