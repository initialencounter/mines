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
	connections map[string]*websocket.Conn
	mu          sync.Mutex
}

var pool = WebSocketPool{
	connections: make(map[string]*websocket.Conn),
}

func main() {
	var config = getConfig()

	// 数据库连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", config.Database.User, config.Database.Password, config.Database.Host, config.Database.Port, "mines")
	fmt.Println(dsn)
	handler, err := NewDBHandler(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Close()
	// 确认连接有效
	err = handler.db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	handler.createTable()

	// 扫雷地图初始化
	var m = newMinefield(config.Mine.Mines, config.Mine.Width, config.Mine.Height)
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

	app.Post("/register", func(c *fiber.Ctx) error { return register(handler, c) })
	app.Post("/login", func(c *fiber.Ctx) error { return login(handler, c) })
	app.Post("/getMinefield", func(c *fiber.Ctx) error {
		return c.JSON(m.openMinefield())
	})

	app.Post("/newGame", func(c *fiber.Ctx) error {
		stats := m.getStats(0)
		if stats.IsWin {
			m = newMinefield(config.Mine.Mines, config.Mine.Width, config.Mine.Height)
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
		userId := c.Params("id")
		// Add connection to the pool
		pool.mu.Lock()
		pool.connections[userId] = c
		pool.mu.Unlock()
		log.Println("New WebSocket connection added")

		var (
			msg []byte
		)
		log.Println(c.Params("id"))
		var newPlayer = true
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
			response.NewPlayer = newPlayer
			response.UserId = userId
			response.ChangeCell = result
			response.TimeStamp = message.TimeStamp
			response.StartTimeStamp = m.StartTimeStamp

			jsonData, err := json.Marshal(response)
			if err != nil {
				fmt.Println(err)
			}
			pool.broadcastMessage(jsonData)
			newPlayer = false
		}

		// Remove connection from the pool
		pool.mu.Lock()
		delete(pool.connections, userId)
		pool.mu.Unlock()
		log.Println("WebSocket connection closed", userId)
		var response Response
		response.PlayerQuit = true
		response.UserId = userId
		jsonData, err := json.Marshal(response)
		if err != nil {
			fmt.Println(err)
		}
		pool.broadcastMessage(jsonData)
		err = c.Close()
		if err != nil {
			return
		}

	}))
	log.Fatal(app.Listen(fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)))
}

func (p *WebSocketPool) broadcastMessage(message []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for tag, conn := range p.connections {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Error sending message: %v", err)
			err := conn.Close()
			if err != nil {
				return
			}
			delete(p.connections, tag)
		}
	}
}
