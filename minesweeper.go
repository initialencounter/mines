package main

import (
	"math"
	"math/rand/v2"
	"sync"
	"time"
)

type Cell struct {
	Id        int
	Mines     int
	IsMine    bool
	IsOpen    bool
	IsFlagged bool
	PropID    int `json:"-"`
}

type ZoneType int

const (
	ZoneDoubleScore ZoneType = 0
	ZoneNoFlag      ZoneType = 1
)

type Zone struct {
	StartRow int      `json:"StartRow"`
	StartCol int      `json:"StartCol"`
	EndRow   int      `json:"EndRow"`
	EndCol   int      `json:"EndCol"`
	Type     ZoneType `json:"Type"`
}

type Cells = []Cell

type Minefield struct {
	mu            sync.Mutex
	Width         int
	Height        int
	Cells         int
	Mines         int
	Cell          Cells
	First         bool
	StartTimeStamp int64
	EndTimeStamp  int64
	IsWind        bool
	Zones         []Zone
	PropCounts    map[int]int
}

type Result struct {
	IsWin       bool
	IsBoom      bool
	RemainCells int
	Message     string
}

type ChangeCell struct {
	Result Result
	Cell   Cells
}

type Request struct {
	Ids        []int  `json:"Ids"`
	IsFlag     bool   `json:"IsFlag"`
	TimeStamp  int64  `json:"TimeStamp"`
	ActionType string `json:"ActionType"`
	PropID     int    `json:"PropID"`
	TargetCell int    `json:"TargetCell"`
}

// --- Response extension types ---

type PropDropInfo struct {
	PropID   int    `json:"PropID"`
	PropName string `json:"PropName"`
	Count    int    `json:"Count"`
}

type DetectorResult struct {
	CenterCell int   `json:"CenterCell"`
	MineCells  []int `json:"MineCells"`
	SafeCells  []int `json:"SafeCells"`
}

type PropSlot struct {
	PropID int    `json:"PropID"`
	Name   string `json:"Name"`
	Count  int    `json:"Count"`
}

type PropBarUpdate struct {
	Inventory            []PropSlot `json:"Inventory"`
	DoubleScoreActive    bool       `json:"DoubleScoreActive"`
	DoubleScoreRemaining int        `json:"DoubleScoreRemaining"`
	ShieldCount          int        `json:"ShieldCount"`
}

type ZoneData struct {
	StartRow int    `json:"StartRow"`
	StartCol int    `json:"StartCol"`
	EndRow   int    `json:"EndRow"`
	EndCol   int    `json:"EndCol"`
	Type     string `json:"Type"`
}

type PropEffectInfo struct {
	PropID     int    `json:"PropID"`
	UserName   string `json:"UserName"`
	TargetCell int    `json:"TargetCell"`
}

type ShieldProtect struct {
	ShieldCount int  `json:"ShieldCount"`
	CellID      int  `json:"CellID"`
	WasMine     bool `json:"WasMine"`
}

type Response struct {
	PlayerQuit     bool            `json:"PlayerQuit"`
	NewPlayer      bool            `json:"NewPlayer"`
	UserName       string          `json:"UserName"`
	ChangeCell     ChangeCell      `json:"ChangeCell"`
	TimeStamp      int64           `json:"TimeStamp"`
	StartTimeStamp int64           `json:"StartTimeStamp"`
	EarnScore      int             `json:"EarnScore"`
	ScoreBoard     map[string]int  `json:"ScoreBoard"`

	// New optional fields for prop system
	MessageType    string          `json:"MessageType,omitempty"`
	PropDrop       *PropDropInfo   `json:"PropDrop,omitempty"`
	DetectorResult *DetectorResult `json:"DetectorResult,omitempty"`
	PropBarUpdate  *PropBarUpdate  `json:"PropBarUpdate,omitempty"`
	ZoneInfo       []ZoneData      `json:"ZoneInfo,omitempty"`
	PropEffect     *PropEffectInfo `json:"PropEffect,omitempty"`
	ShieldProtect  *ShieldProtect  `json:"ShieldProtect,omitempty"`
}

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

// countNewlyOpened returns the number of cells in ids that were NOT already open
func (m *Minefield) countNewlyOpened(ids []int) int {
	count := 0
	for _, id := range ids {
		if id >= 0 && id < m.Cells && !m.Cell[id].IsOpen {
			count++
		}
	}
	return count
}

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

