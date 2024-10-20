package service

import (
	"debate_web/internal/models"
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn   *websocket.Conn
	UserID uint
	RoomID uint
	Role   string
}

type WebSocketManager struct {
	clients    map[uint]map[*websocket.Conn]*Client
	clientsMux sync.RWMutex
}

func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients: make(map[uint]map[*websocket.Conn]*Client),
	}
}

func (manager *WebSocketManager) HandleClient(client *Client) {
	manager.addClient(client)
	defer manager.removeClient(client)

	for {
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		manager.processMessage(client, msg)
	}
}

func (manager *WebSocketManager) addClient(client *Client) {
	manager.clientsMux.Lock()
	defer manager.clientsMux.Unlock()

	if _, ok := manager.clients[client.RoomID]; !ok {
		manager.clients[client.RoomID] = make(map[*websocket.Conn]*Client)
	}
	manager.clients[client.RoomID][client.Conn] = client
}

func (manager *WebSocketManager) removeClient(client *Client) {
	manager.clientsMux.Lock()
	defer manager.clientsMux.Unlock()

	if _, ok := manager.clients[client.RoomID]; ok {
		delete(manager.clients[client.RoomID], client.Conn)
		if len(manager.clients[client.RoomID]) == 0 {
			delete(manager.clients, client.RoomID)
		}
	}
	client.Conn.Close()
}

func (manager *WebSocketManager) processMessage(client *Client, msg []byte) {
	var message models.Message
	if err := json.Unmarshal(msg, &message); err != nil {
		log.Printf("error parsing message: %v", err)
		return
	}

	message.UserID = client.UserID
	message.RoomID = client.RoomID
	message.Role = client.Role

	manager.BroadcastToRoom(message.RoomID, message.ToWebSocketMessage())
}

func (manager *WebSocketManager) BroadcastToRoom(roomID uint, message map[string]interface{}) {
	manager.clientsMux.RLock()
	defer manager.clientsMux.RUnlock()

	if clients, ok := manager.clients[roomID]; ok {
		for _, client := range clients {
			err := client.Conn.WriteJSON(message)
			if err != nil {
				log.Printf("error broadcasting message: %v", err)
				client.Conn.Close()
				delete(clients, client.Conn)
			}
		}
	}
}

func (manager *WebSocketManager) BroadcastSystemMessage(roomID uint, content string) {
	message := models.NewSystemMessage(roomID, content)
	manager.BroadcastToRoom(roomID, message.ToWebSocketMessage())
}

func (manager *WebSocketManager) DisconnectUser(roomID, userID uint) {
	manager.clientsMux.Lock()
	defer manager.clientsMux.Unlock()

	if roomClients, ok := manager.clients[roomID]; ok {
		for conn, client := range roomClients {
			if client.UserID == userID {
				err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "User left the room"))
				if err != nil {
					log.Printf("Error sending close message: %v", err)
				}
				conn.Close()
				delete(roomClients, conn)
				log.Printf("User %d disconnected from room %d", userID, roomID)
			}
		}
		if len(roomClients) == 0 {
			delete(manager.clients, roomID)
		}
	}
}
