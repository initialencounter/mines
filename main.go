package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"main/database"
	"main/fiberHandle"
	"main/utils"
	"net/http"
	"strconv"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/golang-jwt/jwt/v5"
)

// 嵌入 mines-client/dist 目录中的所有文件
//
//go:embed mines-client/dist/*
var distFiles embed.FS

// 嵌入 mines-client/src/assets 目录中的所有文件
//
//go:embed mines-client/src/assets/*
var assetFiles embed.FS

var pool = utils.NewWebSocketPool()
var nameCache = utils.NewNameCache()
var scoreBoard = newScoreBoard()
var codeCache = utils.NewCodeCache()
var playerStates = utils.NewPlayerStateManager()

func main() {
	var config = getConfig()

	// 数据库连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", config.Database.User, config.Database.Password, config.Database.Host, config.Database.Port, "mines")
	fmt.Println(dsn)
	handler, err := database.NewDBHandler(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Close()
	err = handler.Db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	handler.CreateTable()

	// 扫雷地图初始化（带区域生成）
	zones := generateZones(config.Mine.Width, config.Mine.Height)
	fmt.Println(zones)
	propCounts := buildPropCounts(config.Props)
	var m = newMinefield(config.Mine.Mines, config.Mine.Width, config.Mine.Height, zones, propCounts)
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

	// HTTP API 路由
	app.Post("/register", func(c *fiber.Ctx) error { return fiberHandle.Register(handler, c) })
	app.Post("/login", func(c *fiber.Ctx) error { return fiberHandle.Login(handler, c) })
	app.Post("/verify", func(c *fiber.Ctx) error { return fiberHandle.VerifyCode(handler, c, config.Smtp, codeCache) })
	app.Post("/reset", func(c *fiber.Ctx) error { return fiberHandle.ResetPassword(handler, c, codeCache) })
	app.Post("/getMinefield", func(c *fiber.Ctx) error {
		return c.JSON(m.openMinefield())
	})

	app.Post("/getRank", func(c *fiber.Ctx) error {
		data, _ := handler.GetMedalRank()
		jsonData, _ := json.Marshal(data)
		return c.Send(jsonData)
	})

	app.Post("/newGame", func(c *fiber.Ctx) error {
		clearScoreBoard(scoreBoard, handler, nameCache)
		playerStates.ClearAll()
		zones := generateZones(config.Mine.Width, config.Mine.Height)
		propCounts := buildPropCounts(config.Props)
		m = newMinefield(config.Mine.Mines, config.Mine.Width, config.Mine.Height, zones, propCounts)
		return c.JSON(fiber.Map{"result": "ok"})
	})

	// WebSocket 升级中间件
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
	}, websocket.New(handleWebSocket(handler, m, &config)))

	log.Fatal(app.Listen(fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)))
}

