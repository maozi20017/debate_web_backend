// service/websocket.go
package service

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"debate_web/internal/models"
)

// Client 代表一個 WebSocket 客戶端連接
type Client struct {
	Conn     *websocket.Conn
	UserID   uint
	RoomID   uint
	Role     string
	SendChan chan *models.Message // 消息發送通道
}

// WebSocketService 管理 WebSocket 連接和消息
type WebSocketService struct {
	// 使用兩層 map 來管理房間和客戶端
	// 第一層 key 是房間 ID，第二層 key 是客戶端指針
	clients    map[uint]map[*Client]bool
	clientsMux sync.RWMutex
}

// NewWebSocketService 創建新的 WebSocket 服務實例
func NewWebSocketService() *WebSocketService {
	return &WebSocketService{
		clients: make(map[uint]map[*Client]bool),
	}
}

// HandleConnection 處理新的 WebSocket 連接
func (s *WebSocketService) HandleConnection(conn *websocket.Conn, roomID, userID uint, role string) {
	client := &Client{
		Conn:     conn,
		UserID:   userID,
		RoomID:   roomID,
		Role:     role,
		SendChan: make(chan *models.Message, 256), // 緩衝通道
	}

	// 將客戶端加入管理
	s.addClient(client)

	// 清理函數
	defer func() {
		s.removeClient(client)
		conn.Close()
		close(client.SendChan)
	}()

	// 啟動消息處理的 goroutines
	go s.writePump(client)
	s.readPump(client)
}

// readPump 處理從客戶端讀取消息
func (s *WebSocketService) readPump(client *Client) {
	// 設置讀取限制
	client.Conn.SetReadLimit(4096) // 4KB
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// 解析消息
		var msg models.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("error parsing message: %v", err)
			continue
		}

		// 設置消息屬性
		msg.UserID = client.UserID
		msg.RoomID = client.RoomID
		msg.Role = client.Role

		// 廣播消息
		s.BroadcastToRoom(client.RoomID, &msg)
	}
}

// writePump 處理向客戶端發送消息
func (s *WebSocketService) writePump(client *Client) {
	ticker := time.NewTicker(54 * time.Second) // 心跳檢查
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-client.SendChan:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道已關閉
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			// 將消息編碼為 JSON
			messageBytes, err := json.Marshal(message)
			if err != nil {
				log.Printf("error encoding message: %v", err)
				continue
			}

			if _, err := w.Write(messageBytes); err != nil {
				return
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// BroadcastToRoom 向房間內所有客戶端廣播消息
func (s *WebSocketService) BroadcastToRoom(roomID uint, message *models.Message) {
	s.clientsMux.RLock()
	clients := s.clients[roomID]
	s.clientsMux.RUnlock()

	for client := range clients {
		select {
		case client.SendChan <- message:
			// 消息成功加入發送隊列
		default:
			// 如果客戶端的發送隊列已滿，關閉連接
			s.removeClient(client)
			client.Conn.Close()
		}
	}
}

// BroadcastSystemMessage 發送系統消息到指定房間
func (s *WebSocketService) BroadcastSystemMessage(roomID uint, content string) {
	message := &models.Message{
		Type:    "system",
		Content: content,
		RoomID:  roomID,
	}
	s.BroadcastToRoom(roomID, message)
}

// addClient 添加新的客戶端連接
func (s *WebSocketService) addClient(client *Client) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	if s.clients[client.RoomID] == nil {
		s.clients[client.RoomID] = make(map[*Client]bool)
	}
	s.clients[client.RoomID][client] = true

	// 發送歡迎消息
	s.BroadcastSystemMessage(client.RoomID,
		"新用戶加入: "+client.Role)
}

// removeClient 移除客戶端連接
func (s *WebSocketService) removeClient(client *Client) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	if clients, ok := s.clients[client.RoomID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(s.clients, client.RoomID)
			}
		}
	}
}

// DisconnectUser 強制斷開用戶的連接
func (s *WebSocketService) DisconnectUser(roomID, userID uint) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	if clients, ok := s.clients[roomID]; ok {
		for client := range clients {
			if client.UserID == userID {
				client.Conn.Close()
				delete(clients, client)
			}
		}
		if len(clients) == 0 {
			delete(s.clients, roomID)
		}
	}
}

// GetRoomClients 獲取房間內的客戶端數量
func (s *WebSocketService) GetRoomClients(roomID uint) int {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()

	if clients, ok := s.clients[roomID]; ok {
		return len(clients)
	}
	return 0
}
