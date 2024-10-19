package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"debate_web/internal/service"
)

// 定義 WebSocket 升級器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 注意：在生產環境中，應該檢查 origin
	},
}

// WebSocketHandler 處理 WebSocket 連接
type WebSocketHandler struct {
	wsManager   *service.WebSocketManager
	roomService *service.RoomService
}

// NewWebSocketHandler 創建一個新的 WebSocketHandler 實例
func NewWebSocketHandler(wsManager *service.WebSocketManager, roomService *service.RoomService) *WebSocketHandler {
	return &WebSocketHandler{
		wsManager:   wsManager,
		roomService: roomService,
	}
}

// HandleWebSocket 處理 WebSocket 連接請求
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// 升級 HTTP 連接為 WebSocket 連接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade to WebSocket"})
		return
	}

	// 解析房間 ID
	roomID, err := strconv.ParseUint(c.Query("room_id"), 10, 32)
	if err != nil {
		conn.Close()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	// 從上下文中獲取用戶 ID
	userID, exists := c.Get("userID")
	if !exists {
		conn.Close()
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userIDUint := userID.(uint)

	// 獲取房間信息
	room, err := h.roomService.GetRoom(uint(roomID))
	if err != nil {
		conn.Close()
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	// 確定用戶在房間中的角色
	role := h.determineUserRole(room, userIDUint)

	// 創建客戶端
	client := &service.Client{
		Conn:   conn,
		UserID: userIDUint,
		RoomID: uint(roomID),
		Role:   role,
	}

	// 處理客戶端連接
	go h.wsManager.HandleClient(client)

	// 通知房間有新用戶加入
	h.wsManager.BroadcastSystemMessage(uint(roomID), "New user joined: "+role)
}

// determineUserRole 確定用戶在房間中的角色
func (h *WebSocketHandler) determineUserRole(room *service.Room, userID uint) string {
	switch userID {
	case room.ProponentID:
		return "proponent"
	case room.OpponentID:
		return "opponent"
	default:
		return "spectator"
	}
}
