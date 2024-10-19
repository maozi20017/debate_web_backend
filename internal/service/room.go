package service

import (
	"debate_web/internal/models"
	"debate_web/internal/repository"
	"errors"
	"strconv"
	"time"
)

// Room 代表一個辯論房間
type Room struct {
	ID              uint
	Name            string
	Description     string
	Status          string
	ProponentID     uint
	OpponentID      uint
	CurrentSpeaker  uint
	CurrentRound    int
	TotalRounds     int
	RoundDuration   int
	StartTime       time.Time
	EndTime         time.Time
	CurrentRoundEnd time.Time
	Spectators      []uint
}

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

func (s *RoomService) GetRoom(roomID uint) (*Room, error) {
	roomModel, err := s.roomRepo.FindByID(roomID)
	if err != nil {
		return nil, err
	}

	return s.convertModelToRoom(roomModel), nil
}

func (s *RoomService) CreateRoom(name, description string, maxDuration, totalRounds int) (*Room, error) {
	roomModel := &models.Room{
		Name:        name,
		Description: description,
		Status:      models.RoomStatusWaiting,
		TotalRounds: totalRounds,
		MaxDuration: maxDuration,
	}

	err := s.roomRepo.Create(roomModel)
	if err != nil {
		return nil, err
	}

	return s.convertModelToRoom(roomModel), nil
}

func (s *RoomService) JoinRoom(roomID, userID uint, role string) error {
	room, err := s.GetRoom(roomID)
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
	case "spectator":
		return s.AddSpectator(roomID, userID)
	default:
		return errors.New("無效的角色")
	}

	if room.ProponentID != 0 && room.OpponentID != 0 {
		room.Status = "ready"
	}

	return s.updateRoom(room)
}

func (s *RoomService) StartDebate(roomID uint) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	if room.Status != "ready" {
		return errors.New("房間尚未準備就緒")
	}

	room.Status = "ongoing"
	room.StartTime = time.Now()
	room.EndTime = room.StartTime.Add(time.Duration(room.TotalRounds*room.RoundDuration) * time.Second)
	room.CurrentRound = 1
	room.CurrentSpeaker = room.ProponentID
	room.CurrentRoundEnd = time.Now().Add(time.Duration(room.RoundDuration) * time.Second)

	err = s.updateRoom(room)
	if err != nil {
		return err
	}

	s.wsManager.BroadcastSystemMessage(roomID, "辯論開始，第1回合，正方發言")
	return nil
}

func (s *RoomService) NextRound(roomID uint) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	if room.Status != "ongoing" {
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

	room.CurrentRoundEnd = time.Now().Add(time.Duration(room.RoundDuration) * time.Second)

	return s.updateRoom(room)
}

func (s *RoomService) EndDebate(roomID uint) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	if room.Status != "ongoing" {
		return errors.New("辯論尚未開始或已結束")
	}

	room.Status = "finished"
	room.EndTime = time.Now()

	err = s.updateRoom(room)
	if err != nil {
		return err
	}

	s.wsManager.BroadcastSystemMessage(roomID, "辯論結束")
	return nil
}

func (s *RoomService) AddSpectator(roomID, userID uint) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	for _, spectatorID := range room.Spectators {
		if spectatorID == userID {
			return errors.New("用戶已經是觀眾")
		}
	}

	room.Spectators = append(room.Spectators, userID)
	err = s.updateRoom(room)
	if err != nil {
		return err
	}

	s.wsManager.BroadcastSystemMessage(roomID, "新觀眾加入了房間")
	return nil
}

func (s *RoomService) RemoveSpectator(roomID, userID uint) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	for i, spectatorID := range room.Spectators {
		if spectatorID == userID {
			room.Spectators = append(room.Spectators[:i], room.Spectators[i+1:]...)
			err = s.updateRoom(room)
			if err != nil {
				return err
			}
			s.wsManager.BroadcastSystemMessage(roomID, "一位觀眾離開了房間")
			return nil
		}
	}

	return errors.New("用戶不是觀眾")
}

func (s *RoomService) GetDebateMessages(roomID uint) ([]models.DebateMessage, error) {
	room, err := s.roomRepo.FindByID(roomID)
	if err != nil {
		return nil, err
	}
	return room.Messages, nil
}

func (s *RoomService) AddMessage(roomID uint, userID uint, content string) error {
	room, err := s.roomRepo.FindByID(roomID)
	if err != nil {
		return err
	}

	message := models.DebateMessage{
		RoomID:    roomID,
		UserID:    userID,
		Content:   content,
		Timestamp: time.Now(),
	}

	room.Messages = append(room.Messages, message)
	return s.roomRepo.Update(room)
}

func (s *RoomService) convertModelToRoom(model *models.Room) *Room {
	return &Room{
		ID:              model.ID,
		Name:            model.Name,
		Description:     model.Description,
		Status:          string(model.Status),
		ProponentID:     model.ProponentID,
		OpponentID:      model.OpponentID,
		CurrentSpeaker:  model.CurrentSpeaker,
		CurrentRound:    model.CurrentRound,
		TotalRounds:     model.TotalRounds,
		RoundDuration:   model.RoundDuration,
		StartTime:       model.StartTime,
		EndTime:         model.EndTime,
		CurrentRoundEnd: model.CurrentRoundEnd,
		Spectators:      model.Spectators,
	}
}

func (s *RoomService) updateRoom(room *Room) error {
	model := &models.Room{
		Name:            room.Name,
		Description:     room.Description,
		Status:          models.RoomStatus(room.Status),
		ProponentID:     room.ProponentID,
		OpponentID:      room.OpponentID,
		CurrentSpeaker:  room.CurrentSpeaker,
		CurrentRound:    room.CurrentRound,
		TotalRounds:     room.TotalRounds,
		RoundDuration:   room.RoundDuration,
		StartTime:       room.StartTime,
		EndTime:         room.EndTime,
		CurrentRoundEnd: room.CurrentRoundEnd,
		Spectators:      room.Spectators,
	}
	model.ID = room.ID
	return s.roomRepo.Update(model)
}
