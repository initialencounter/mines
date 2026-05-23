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
	"time"

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
	// 确认连接有效
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
		id, err := strconv.Atoi(userId)
		if err != nil {
			err := c.WriteMessage(websocket.TextMessage, []byte("invalid id"))
			if err != nil {
				return
			}
		}
		// Add connection to the pool
		pool.Set(id, c)
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
			if newPlayer {
				userName, err := handler.GetName(id)
				if err != nil {
					break
				}
				nameCache.Set(id, userName)
			}

			playerState := playerStates.Get(id)

			// Route by ActionType
			var response Response
			switch message.ActionType {
			case "useDetector":
				response = handleUseDetector(&m, message, id, playerState, &config)
			case "useXJBD":
				response = handleUseXJBD(&m, message, id, playerState, &config)
			default:
				// Normal open/flag flow
				response = handleNormalAction(&m, message, id, playerState, &config, handler, newPlayer)
			}

			if message.ActionType == "useDetector" {
				// Detector: send mine positions only to the acting player
				// Broadcast propEffect notification to all others
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
				// XJBD: broadcast cell changes to ALL players (shared game state)
				scoreBoard.addScore(response.UserName, response.EarnScore)
				response.ScoreBoard = scoreBoard.Board

				jsonData, err := json.Marshal(response)
				if err != nil {
					fmt.Println(err)
				}
				pool.BroadcastMessage(jsonData)

				// Send personalized propBarUpdate to the actor
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
				// Normal action: broadcast to all
				jsonData, err := json.Marshal(response)
				if err != nil {
					fmt.Println(err)
				}
				pool.BroadcastMessage(jsonData)

				// Send personalized propBarUpdate to the acting player
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

		// Remove connection from the pool
		pool.Delete(id)
		playerStates.Delete(id)
		log.Println("WebSocket connection closed", userId)
		var response Response
		response.PlayerQuit = true
		response.UserName = userId
		jsonData, err := json.Marshal(response)
		if err != nil {
			fmt.Println(err)
		}
		pool.BroadcastMessage(jsonData)
		err = c.Close()
		if err != nil {
			return
		}

	}))
	log.Fatal(app.Listen(fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)))
}

func handleUseDetector(m *Minefield, message Request, playerID int, ps *utils.PlayerState, config *Config) Response {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !ps.HasProp(PropDetector) {
		return Response{}
	}

	ps.UseProp(PropDetector)
	detectorResult := m.useDetector(message.TargetCell)

	userName, _ := nameCache.GetName(playerID)

	return Response{
		UserName:       userName,
		MessageType:    "detectorResult",
		DetectorResult: &detectorResult,
		PropBarUpdate:  buildPropBarUpdate(ps),
		PropEffect: &PropEffectInfo{
			PropID:     PropDetector,
			UserName:   userName,
			TargetCell: message.TargetCell,
		},
		TimeStamp:      message.TimeStamp,
		StartTimeStamp: m.StartTimeStamp,
		ChangeCell:     ChangeCell{Cell: []Cell{}},
	}
}

func handleUseXJBD(m *Minefield, message Request, playerID int, ps *utils.PlayerState, config *Config) Response {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !ps.HasProp(PropXJBD) {
		return Response{}
	}

	ps.UseProp(PropXJBD)
	result := m.useXJBD(message.TargetCell)

	if m.IsWind {
		result = ChangeCell{
			Result: Result{
				IsWin:       true,
				IsBoom:      false,
				RemainCells: 0,
				Message:     "You Win!",
			},
			Cell: []Cell{},
		}
	}

	var timeStamp = message.TimeStamp
	if m.IsWind && m.EndTimeStamp == 0 {
		m.EndTimeStamp = timeStamp
	}
	if m.IsWind {
		timeStamp = m.EndTimeStamp
	}

	doubleScoreActive := ps.IsDoubleScoreActive()
	inZone := m.anyInDoubleScoreZone(message.Ids)
	earnScore := calculateScoreWithContext(message, result, doubleScoreActive, inZone)

	userName, _ := nameCache.GetName(playerID)

	return Response{
		UserName:       userName,
		ChangeCell:     result,
		TimeStamp:      timeStamp,
		StartTimeStamp: m.StartTimeStamp,
		EarnScore:      earnScore,
		PropBarUpdate:  buildPropBarUpdate(ps),
		PropEffect: &PropEffectInfo{
			PropID:     PropXJBD,
			UserName:   userName,
			TargetCell: message.TargetCell,
		},
	}
}

