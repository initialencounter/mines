package main

import (
	"math"
	"math/rand/v2"
	"time"
)

// --- 格子操作 ---

// doFlag 标记/取消标记一个格子
func (m *Minefield) doFlag(id int) ChangeCell {
	if m.First {
		m.StartTimeStamp = time.Now().UnixMilli()
		m.First = false
		ignoreCells := m.getNearbyCells(id)
		m.randomShot(ignoreCells)
		m.countMines()
	}
	var changes []Cell
	m.Cell[id].IsOpen = true
	changes = append(changes, m.Cell[id])
	stats := m.getStats(id)
	return ChangeCell{stats, changes}
}

// openCells 翻开指定的格子，会自动展开零值格子的连锁反应
func (m *Minefield) openCells(ids []int) ChangeCell {
	if m.First {
		m.StartTimeStamp = time.Now().UnixMilli()
		m.First = false
		ignoreCells := m.getNearbyCells(ids[0])
		m.randomShot(ignoreCells)
		m.countMines()
	}
	var changes []Cell
	for i := 0; i < len(ids); i++ {
		id := ids[i]
		m.Cell[id].IsOpen = true
		if m.Cell[id].Mines == 0 && !m.Cell[id].IsMine {
			changes = append(changes, m.autoOpenCells(id)...)
		}
		changes = append(changes, m.Cell[id])
	}
	stats := m.getStats(ids[0])
	return ChangeCell{stats, changes}
}

// --- 相邻格子计算 ---

// getNearbyCells 返回指定格子的所有相邻格子 ID（最多 8 个）
func (m *Minefield) getNearbyCells(id int) []int {
	var nearbyCells []int
	width := m.Width
	height := m.Height
	x := id % width
	y := int(math.Floor(float64(id / width)))
	isNotFirstRow := y > 0
	isNotLastRow := y < height-1

	if isNotFirstRow {
		nearbyCells = append(nearbyCells, id-width)
	}
	if isNotLastRow {
		nearbyCells = append(nearbyCells, id+width)
	}

	if x > 0 {
		nearbyCells = append(nearbyCells, id-1)
		if isNotFirstRow {
			nearbyCells = append(nearbyCells, id-width-1)
		}
		if isNotLastRow {
			nearbyCells = append(nearbyCells, id+width-1)
		}
	}
	if x < width-1 {
		nearbyCells = append(nearbyCells, id+1)
		if isNotFirstRow {
			nearbyCells = append(nearbyCells, id-width+1)
		}
		if isNotLastRow {
			nearbyCells = append(nearbyCells, id+width+1)
		}
	}

	return nearbyCells
}

// getCellsInRange 返回以 targetCell 为中心、半径 radius 的矩形区域内所有格子 ID
func (m *Minefield) getCellsInRange(targetCell, radius int) []int {
	var cells []int
	w := m.Width
	h := m.Height
	cx := targetCell % w
	cy := targetCell / w
	r0 := max(0, cy-radius)
	r1 := min(h-1, cy+radius)
	c0 := max(0, cx-radius)
	c1 := min(w-1, cx+radius)
	for r := r0; r <= r1; r++ {
		for c := c0; c <= c1; c++ {
			cells = append(cells, r*w+c)
		}
	}
	return cells
}

// --- 自动展开 ---

// autoOpenCells 连锁展开零值格子的相邻格子
func (m *Minefield) autoOpenCells(id int) Cells {
	var changes []Cell
	round := m.getNearbyCells(id)
	for i := 0; i < len(round); i++ {
		if m.Cell[round[i]].IsOpen {
			continue
		}
		m.Cell[round[i]].IsOpen = true
		changes = append(changes, m.Cell[round[i]])
		if m.Cell[round[i]].Mines == 0 {
			changes = append(changes, m.autoOpenCells(round[i])...)
		}
	}
	return changes
}

// autoOpenCellsFrom 带 visited 跟踪的自动展开，用于 XJBD 道具，不会翻开雷
func (m *Minefield) autoOpenCellsFrom(id int, visited map[int]bool) Cells {
	var changes []Cell
	round := m.getNearbyCells(id)
	for i := 0; i < len(round); i++ {
		cid := round[i]
		if visited[cid] {
			continue
		}
		visited[cid] = true
		if m.Cell[cid].IsOpen || m.Cell[cid].IsFlagged {
			continue
		}
		if m.Cell[cid].IsMine {
			continue
		}
		m.Cell[cid].IsOpen = true
		changes = append(changes, m.Cell[cid])
		if m.Cell[cid].Mines == 0 {
			changes = append(changes, m.autoOpenCellsFrom(cid, visited)...)
		}
	}
	return changes
}

// --- 雷区初始化 ---

