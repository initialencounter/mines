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

func (c *WebSocketPool) BroadcastMessage(message []byte) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for id, conn := range c.store {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			fmt.Printf("Error sending message to connection %d: %v\n", id, err)
			err := conn.Close()
			if err != nil {
				return
			}
			delete(c.store, id)
		}
	}
}