func handleNormalAction(m *Minefield, message Request, playerID int, ps *utils.PlayerState, config *Config, handler *database.DBHandler, newPlayer bool) Response {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check no-flag zone for flag actions
	if message.IsFlag {
		for _, id := range message.Ids {
			if m.isInNoFlagZone(id) {
				userName, _ := nameCache.GetName(playerID)
				return Response{
					NewPlayer:      newPlayer,
					UserName:       userName,
					ChangeCell:     ChangeCell{Cell: []Cell{}},
					TimeStamp:      message.TimeStamp,
					StartTimeStamp: m.StartTimeStamp,
					PropBarUpdate:  buildPropBarUpdate(ps),
				}
			}
		}
	}

	var result ChangeCell
	if message.IsFlag {
		result = m.doFlag(message.Ids[0])
	} else {
		result = m.openCells(message.Ids)
	}

	var timeStamp = message.TimeStamp
	if m.IsWind {
		result = ChangeCell{
			Result: Result{
				IsWin:       true,
				IsBoom:      false,
				RemainCells: 0,
				Message:     "You Win!",
			},
			Cell: []Cell{},
		}
		if m.EndTimeStamp == 0 {
			m.EndTimeStamp = timeStamp
		}
		timeStamp = m.EndTimeStamp
	}

	// Shield protection check for mine hit
	if result.Result.IsBoom && message.ActionType == "" {
		if ps.ConsumeShield() {
			// Shield absorbed the hit — revert the mine cell
			for i := range result.Cell {
				cid := result.Cell[i].Id
				if m.Cell[cid].IsMine {
					m.Cell[cid].IsOpen = false
					result.Cell[i].IsOpen = false
				}
			}
			result.Result.IsBoom = false
			result.Result.Message = "shield protected"

			shieldCount := ps.GetShieldCount()
			userName, _ := nameCache.GetName(playerID)

			return Response{
				NewPlayer:      newPlayer,
				UserName:       userName,
				ChangeCell:     result,
				TimeStamp:      timeStamp,
				StartTimeStamp: m.StartTimeStamp,
				EarnScore:      0,
				ShieldProtect: &ShieldProtect{
					ShieldCount: shieldCount,
					CellID:      message.Ids[0],
					WasMine:     true,
				},
				PropBarUpdate: buildPropBarUpdate(ps),
			}
		}
	}

	// Check shield for wrong flag (flag on non-mine)
	if message.IsFlag && !result.Result.IsBoom && len(result.Cell) > 0 {
		flaggedCell := m.Cell[message.Ids[0]]
		if !flaggedCell.IsMine && ps.ConsumeShield() {
			shieldCount := ps.GetShieldCount()
			userName, _ := nameCache.GetName(playerID)

			return Response{
				NewPlayer:      newPlayer,
				UserName:       userName,
				ChangeCell:     result,
				TimeStamp:      timeStamp,
				StartTimeStamp: m.StartTimeStamp,
				EarnScore:      0,
				ShieldProtect: &ShieldProtect{
					ShieldCount: shieldCount,
					CellID:      message.Ids[0],
					WasMine:     false,
				},
				PropBarUpdate: buildPropBarUpdate(ps),
			}
		}
	}

	var response Response

	// Prop drop check for normal opens
	earnScore := scoreCalculatorBase(message, result)
	doubleScoreActive := false
	inZone := false

	if !message.IsFlag && result.Result.Message != "shield protected" {
		var propDrops []PropDropInfo
		for _, c := range result.Cell {
			if c.IsOpen && !c.IsMine && c.PropID != 0 {
				propID := c.PropID
				ps.AddProp(propID, 1)
				m.Cell[c.Id].PropID = 0 // clear prop from cell

				def, _ := PropDefs[propID]
				propDrops = append(propDrops, PropDropInfo{
					PropID:   propID,
					PropName: def.Name,
					Count:    1,
				})

				// Auto-activate passive props
				if propID == PropDoubleScore {
					ps.ActivateDoubleScore(time.Duration(config.Props.DoubleScoreDuration) * time.Second)
				}
				if propID == PropShield {
					ps.AddShield(1)
				}
			}
		}
		if len(propDrops) > 0 {
			response.PropDrop = &propDrops[0]
		}

		doubleScoreActive = ps.IsDoubleScoreActive()
		inZone = m.anyInDoubleScoreZone(message.Ids)
		if earnScore > 0 {
			earnScore = calculateScoreWithContext(message, result, doubleScoreActive, inZone)
		}
	}

	response.NewPlayer = newPlayer
	response.ChangeCell = result
	response.TimeStamp = timeStamp
	response.StartTimeStamp = m.StartTimeStamp
	response.EarnScore = earnScore
	response.PropBarUpdate = buildPropBarUpdate(ps)
	response.ZoneInfo = m.zoneToZoneData()

	if name, ok := nameCache.GetName(playerID); ok {
		response.UserName = name
	} else {
		response.UserName = ""
	}
	scoreBoard.addScore(response.UserName, earnScore)
	response.ScoreBoard = scoreBoard.Board

	return response
}

func buildPropBarUpdate(ps *utils.PlayerState) *PropBarUpdate {
	inventory := ps.GetInventory()
	slots := make([]PropSlot, 0)
	for propID, count := range inventory {
		def, ok := PropDefs[propID]
		if !ok {
			continue
		}
		slots = append(slots, PropSlot{
			PropID: propID,
			Name:   def.Name,
			Count:  count,
		})
	}
	doubleScoreActive := ps.IsDoubleScoreActive()
	doubleRemaining := ps.GetDoubleScoreRemaining()
	shieldCount := ps.GetShieldCount()

	return &PropBarUpdate{
		Inventory:            slots,
		DoubleScoreActive:    doubleScoreActive,
		DoubleScoreRemaining: doubleRemaining,
		ShieldCount:          shieldCount,
	}
}

func buildPropCounts(config PropConfig) map[int]int {
	return map[int]int{
		PropDetector:    config.DetectorCount,
		PropXJBD:        config.XJBDCount,
		PropDoubleScore: config.DoubleScoreCount,
		PropShield:      config.ShieldCount,
	}
}