// newMinefield 创建新的雷区，初始状态所有格子未翻开
func newMinefield(mines, width, height int, zones []Zone, propCounts map[int]int) *Minefield {
	m := &Minefield{
		Mines:      mines,
		Width:      width,
		Height:     height,
		Cells:      width * height,
		Cell:       make([]Cell, width*height),
		First:      true,
		Zones:      zones,
		PropCounts: propCounts,
	}
	for i := 0; i < m.Cells; i++ {
		m.Cell[i] = Cell{Id: i}
	}
	return m
}

// randomShot 随机布雷，ignore 列表中的格子不会被布雷
func (m *Minefield) randomShot(ignore []int) {
	count := 0
	for {
		randInt := rand.IntN(m.Cells)
		if contain(ignore, randInt) {
			continue
		}
		if !m.Cell[randInt].IsMine {
			m.Cell[randInt].IsMine = true
			count++
		}
		if count == m.Mines {
			break
		}
	}
	m.scatterProps(ignore)
}

// scatterProps 将道具随机分配到非雷、非忽略、无道具的格子上
func (m *Minefield) scatterProps(ignore []int) {
	if len(m.PropCounts) == 0 {
		return
	}

	var propList []int
	for propID, cnt := range m.PropCounts {
		for i := 0; i < cnt; i++ {
			propList = append(propList, propID)
		}
	}
	if len(propList) == 0 {
		return
	}
	rand.Shuffle(len(propList), func(i, j int) {
		propList[i], propList[j] = propList[j], propList[i]
	})

	var available []int
	for i := 0; i < m.Cells; i++ {
		if !m.Cell[i].IsMine && m.Cell[i].PropID == 0 && !contain(ignore, i) {
			available = append(available, i)
		}
	}
	rand.Shuffle(len(available), func(i, j int) {
		available[i], available[j] = available[j], available[i]
	})

	for i, propID := range propList {
		if i >= len(available) {
			break
		}
		m.Cell[available[i]].PropID = propID
	}
}

// initAndOpenFirstCells 服务端首次布雷并随机翻开 n 个零值格子作为开局
func (m *Minefield) initAndOpenFirstCells(n int) ChangeCell {
	m.StartTimeStamp = time.Now().UnixMilli()
	m.First = false
	m.randomShot(nil)
	m.countMines()

	var zeroCells []int
	for i := 0; i < m.Cells; i++ {
		if !m.Cell[i].IsMine && m.Cell[i].Mines == 0 {
			zeroCells = append(zeroCells, i)
		}
	}

	rand.Shuffle(len(zeroCells), func(i, j int) {
		zeroCells[i], zeroCells[j] = zeroCells[j], zeroCells[i]
	})

	if len(zeroCells) > n {
		zeroCells = zeroCells[:n]
	}

	if len(zeroCells) == 0 {
		return ChangeCell{}
	}

	return m.openCells(zeroCells)
}

// countMines 计算每个格子周围雷的数量
func (m *Minefield) countMines() {
	for id := 0; id < m.Cells; id++ {
		round := m.getNearbyCells(id)
		var count = 0
		for i := 0; i < len(round); i++ {
			if m.Cell[round[i]].IsMine {
				count++
			}
		}
		m.Cell[id].Mines = count
	}
}

// --- 道具 ---

// useDetector 探测仪扫描 5x5 范围，返回雷和安全的格子列表（不翻开）
func (m *Minefield) useDetector(targetCell int) DetectorResult {
	cellsInRange := m.getCellsInRange(targetCell, 2)
	var mineCells, safeCells []int
	for _, id := range cellsInRange {
		if id < 0 || id >= m.Cells {
			continue
		}
		if m.Cell[id].IsMine {
			mineCells = append(mineCells, id)
		} else {
			safeCells = append(safeCells, id)
		}
	}
	return DetectorResult{
		CenterCell: targetCell,
		MineCells:  mineCells,
		SafeCells:  safeCells,
	}
}

// useXJBD 雷之奥义：自动标记 7x7 范围内的雷并翻开安全格子
func (m *Minefield) useXJBD(targetCell int) ChangeCell {
	cellsInRange := m.getCellsInRange(targetCell, 3)
	seen := make(map[int]bool)
	for _, id := range cellsInRange {
		seen[id] = true
	}

	var changes []Cell

	// 第一遍：标记所有雷（跳过 no-flag 区域的雷）
	for id := range seen {
		if id < 0 || id >= m.Cells {
			continue
		}
		if m.Cell[id].IsOpen || m.Cell[id].IsFlagged {
			continue
		}
		if m.Cell[id].IsMine {
			if m.isInNoFlagZone(id) {
				continue
			}
			m.Cell[id].IsFlagged = true
			m.Cell[id].IsOpen = true
			changes = append(changes, m.Cell[id])
		}
	}

	// 第二遍：翻开安全格子（零值格子会连锁展开）
	for id := range seen {
		if id < 0 || id >= m.Cells {
			continue
		}
		if m.Cell[id].IsOpen || m.Cell[id].IsFlagged {
			continue
		}
		if !m.Cell[id].IsMine {
			m.Cell[id].IsOpen = true
			changes = append(changes, m.Cell[id])
			if m.Cell[id].Mines == 0 {
				changes = append(changes, m.autoOpenCellsFrom(id, seen)...)
			}
		}
	}

	stats := m.getStats(targetCell)
	return ChangeCell{stats, changes}
}

