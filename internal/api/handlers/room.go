package handlers

import (
	"debate_web/internal/models"
	"debate_web/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RoomHandler struct {
	roomService *service.RoomService
}

func NewRoomHandler(roomService *service.RoomService) *RoomHandler {
	return &RoomHandler{roomService: roomService}
}

func (h *RoomHandler) CreateRoom(c *gin.Context) {
	var room models.Room
	if err := c.ShouldBindJSON(&room); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.roomService.CreateRoom(&room); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, room)
}

func (h *RoomHandler) JoinRoom(c *gin.Context) {
	roomID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint("userID")
	role := c.PostForm("role")

	err := h.roomService.JoinRoom(uint(roomID), userID, role)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "成功加入房間"})
}

// GetRoom 獲取特定房間資訊
func (h *RoomHandler) GetRoom(c *gin.Context) {
	// 從路徑參數獲取房間ID
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "無效的房間ID",
		})
		return
	}

	room, err := h.roomService.GetRoom(uint(roomID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "找不到該房間",
		})
		return
	}

	// 根據用戶角色返回適當的資訊
	response := gin.H{
		"id":           room.ID,
		"name":         room.Name,
		"status":       room.Status,
		"created_at":   room.CreatedAt,
		"proponent_id": room.ProponentID,
		"opponent_id":  room.OpponentID,
		"spectators":   room.Spectators,
	}

	c.JSON(http.StatusOK, response)
}

// LeaveRoom 離開房間
func (h *RoomHandler) LeaveRoom(c *gin.Context) {
	// 從路徑參數獲取房間ID
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "無效的房間ID",
		})
		return
	}

	// 獲取當前用戶ID
	userID := c.GetUint("userID")

	// 調用服務層的離開房間方法
	err = h.roomService.LeaveRoom(uint(roomID), userID)
	if err != nil {
		// 根據錯誤類型返回適當的狀態碼和訊息
		switch err.Error() {
		case "房間不存在":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "用戶不在此房間中":
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case "辯論進行中，無法離開":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "離開房間失敗"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "成功離開房間",
	})
}

// ListRooms 獲取房間列表
func (h *RoomHandler) ListRooms(c *gin.Context) {
	rooms, err := h.roomService.ListRooms()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "獲取房間列表失敗",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rooms": rooms,
	})
}
