package service

import (
	"debate_web/internal/models"
	"debate_web/internal/repository"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client 表示一個 WebSocket 客戶端連接
type Client struct {
	Conn   *websocket.Conn // WebSocket 連接
	UserID uint            // 用戶 ID
	RoomID uint            // 房間 ID
	Role   string          // 用戶角色
}

// Message 定義了 WebSocket 消息的結構
type Message struct {
	Type      string      `json:"type"`      // 消息類型，例如 "chat", "system", "join", "leave" 等
	Content   string      `json:"content"`   // 消息內容
	UserID    uint        `json:"user_id"`   // 發送消息的用戶 ID
	RoomID    uint        `json:"room_id"`   // 消息所屬的房間 ID
	Role      string      `json:"role"`      // 發送消息的用戶角色
	Data      interface{} `json:"data"`      // 可選的附加數據，用於特定類型的消息
	Timestamp time.Time   `json:"timestamp"` // 消息發送時間戳
}

// WebSocketManager 管理所有 WebSocket 連接和消息廣播
type WebSocketManager struct {
	clients     map[*Client]bool                   // 存儲所有活躍的客戶端連接
	broadcast   chan Message                       // 用於廣播消息的通道
	register    chan *Client                       // 用於註冊新客戶端的通道
	unregister  chan *Client                       // 用於註銷客戶端的通道
	mutex       sync.Mutex                         // 用於保護對 clients map 的併發訪問
	messageRepo repository.DebateMessageRepository // 用於持久化消息的 repository
}

// NewWebSocketManager 創建一個新的 WebSocketManager 實例
func NewWebSocketManager(messageRepo repository.DebateMessageRepository) *WebSocketManager {
	return &WebSocketManager{
		clients:     make(map[*Client]bool),
		broadcast:   make(chan Message),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		messageRepo: messageRepo,
	}
}

// Run 啟動 WebSocket 管理器，處理客戶端的註冊、註銷和消息廣播
func (manager *WebSocketManager) Run() {
	for {
		select {
		case client := <-manager.register:
			// 註冊新客戶端
			manager.mutex.Lock()
			manager.clients[client] = true
			manager.mutex.Unlock()
		case client := <-manager.unregister:
			// 註銷客戶端
			if _, ok := manager.clients[client]; ok {
				manager.mutex.Lock()
				delete(manager.clients, client)
				manager.mutex.Unlock()
				client.Conn.Close()
			}
		case message := <-manager.broadcast:
			// 廣播消息給所有相關的客戶端
			for client := range manager.clients {
				if client.RoomID == message.RoomID {
					err := client.Conn.WriteJSON(message)
					if err != nil {
						log.Printf("error: %v", err)
						client.Conn.Close()
						manager.mutex.Lock()
						delete(manager.clients, client)
						manager.mutex.Unlock()
					}
				}
			}
		}
	}
}

// RegisterClient 註冊一個新的客戶端
func (manager *WebSocketManager) RegisterClient(client *Client) {
	manager.register <- client
}

// UnregisterClient 註銷一個客戶端
func (manager *WebSocketManager) UnregisterClient(client *Client) {
	manager.unregister <- client
}

// BroadcastMessage 廣播消息到所有相關的客戶端
func (manager *WebSocketManager) BroadcastMessage(message Message) {
	manager.broadcast <- message
}

// HandleMessages 處理來自客戶端的消息
func (manager *WebSocketManager) HandleMessages(client *Client) {
	defer func() {
		manager.UnregisterClient(client)
		client.Conn.Close()
	}()

	for {
		// 讀取客戶端發送的消息
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// 解析消息
		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Printf("error: %v", err)
			continue
		}

		// 保存消息到數據庫
		dbMessage := &models.DebateMessage{
			RoomID:    client.RoomID,
			UserID:    client.UserID,
			Content:   message.Content,
			Timestamp: time.Now(),
		}
		if err := manager.messageRepo.Create(dbMessage); err != nil {
			log.Printf("Error saving message to database: %v", err)
		}

		// 廣播消息給其他客戶端
		manager.BroadcastMessage(message)
	}
}

// BroadcastToRoom 向特定房間的所有客戶端廣播消息
func (manager *WebSocketManager) BroadcastToRoom(roomID uint, message Message) {
	for client := range manager.clients {
		if client.RoomID == roomID {
			err := client.Conn.WriteJSON(message)
			if err != nil {
				log.Printf("Error broadcasting to client: %v", err)
				manager.UnregisterClient(client)
			}
		}
	}
}

// BroadcastSystemMessage 向特定房間廣播系統消息
func (manager *WebSocketManager) BroadcastSystemMessage(roomID uint, content string) {
	message := Message{
		Type:      "system",
		Content:   content,
		RoomID:    roomID,
		Timestamp: time.Now(),
	}
	manager.BroadcastToRoom(roomID, message)
}