// --- 区域判断 ---

// isInNoFlagZone 判断格子是否在禁旗区域内
func (m *Minefield) isInNoFlagZone(id int) bool {
	w := m.Width
	row := id / w
	col := id % w
	for _, z := range m.Zones {
		if z.Type == ZoneNoFlag &&
			row >= z.StartRow && row <= z.EndRow &&
			col >= z.StartCol && col <= z.EndCol {
			return true
		}
	}
	return false
}

// isInDoubleScoreZone 判断格子是否在双倍积分区域内
func (m *Minefield) isInDoubleScoreZone(id int) bool {
	w := m.Width
	row := id / w
	col := id % w
	for _, z := range m.Zones {
		if z.Type == ZoneDoubleScore &&
			row >= z.StartRow && row <= z.EndRow &&
			col >= z.StartCol && col <= z.EndCol {
			return true
		}
	}
	return false
}

// anyInDoubleScoreZone 判断任意格子是否在双倍积分区域内
func (m *Minefield) anyInDoubleScoreZone(ids []int) bool {
	for _, id := range ids {
		if m.isInDoubleScoreZone(id) {
			return true
		}
	}
	return false
}

// --- 状态查询 ---

// getStats 根据操作后的状态返回 Result（win/boom/ok）
func (m *Minefield) getStats(id int) Result {
	RemainCells := 0
	isWin := false
	isBoom := false
	for i := 0; i < m.Cells; i++ {
		if m.Cell[id].IsOpen && m.Cell[id].IsMine {
			isBoom = true
		}
		if m.Cell[i].IsOpen && !m.Cell[i].IsMine {
			RemainCells++
		}
	}
	if RemainCells == m.Cells-m.Mines {
		isWin = true
		m.IsWind = true
	}
	return Result{isWin, isBoom, RemainCells, "ok"}
}

// RemainCells 返回剩余未翻开的安全格子数
func (m *Minefield) RemainCells() int {
	count := 0
	for i := 0; i < m.Cells; i++ {
		if !m.Cell[i].IsOpen && !m.Cell[i].IsMine {
			count++
		}
	}
	return count
}

// openMinefield 返回对客户端可见的雷区快照（未翻开的格子内容被隐藏）
func (m *Minefield) openMinefield() *Minefield {
	om := &Minefield{
		Mines:  m.Mines,
		Width:  m.Width,
		Height: m.Height,
		Cells:  m.Cells,
		Zones:  m.Zones,
		Cell:   make([]Cell, m.Cells),
	}
	for i := 0; i < m.Cells; i++ {
		if m.Cell[i].IsOpen {
			om.Cell[i] = m.Cell[i]
		} else {
			om.Cell[i] = Cell{Id: i, Mines: 9}
		}
	}
	return om
}

func (m *Minefield) zoneToZoneData() []ZoneData {
	data := make([]ZoneData, 0)
	for _, z := range m.Zones {
		typeStr := "doubleScore"
		if z.Type == ZoneNoFlag {
			typeStr = "noFlag"
		}
		data = append(data, ZoneData{
			StartRow: z.StartRow,
			StartCol: z.StartCol,
			EndRow:   z.EndRow,
			EndCol:   z.EndCol,
			Type:     typeStr,
		})
	}
	return data
}

// --- 胜利结果处理 ---

// applyWinResult 如果已胜利，将结果替换为统一的胜利响应，并锁定结束时间戳
func (m *Minefield) applyWinResult(result ChangeCell, timeStamp int64) (ChangeCell, int64) {
	if !m.IsWind {
		return result, timeStamp
	}
	result = ChangeCell{
		Result: Result{
			IsWin: true, IsBoom: false, RemainCells: 0, Message: "You Win!",
		},
		Cell: []Cell{},
	}
	if m.EndTimeStamp == 0 {
		m.EndTimeStamp = timeStamp
	}
	return result, m.EndTimeStamp
}

// --- 工具函数 ---

// contain 检查 arr 中是否包含 target
func contain(arr []int, target int) bool {
	for _, v := range arr {
		if v == target {
			return true
		}
	}
	return false
}
