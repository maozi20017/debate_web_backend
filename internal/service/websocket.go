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
	Conn   *websocket.Conn
	UserID uint
	RoomID uint
	Role   string
}

// Message 定義了 WebSocket 消息的結構
type Message struct {
	Type      string      `json:"type"`
	Content   string      `json:"content"`
	UserID    uint        `json:"user_id"`
	RoomID    uint        `json:"room_id"`
	Role      string      `json:"role"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// WebSocketManager 管理所有 WebSocket 連接和消息廣播
type WebSocketManager struct {
	rooms       map[uint]map[*Client]bool
	messageRepo repository.DebateMessageRepository
	mu          sync.RWMutex
}

// NewWebSocketManager 創建一個新的 WebSocketManager 實例
func NewWebSocketManager(messageRepo repository.DebateMessageRepository) *WebSocketManager {
	return &WebSocketManager{
		rooms:       make(map[uint]map[*Client]bool),
		messageRepo: messageRepo,
	}
}

// HandleClient 處理單個客戶端的連接
func (manager *WebSocketManager) HandleClient(client *Client) {
	manager.addClientToRoom(client)
	defer manager.removeClientFromRoom(client)

	for {
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		manager.ProcessMessage(client, msg)
	}
}

// ProcessMessage 處理來自客戶端的消息
func (manager *WebSocketManager) ProcessMessage(client *Client, msg []byte) {
	var message Message
	if err := json.Unmarshal(msg, &message); err != nil {
		log.Printf("error parsing message: %v", err)
		return
	}

	message.UserID = client.UserID
	message.RoomID = client.RoomID
	message.Role = client.Role
	message.Timestamp = time.Now()

	// 保存消息到數據庫
	dbMessage := &models.DebateMessage{
		RoomID:    message.RoomID,
		UserID:    message.UserID,
		Content:   message.Content,
		Timestamp: message.Timestamp,
	}
	if err := manager.messageRepo.Create(dbMessage); err != nil {
		log.Printf("Error saving message to database: %v", err)
	}

	// 廣播消息給房間內的其他客戶端
	manager.BroadcastToRoom(message.RoomID, message)
}

// BroadcastToRoom 向特定房間的所有客戶端廣播消息
func (manager *WebSocketManager) BroadcastToRoom(roomID uint, message Message) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	clients, ok := manager.rooms[roomID]
	if !ok {
		return
	}

	for client := range clients {
		err := client.Conn.WriteJSON(message)
		if err != nil {
			log.Printf("error broadcasting message: %v", err)
			client.Conn.Close()
			delete(clients, client)
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

// addClientToRoom 將客戶端添加到指定房間
func (manager *WebSocketManager) addClientToRoom(client *Client) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if _, ok := manager.rooms[client.RoomID]; !ok {
		manager.rooms[client.RoomID] = make(map[*Client]bool)
	}
	manager.rooms[client.RoomID][client] = true
}

// removeClientFromRoom 將客戶端從指定房間移除
func (manager *WebSocketManager) removeClientFromRoom(client *Client) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if clients, ok := manager.rooms[client.RoomID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(manager.rooms, client.RoomID)
		}
	}
	client.Conn.Close()
}
