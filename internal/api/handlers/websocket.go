package handlers

import (
	"debate_web/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketHandler 處理 WebSocket 連接
type WebSocketHandler struct {
	wsService   *service.WebSocketService
	roomService *service.RoomService
}

// 設定 WebSocket 升級器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// NewWebSocketHandler 創建新的 WebSocket 處理器
func NewWebSocketHandler(wsService *service.WebSocketService, roomService *service.RoomService) *WebSocketHandler {
	return &WebSocketHandler{
		wsService:   wsService,
		roomService: roomService,
	}
}

// HandleWebSocket 處理 WebSocket 連接請求
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// 從 URL 獲取房間 ID
	roomID, err := strconv.ParseUint(c.Query("room_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的房間ID"})
		return
	}

	// 從 context 獲取用戶 ID（經過身份驗證中間件設置）
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授權的訪問"})
		return
	}

	// 檢查用戶是否在房間中
	role, err := h.roomService.CheckUserInRoom(uint(roomID), userID.(uint))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// 升級 HTTP 連接為 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// 開始處理 WebSocket 連接
	h.wsService.HandleConnection(conn, uint(roomID), userID.(uint), role)
}
