package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"debate_web/internal/service"
)

// RoomHandler 處理與辯論房間相關的請求
type RoomHandler struct {
	roomService *service.RoomService
}

// NewRoomHandler 創建一個新的 RoomHandler 實例
func NewRoomHandler(roomService *service.RoomService) *RoomHandler {
	return &RoomHandler{roomService: roomService}
}

// CreateRoom 處理創建新房間的請求
func (h *RoomHandler) CreateRoom(c *gin.Context) {
	var input struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		MaxDuration int    `json:"max_duration" binding:"required"`
		TotalRounds int    `json:"total_rounds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	room, err := h.roomService.CreateRoom(input.Name, input.Description, input.MaxDuration, input.TotalRounds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "創建房間失敗"})
		return
	}

	c.JSON(http.StatusCreated, room)
}

// GetRoom 處理獲取房間訊息的請求
func (h *RoomHandler) GetRoom(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不存在的房間ID"})
		return
	}

	room, err := h.roomService.GetRoom(uint(roomID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "房間不存在"})
		return
	}

	c.JSON(http.StatusOK, room)
}

// JoinRoom 處理加入房間的請求
func (h *RoomHandler) JoinRoom(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不存在的房間ID"})
		return
	}

	userID, _ := c.Get("userID")
	role := c.Query("role")

	err = h.roomService.JoinRoom(uint(roomID), userID.(uint), role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "成功加入房間"})
}

// LeaveRoom 處理離開房間的請求
func (h *RoomHandler) LeaveRoom(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不存在的房間ID"})
		return
	}

	userID, _ := c.Get("userID")

	err = h.roomService.LeaveRoom(uint(roomID), userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "成功離開房間"})
}

// StartDebate 處理開始辯論的請求
func (h *RoomHandler) StartDebate(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不存在的房間ID"})
		return
	}

	err = h.roomService.StartDebate(uint(roomID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "辯論開始"})
}

// EndDebate 處理結束辯論的請求
func (h *RoomHandler) EndDebate(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不存在的房間ID"})
		return
	}

	err = h.roomService.EndDebate(uint(roomID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "辯論結束"})
}

func (h *RoomHandler) GetDebateMessages(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的房間 ID"})
		return
	}

	messages, err := h.roomService.GetDebateMessages(uint(roomID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法搜尋辯論訊息"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func (h *RoomHandler) NextRound(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的房間 ID"})
		return
	}

	err = h.roomService.NextRound(uint(roomID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "進入下一回合"})
}

func (h *RoomHandler) GetRemainingTime(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的房間 ID"})
		return
	}

	room, err := h.roomService.GetRoom(uint(roomID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "房間不存在"})
		return
	}

	remainingTime := time.Until(room.CurrentRoundEnd)
	if remainingTime < 0 {
		remainingTime = 0
	}

	c.JSON(http.StatusOK, gin.H{"remaining_time": int(remainingTime.Seconds())})
}

func (h *RoomHandler) AddSpectator(c *gin.Context) {
	roomID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	userID, _ := c.Get("userID")

	err := h.roomService.JoinRoom(uint(roomID), userID.(uint), "spectator")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "成功加入為觀眾"})
}

func (h *RoomHandler) RemoveSpectator(c *gin.Context) {
	roomID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	userID, _ := c.Get("userID")

	err := h.roomService.LeaveRoom(uint(roomID), userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "成功離開觀眾席"})
}
