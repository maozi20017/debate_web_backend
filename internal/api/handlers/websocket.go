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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "升級WebSocket失敗"})
		return
	}
	defer conn.Close()

	// 解析房間 ID
	roomID, err := strconv.ParseUint(c.Query("room_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "錯誤的房間ID"})
		return
	}

	// 從上下文中獲取用戶 ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userIDUint := userID.(uint)

	// 確定用戶在房間中的角色
	role, err := h.determineUserRole(h.roomService, uint(roomID), userIDUint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法確定用戶角色"})
		return
	}

	if role == "unknown" {
		c.JSON(http.StatusForbidden, gin.H{"error": "用戶未加入此房間"})
		return
	}

	// 創建客戶端
	client := &service.Client{
		Conn:   conn,
		UserID: userIDUint,
		RoomID: uint(roomID),
		Role:   role,
	}

	// 處理客戶端連接
	h.wsManager.HandleClient(client)

	// 通知房間有新用戶加入
	h.wsManager.BroadcastSystemMessage(uint(roomID), "New user joined: "+role)
}

// determineUserRole 確定用戶在房間中的角色
func (h *WebSocketHandler) determineUserRole(roomService *service.RoomService, roomID, userID uint) (string, error) {
	room, err := roomService.GetRoom(roomID)
	if err != nil {
		return "", err
	}

	switch userID {
	case room.ProponentID:
		return "proponent", nil
	case room.OpponentID:
		return "opponent", nil
	default:
		// 檢查是否為觀眾
		for _, spectatorID := range room.Spectators {
			if spectatorID == userID {
				return "spectator", nil
			}
		}
		return "unknown", nil
	}
}
