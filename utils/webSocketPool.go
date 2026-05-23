package utils

import (
	"fmt"
	"github.com/gofiber/contrib/websocket"
	"sync"
)

type WebSocketPool struct {
	mu    sync.RWMutex
	store map[int]*websocket.Conn
}

func NewWebSocketPool() *WebSocketPool {
	return &WebSocketPool{
		store: make(map[int]*websocket.Conn),
	}
}

func (c *WebSocketPool) Set(id int, conn *websocket.Conn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[id] = conn
}

func (c *WebSocketPool) Get(id int) (*websocket.Conn, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	conn, ok := c.store[id]
	return conn, ok
}

func (c *WebSocketPool) Delete(id int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, id)
}

func (c *WebSocketPool) SendToPlayer(id int, message []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	conn, ok := c.store[id]
	if !ok {
		return fmt.Errorf("player %d not connected", id)
	}
	return conn.WriteMessage(websocket.TextMessage, message)
}

func (c *WebSocketPool) BroadcastMessage(message []byte) {
	c.mu.RLock()
	var failedIDs []int
	for id, conn := range c.store {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			fmt.Printf("Error sending message to connection %d: %v\n", id, err)
			failedIDs = append(failedIDs, id)
		}
	}
	c.mu.RUnlock()
	if len(failedIDs) > 0 {
		c.mu.Lock()
		for _, id := range failedIDs {
			delete(c.store, id)
		}
		c.mu.Unlock()
	}
}

// BroadcastExcept sends a message to all players except the specified one
func (c *WebSocketPool) BroadcastExcept(exceptID int, message []byte) {
	c.mu.RLock()
	var failedIDs []int
	for id, conn := range c.store {
		if id == exceptID {
			continue
		}
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			fmt.Printf("Error sending message to connection %d: %v\n", id, err)
			failedIDs = append(failedIDs, id)
		}
	}
	c.mu.RUnlock()
	if len(failedIDs) > 0 {
		c.mu.Lock()
		for _, id := range failedIDs {
			delete(c.store, id)
		}
		c.mu.Unlock()
	}
}
