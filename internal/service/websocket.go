package service

import (
	"debate_web/internal/repository/models"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client 代表一個 WebSocket 客戶端連接
type Client struct {
	Conn     *websocket.Conn      // WebSocket 連接
	UserID   uint                 // 用戶 ID
	RoomID   uint                 // 房間 ID
	Role     string               // 用戶角色 (proponent/opponent/spectator)
	SendChan chan *models.Message // 消息發送通道，用於異步傳送消息
}

// WebSocketService 管理所有的 WebSocket 連接和消息傳遞
type WebSocketService struct {
	clients    map[uint]map[*Client]bool // 兩層 map: roomID -> client -> bool
	clientsMux sync.RWMutex              // 用於保護 clients map 的讀寫鎖
}

// NewWebSocketService 創建並初始化新的 WebSocket 服務
func NewWebSocketService() *WebSocketService {
	return &WebSocketService{
		clients: make(map[uint]map[*Client]bool),
	}
}

// HandleConnection 處理新的 WebSocket 連接請求
// 參數: websocket 連接、房間ID、用戶ID、用戶角色
func (s *WebSocketService) HandleConnection(conn *websocket.Conn, roomID, userID uint, role string) {
	client := &Client{
		Conn:     conn,
		UserID:   userID,
		RoomID:   roomID,
		Role:     role,
		SendChan: make(chan *models.Message, 256), // 設置緩衝大小為 256 的消息通道
	}

	s.addClient(client)

	// 確保連接關閉時清理資源
	defer func() {
		s.removeClient(client)
		conn.Close()
		close(client.SendChan)
	}()

	// 啟動讀寫處理
	go s.writePump(client)
	s.readPump(client)
}

// readPump 持續監聽並處理從客戶端接收的消息
func (s *WebSocketService) readPump(client *Client) {
	client.Conn.SetReadLimit(4096) // 設置最大消息大小為 4KB
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket unexpected close error: %v", err)
			}
			break
		}

		// 解析接收到的消息
		var msg models.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("message parse error: %v", err)
			continue
		}

		// 設置消息的基本屬性
		msg.UserID = client.UserID
		msg.RoomID = client.RoomID
		msg.Role = client.Role

		// 廣播消息給房間內所有用戶
		s.BroadcastToRoom(client.RoomID, &msg)
	}
}

// writePump 處理向客戶端發送消息的邏輯
func (s *WebSocketService) writePump(client *Client) {
	// 設置心跳檢查計時器
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-client.SendChan:
			// 設置寫入超時
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 獲取寫入器並發送消息
			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			// JSON 編碼
			messageBytes, err := json.Marshal(message)
			if err != nil {
				log.Printf("message encoding error: %v", err)
				continue
			}

			if _, err := w.Write(messageBytes); err != nil {
				return
			}
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// 發送心跳包
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// BroadcastToRoom 向房間內的所有客戶端廣播消息
func (s *WebSocketService) BroadcastToRoom(roomID uint, message *models.Message) {
	s.clientsMux.RLock()
	clients := s.clients[roomID]
	s.clientsMux.RUnlock()

	for client := range clients {
		select {
		case client.SendChan <- message:
			// 消息成功加入發送隊列
		default:
			// 客戶端消息隊列已滿，關閉連接
			s.removeClient(client)
			client.Conn.Close()
		}
	}
}

// BroadcastSystemMessage 發送系統消息到指定房間
func (s *WebSocketService) BroadcastSystemMessage(roomID uint, content string) {
	msg := &models.Message{
		Type:    "system",
		Content: content,
		RoomID:  roomID,
	}
	s.BroadcastToRoom(roomID, msg)
}

// addClient 安全地添加新的客戶端連接
func (s *WebSocketService) addClient(client *Client) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	if s.clients[client.RoomID] == nil {
		s.clients[client.RoomID] = make(map[*Client]bool)
	}
	s.clients[client.RoomID][client] = true

	// 發送用戶加入通知
	s.BroadcastSystemMessage(client.RoomID,
		fmt.Sprintf("用戶 %d 加入房間", client.UserID))
}

// removeClient 安全地移除客戶端連接
func (s *WebSocketService) removeClient(client *Client) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	if clients, ok := s.clients[client.RoomID]; ok {
		delete(clients, client)
		// 如果房間空了，刪除房間
		if len(clients) == 0 {
			delete(s.clients, client.RoomID)
		}
	}
}

// GetRoomClients 獲取指定房間的在線客戶端數量
func (s *WebSocketService) GetRoomClients(roomID uint) int {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()

	return len(s.clients[roomID])
}
