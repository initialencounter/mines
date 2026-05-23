package main

import (
	"main/database"
	"main/utils"
	"time"
)

// --- 动作处理函数 ---

// handleHint 提示道具：随机返回一个未翻开的安全格子位置
func handleHint(m *Minefield, message Request, playerID int, ps *utils.PlayerState) Response {
	m.mu.Lock()
	defer m.mu.Unlock()

	userName, _ := nameCache.GetName(playerID)

	var safeCells []int
	for i := 0; i < m.Cells; i++ {
		if !m.Cell[i].IsOpen && !m.Cell[i].IsFlagged && !m.Cell[i].IsMine {
			safeCells = append(safeCells, i)
		}
	}
	if len(safeCells) == 0 {
		return Response{
			UserName:    userName,
			MessageType: "hintResult",
			ChangeCell:  ChangeCell{Cell: []Cell{}},
		}
	}

	// 使用时间纳秒取模以获得伪随机结果
	hintID := safeCells[time.Now().UnixNano()%int64(len(safeCells))]

	return Response{
		UserName:    userName,
		MessageType: "hintResult",
		ChangeCell: ChangeCell{
			Cell: []Cell{{Id: hintID}},
		},
	}
}

// handleUseDetector 探测仪：扫描 5x5 范围，返回雷和安全格子的位置（不翻开）
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

// handleUseXJBD 雷之奥义：自动处理 7x7 范围——标记雷、翻开安全格子
func handleUseXJBD(m *Minefield, message Request, playerID int, ps *utils.PlayerState, config *Config) Response {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !ps.HasProp(PropXJBD) {
		return Response{}
	}

	ps.UseProp(PropXJBD)
	result := m.useXJBD(message.TargetCell)

	// 如果游戏已胜利，统一处理胜利结果
	result, timeStamp := m.applyWinResult(result, message.TimeStamp)

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
		SafeCells:      m.RemainCells(),
		PropBarUpdate:  buildPropBarUpdate(ps),
		PropEffect: &PropEffectInfo{
			PropID:     PropXJBD,
			UserName:   userName,
			TargetCell: message.TargetCell,
		},
	}
}

// handleNormalAction 处理普通操作（翻开/标记），包含护盾保护、道具掉落、计分
func handleNormalAction(m *Minefield, message Request, playerID int, ps *utils.PlayerState, config *Config, handler *database.DBHandler, newPlayer bool) Response {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 禁旗区域阻止标记操作
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

	// 执行翻开或标记操作
	var result ChangeCell
	if message.IsFlag {
		result = m.doFlag(message.Ids[0])
	} else {
		result = m.openCells(message.Ids)
	}

	// 如果游戏已胜利，统一处理胜利结果
	result, timeStamp := m.applyWinResult(result, message.TimeStamp)

	// 护盾保护：踩雷时消耗护盾抵消
	if result.Result.IsBoom && message.ActionType == "" && !message.IsFlag {
		if ps.ConsumeShield() {
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

	// 护盾保护：错误标记（标的不是雷）时消耗护盾抵消扣分
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

	// 道具掉落检查（仅普通翻开操作时触发）
	earnScore := scoreCalculatorBase(message, result)
	doubleScoreActive := false
	inZone := false

	if !message.IsFlag && result.Result.Message != "shield protected" {
		var propDrops []PropDropInfo
		for _, c := range result.Cell {
			if c.IsOpen && !c.IsMine && c.PropID != 0 {
				propID := c.PropID
				ps.AddProp(propID, 1)
				m.Cell[c.Id].PropID = 0 // 拾取后清除格子上的道具

				def, _ := PropDefs[propID]
				propDrops = append(propDrops, PropDropInfo{
					PropID:   propID,
					PropName: def.Name,
					Count:    1,
				})

				// 被动道具自动激活
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
	response.SafeCells = m.RemainCells()
	response.PropBarUpdate = buildPropBarUpdate(ps)
	response.ZoneInfo = m.zoneToZoneData()

	if name, ok := nameCache.GetName(playerID); ok {
		response.UserName = name
	}
	scoreBoard.addScore(response.UserName, earnScore)
	response.ScoreBoard = scoreBoard.Board

	return response
}

// --- 道具栏/配置辅助 ---

// buildPropBarUpdate 构建当前玩家道具栏状态
func buildPropBarUpdate(ps *utils.PlayerState) *PropBarUpdate {
	inventory := ps.GetInventory()
	doubleScoreActive := ps.IsDoubleScoreActive()
	doubleRemaining := ps.GetDoubleScoreRemaining()
	shieldCount := ps.GetShieldCount()

	slots := make([]PropSlot, 0)
	for propID, count := range inventory {
		def, ok := PropDefs[propID]
		if !ok {
			continue
		}
		// 双倍积分：仅在激活时显示
		if propID == PropDoubleScore && !doubleScoreActive {
			continue
		}
		// 护盾：同步库存计数
		if propID == PropShield {
			count = shieldCount
			if count <= 0 {
				continue
			}
		}
		slots = append(slots, PropSlot{
			PropID: propID,
			Name:   def.Name,
			Count:  count,
		})
	}

	return &PropBarUpdate{
		Inventory:            slots,
		DoubleScoreActive:    doubleScoreActive,
		DoubleScoreRemaining: doubleRemaining,
		ShieldCount:          shieldCount,
	}
}

// buildPropCounts 将配置中的道具数量转换为 map[propID]count
func buildPropCounts(config PropConfig) map[int]int {
	return map[int]int{
		PropDetector:    config.DetectorCount,
		PropXJBD:        config.XJBDCount,
		PropDoubleScore: config.DoubleScoreCount,
		PropShield:      config.ShieldCount,
	}
}
