package main

import (
	"math"
	"math/rand/v2"
	"time"
)

type Cell struct {
	Id        int
	Mines     int
	IsMine    bool
	IsOpen    bool
	IsFlagged bool
}
type Cells = []Cell

type Minefield struct {
	Width          int
	Height         int
	Cells          int
	Mines          int
	Cell           Cells
	First          bool
	StartTimeStamp int64
	EndTimeStamp   int64
	IsWind         bool
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
	Ids       []int
	IsFlag    bool
	TimeStamp int64
}

type Response struct {
	PlayerQuit     bool
	NewPlayer      bool
	UserName       string
	ChangeCell     ChangeCell
	TimeStamp      int64
	StartTimeStamp int64
	EarnScore      int
	ScoreBoard     map[string]int
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
	om.Cell = make([]Cell, m.Cells)
	for i := 0; i < m.Cells; i++ {
		if m.Cell[i].IsOpen {
			om.Cell[i] = m.Cell[i]
		} else {
			om.Cell[i] = Cell{i, 9, false, false, false}
		}
	}
	return om
}

func (m *Minefield) getStats(id int) Result {
	RemainCells := 0
	isWin := false
	isBoom := false
	for i := 0; i < m.Cells; i++ {
		if m.Cell[i].IsOpen && !m.Cell[i].IsMine {
			RemainCells++
		}
	}
	if RemainCells == m.Cells-m.Mines {
		isWin = true
		m.IsWind = true
	}
	if m.Cell[id].IsMine && m.Cell[id].IsOpen {
		isBoom = true
	}
	return Result{isWin, isBoom, RemainCells, "ok"}
}

func newMinefield(mines, width, height int) Minefield {
	var m Minefield
	m.Mines = mines
	m.Width = width
	m.Height = height
	m.Cells = width * height
	m.Cell = make([]Cell, m.Cells)
	m.First = true
	for i := 0; i < m.Cells; i++ {
		m.Cell[i] = Cell{i, 0, false, false, false}
	}
	return m
}
