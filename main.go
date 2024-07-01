package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/http"
	"sync"
)

// 嵌入 mines-client/dist 目录中的所有文件
//
//go:embed mines-client/dist/*
var distFiles embed.FS

// 嵌入 mines-client/src/assets 目录中的所有文件
//
//go:embed mines-client/src/assets/*
var assetFiles embed.FS

type WebSocketPool struct {
	connections map[*websocket.Conn]bool
	mu          sync.Mutex
}

var pool = WebSocketPool{
	connections: make(map[*websocket.Conn]bool),
}

const MINES = 99
const WIDTH = 30
const HEIGHT = 16

func main() {
	var m = newMinefield(MINES, WIDTH, HEIGHT)
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173",
		AllowHeaders: "Origin, Content-Type, Accept",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
	}))

	app.Use("/", filesystem.New(filesystem.Config{
		Root:       http.FS(distFiles),
		PathPrefix: "mines-client/dist",
	}))

	app.Use("/src/assets", filesystem.New(filesystem.Config{
		Root:       http.FS(assetFiles),
		PathPrefix: "mines-client/src/assets",
	}))
	//app.Static("/", "./mines-client/dist")
	//app.Static("/src/assets", "./mines-client/src/assets")
	// Login route
	app.Post("/login", login)
	app.Post("/getMinefield", func(c *fiber.Ctx) error {
		return c.JSON(m.openMinefield())
	})

	app.Post("/newGame", func(c *fiber.Ctx) error {
		stats := m.getStats(0)
		if stats.IsWin {
			m = newMinefield(MINES, WIDTH, HEIGHT)
			return c.JSON(fiber.Map{"result": "ok"})
		} else {
			return c.JSON(fiber.Map{"result": "fail"})
		}
	})

	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws/:id", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}, websocket.New(func(c *websocket.Conn) {
		// Extract token from query parameters
		tokenString := c.Query("token")
		if tokenString == "" {
			err := c.WriteMessage(websocket.TextMessage, []byte("missing token"))
			if err != nil {
				return
			}
			err = c.Close()
			if err != nil {
				return
			}
			return
		}
		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid signing method")
			}
			return []byte("secret"), nil
		})

		if err != nil || !token.Valid {
			err := c.WriteMessage(websocket.TextMessage, []byte("invalid token"))
			if err != nil {
				return
			}
			err = c.Close()
			if err != nil {
				return
			}
			return
		}

		// Add connection to the pool
		pool.mu.Lock()
		pool.connections[c] = true
		pool.mu.Unlock()
		log.Println("New WebSocket connection added")

		var (
			msg []byte
		)
		log.Println(c.Params("id"))
		for {
			if _, msg, err = c.ReadMessage(); err != nil {
				log.Println("read:", err)
				break
			}
			var message Request
			if err := json.Unmarshal(msg, &message); err != nil {
				log.Println("Unmarshal error:", err)
				continue
			}
			var response Response
			result := m.openCells(message.Id)
			response.ChangeCell = result
			response.TimeStamp = message.TimeStamp
			response.StartTimeStamp = m.StartTimeStamp

			jsonData, err := json.Marshal(response)
			if err != nil {
				fmt.Println(err)
			}
			pool.broadcastMessage(jsonData)
			//log.Println("send:", string(jsonData))
		}

		// Remove connection from the pool
		pool.mu.Lock()
		delete(pool.connections, c)
		pool.mu.Unlock()
		log.Println("WebSocket connection closed")
		err = c.Close()
		if err != nil {
			return
		}

	}))
	log.Fatal(app.Listen(":3000"))
}

func (p *WebSocketPool) broadcastMessage(message []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for conn := range p.connections {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Error sending message: %v", err)
			err := conn.Close()
			if err != nil {
				return
			}
			delete(p.connections, conn)
		}
	}
}