// handleWebSocket 返回 WebSocket 连接的处理函数
func handleWebSocket(handler *database.DBHandler, m *Minefield, config *Config) func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		// Token 验证
		tokenString := c.Query("token")
		if tokenString == "" {
			c.WriteMessage(websocket.TextMessage, []byte("missing token"))
			c.Close()
			return
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid signing method")
			}
			return []byte("secret"), nil
		})

		if err != nil || !token.Valid {
			c.WriteMessage(websocket.TextMessage, []byte("invalid token"))
			c.Close()
			return
		}
		userId := c.Params("id")
		id, err := strconv.Atoi(userId)
		if err != nil {
			c.WriteMessage(websocket.TextMessage, []byte("invalid id"))
		}

		// 连接注册
		pool.Set(id, c)
		log.Println("New WebSocket connection added")

		// 获取用户名并缓存
		userName, err := handler.GetName(id)
		if err != nil {
			log.Println("Failed to get username:", err)
			return
		}
		nameCache.Set(id, userName)

		// 首个玩家连接时，布雷并翻开 4 个零值格子
		var initChangeCell ChangeCell
		if m.First {
			initChangeCell = m.initAndOpenFirstCells(4)
		}

		// 发送初始化消息给新连接
		initMinefield := m.openMinefield()
		if initBytes, err := json.Marshal(InitMessage{
			MessageType:    "init",
			Minefield:      initMinefield,
			ScoreBoard:     scoreBoard.Board,
			StartTimeStamp: m.StartTimeStamp,
			SafeCells:      m.RemainCells(),
			UserName:       userName,
		}); err == nil {
			pool.SendToPlayer(id, initBytes)
		}

		// 广播新玩家加入
		if joinBytes, err := json.Marshal(Response{NewPlayer: true, UserName: userName}); err == nil {
			pool.BroadcastExcept(id, joinBytes)
		}

		// 广播初始翻开的格子
		if len(initChangeCell.Cell) > 0 {
			if jsonData, err := json.Marshal(Response{
				ChangeCell:     initChangeCell,
				TimeStamp:      m.StartTimeStamp,
				StartTimeStamp: m.StartTimeStamp,
			}); err == nil {
				pool.BroadcastMessage(jsonData)
			}
		}

		// --- 消息处理主循环 ---
		var (
			msg       []byte
			newPlayer = false
		)
		log.Println(userId)
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
			if newPlayer {
				userName, err := handler.GetName(id)
				if err != nil {
					break
				}
				nameCache.Set(id, userName)
			}

			ps := playerStates.Get(id)

			// 根据 ActionType 路由到对应 handler
			var response Response
			switch message.ActionType {
			case "hint":
				response = handleHint(m, message, id, ps)
			case "useDetector":
				response = handleUseDetector(m, message, id, ps, config)
			case "useXJBD":
				response = handleUseXJBD(m, message, id, ps, config)
			default:
				response = handleNormalAction(m, message, id, ps, config, handler, newPlayer)
			}

			// 根据 ActionType 分发响应
			if message.ActionType == "hint" {
				if jsonData, err := json.Marshal(response); err == nil {
					pool.SendToPlayer(id, jsonData)
				}
			} else if message.ActionType == "useDetector" {
				if response.PropEffect != nil {
					effectResp := Response{
						MessageType: "propEffect",
						PropEffect:  response.PropEffect,
					}
					if effectBytes, err := json.Marshal(effectResp); err == nil {
						pool.BroadcastExcept(id, effectBytes)
					}
				}
				if jsonData, err := json.Marshal(response); err == nil {
					pool.SendToPlayer(id, jsonData)
				}
			} else if message.ActionType == "useXJBD" {
				scoreBoard.addScore(response.UserName, response.EarnScore)
				response.ScoreBoard = scoreBoard.Board
				jsonData, err := json.Marshal(response)
				if err != nil {
					fmt.Println(err)
				}
				pool.BroadcastMessage(jsonData)
				if response.PropBarUpdate != nil {
					personalResp := Response{
						MessageType:   "personal",
						PropBarUpdate: response.PropBarUpdate,
					}
					if personalBytes, err := json.Marshal(personalResp); err == nil {
						pool.SendToPlayer(id, personalBytes)
					}
				}
			} else {
				jsonData, err := json.Marshal(response)
				if err != nil {
					fmt.Println(err)
				}
				pool.BroadcastMessage(jsonData)
				if response.PropBarUpdate != nil || response.PropDrop != nil || response.ShieldProtect != nil {
					personalResp := Response{
						MessageType:   "personal",
						PropBarUpdate: response.PropBarUpdate,
						PropDrop:      response.PropDrop,
						ShieldProtect: response.ShieldProtect,
					}
					if personalBytes, err := json.Marshal(personalResp); err == nil {
						pool.SendToPlayer(id, personalBytes)
					}
				}
			}

			if response.ChangeCell.Result.IsWin {
				clearScoreBoard(scoreBoard, handler, nameCache)
				playerStates.ClearAll()
			}
			newPlayer = false
		}

		// 连接断开清理
		pool.Delete(id)
		playerStates.Delete(id)
		log.Println("WebSocket connection closed", userId)
		exitResp := Response{PlayerQuit: true, UserName: userId}
		if jsonData, err := json.Marshal(exitResp); err == nil {
			pool.BroadcastMessage(jsonData)
		}
		c.Close()
	}
}
