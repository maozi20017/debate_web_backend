package service

import (
	"debate_web/internal/repository"
	"debate_web/internal/repository/models"
	"errors"
	"fmt"
)

type RoomService struct {
	repo      repository.RoomRepository
	wsService *WebSocketService
}

func NewRoomService(repo repository.RoomRepository, ws *WebSocketService) *RoomService {
	return &RoomService{
		repo:      repo,
		wsService: ws,
	}
}

func (s *RoomService) CreateRoom(room *models.Room) error {
	room.Status = models.RoomStatusWaiting
	return s.repo.Create(room)
}

func (s *RoomService) GetRoom(id uint) (*models.Room, error) {
	room, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return room, nil
}

func (s *RoomService) JoinRoom(roomID uint, userID uint, role string) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	if room.Status != models.RoomStatusWaiting {
		return errors.New("房間狀態不允許加入")
	}

	// 檢查角色分配
	switch role {
	case "proponent":
		if room.ProponentID != 0 {
			return errors.New("正方位置已被占用")
		}
		room.ProponentID = userID
	case "opponent":
		if room.OpponentID != 0 {
			return errors.New("反方位置已被占用")
		}
		room.OpponentID = userID
	default:
		return errors.New("無效的角色")
	}

	// 如果雙方都到齊，更新狀態為準備就緒
	if room.ProponentID != 0 && room.OpponentID != 0 {
		room.Status = models.RoomStatusReady
	}

	if err := s.repo.Update(room); err != nil {
		return err
	}

	// 透過 WebSocket 發送系統消息
	s.wsService.BroadcastSystemMessage(roomID, fmt.Sprintf("用戶 %d 以 %s 身份加入房間", userID, role))

	return nil
}

// LeaveRoom 離開房間
func (s *RoomService) LeaveRoom(roomID, userID uint) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return errors.New("房間不存在")
	}

	// 檢查用戶是否在房間中
	if room.ProponentID != userID && room.OpponentID != userID {
		return errors.New("用戶不在此房間中")
	}

	// 檢查房間狀態
	if room.Status == models.RoomStatusOngoing {
		return errors.New("辯論進行中，無法離開")
	}

	// 更新房間狀態
	if room.ProponentID == userID {
		room.ProponentID = 0
	} else if room.OpponentID == userID {
		room.OpponentID = 0
	}

	// 如果其中一方離開房間，將更改狀態
	if room.ProponentID == 0 || room.OpponentID == 0 {
		//如果還沒開始就轉回等待中，如果開始了就改成已結束
		if room.Status == models.RoomStatusReady {
			room.Status = models.RoomStatusWaiting
		} else if room.Status == models.RoomStatusOngoing {
			room.Status = models.RoomStatusFinished
		}
	}

	// 保存更改
	if err := s.repo.Update(room); err != nil {
		return err
	}

	// 發送系統消息
	s.wsService.BroadcastSystemMessage(roomID, "用戶離開了房間")

	return nil
}

// ListRooms 獲取房間列表
func (s *RoomService) ListRooms() ([]models.Room, error) {
	return s.repo.FindAll()
}

// checkUserInRoom 檢查用戶是否在房間中並返回其角色
func (h *RoomService) CheckUserInRoom(roomID, userID uint) (string, error) {
	room, err := h.GetRoom(roomID)
	if err != nil {
		return "", err
	}

	switch userID {
	case room.ProponentID:
		return "proponent", nil
	case room.OpponentID:
		return "opponent", nil
	default:
		for _, spectatorID := range room.Spectators {
			if spectatorID == userID {
				return "spectator", nil
			}
		}
		return "", errors.New("用戶不在此房間中")
	}
}