// getCellsInRange returns all cell IDs in a rectangular area centered on targetCell
// with the given radius (e.g., radius=2 gives 5x5, radius=3 gives 7x7)
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

// useDetector scans 5x5 range and returns mine/safe cell lists without opening anything
func (m *Minefield) useDetector(targetCell int) DetectorResult {
	cellsInRange := m.getCellsInRange(targetCell, 2) // 5x5
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

// useXJBD auto-completes 7x7 area: flags all mines, opens all safe cells (with chain reaction)
func (m *Minefield) useXJBD(targetCell int) ChangeCell {
	cellsInRange := m.getCellsInRange(targetCell, 3) // 7x7
	seen := make(map[int]bool)
	for _, id := range cellsInRange {
		seen[id] = true
	}

	var changes []Cell

	// First pass: flag all mines in range (skip no-flag zones)
	for id := range seen {
		if id < 0 || id >= m.Cells {
			continue
		}
		if m.Cell[id].IsOpen || m.Cell[id].IsFlagged {
			continue
		}
		if m.Cell[id].IsMine {
			if m.isInNoFlagZone(id) {
				continue // skip mines in no-flag zones
			}
			m.Cell[id].IsFlagged = true
			m.Cell[id].IsOpen = true
			changes = append(changes, m.Cell[id])
		}
	}

	// Second pass: open safe cells (with chain reaction for 0-value cells)
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

// autoOpenCellsFrom is like autoOpenCells but tracks visited cells and respects no-flag zone
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
			continue // don't open mines during chain reaction
		}
		m.Cell[cid].IsOpen = true
		changes = append(changes, m.Cell[cid])
		if m.Cell[cid].Mines == 0 {
			changes = append(changes, m.autoOpenCellsFrom(cid, visited)...)
		}
	}
	return changes
}

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

// anyInDoubleScoreZone returns true if any of the given IDs is in a double-score zone
func (m *Minefield) anyInDoubleScoreZone(ids []int) bool {
	for _, id := range ids {
		if m.isInDoubleScoreZone(id) {
			return true
		}
	}
	return false
}

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

func (m *Minefield) getChangeCells(changeCell []int) Cells {
	var change Cells
	for i := 0; i < len(changeCell); i++ {
		change = append(change, m.Cell[i])
	}
	return change
}

func (m *Minefield) isLost() bool {
	for i, n := 0, len(m.Cell); i < n; i++ {
		if m.Cell[i].IsOpen && m.Cell[i].IsMine {
			return true
		}
	}
	return false
}

func (m *Minefield) randomShot(ignore []int) {
	contain := func(arr []int, target int) bool {
		for _, v := range arr {
			if v == target {
				return true
			}
		}
		return false
	}
	count := 0
	for {
		randInt := rand.IntN(m.Cells)
		if contain(ignore, randInt) {
			continue
		}
		if m.Cell[randInt].IsMine {
			continue
		} else {
			m.Cell[randInt].IsMine = true
			count++
		}
		if count == m.Mines {
			break
		}
	}
	m.scatterProps(ignore)
}

func (m *Minefield) scatterProps(ignore []int) {
	if len(m.PropCounts) == 0 {
		return
	}

	// Build flat prop list
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

	// Collect available cells: non-mine, not ignored, no prop yet
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

// contain is a helper used in scatterProps
func contain(arr []int, target int) bool {
	for _, v := range arr {
		if v == target {
			return true
		}
	}
	return false
}

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

func (m *Minefield) openMinefield() Minefield {
	var om Minefield
	om.Mines = m.Mines
	om.Width = m.Width
	om.Height = m.Height
	om.Cells = m.Cells
	om.Zones = m.Zones
	om.Cell = make([]Cell, m.Cells)
	for i := 0; i < m.Cells; i++ {
		if m.Cell[i].IsOpen {
			om.Cell[i] = m.Cell[i]
		} else {
			om.Cell[i] = Cell{i, 9, false, false, false, 0}
		}
	}
	return om
}

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

func newMinefield(mines, width, height int, zones []Zone, propCounts map[int]int) Minefield {
	var m Minefield
	m.Mines = mines
	m.Width = width
	m.Height = height
	m.Cells = width * height
	m.Cell = make([]Cell, m.Cells)
	m.First = true
	m.Zones = zones
	m.PropCounts = propCounts
	for i := 0; i < m.Cells; i++ {
		m.Cell[i] = Cell{i, 0, false, false, false, 0}
	}
	return m
}
