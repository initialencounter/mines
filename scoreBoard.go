package main

import (
	"main/database"
	"main/utils"
)

type Board = map[string]int
type Player struct {
	Name  string
	Score int
}
type ScoreBoard struct {
	Board Board
}

func (board *ScoreBoard) clear() {
	board.Board = make(map[string]int)
}

func (board *ScoreBoard) addScore(userName string, score int) {
	oldScore, exists := board.Board[userName]
	if exists {
		board.Board[userName] = score + oldScore
	} else {
		board.Board[userName] = score
	}
}

func (board *ScoreBoard) getTopPlayer() Player {
	var player Player
	for k, v := range board.Board {
		if player.Score < v {
			player = Player{
				Name:  k,
				Score: v,
			}
		}
	}
	return player
}

func newScoreBoard() *ScoreBoard {
	return &ScoreBoard{
		Board: make(map[string]int),
	}
}

// scoreCalculatorUnscored returns the base score without multipliers
func scoreCalculatorBase(message Request, result ChangeCell) int {
	if message.IsFlag {
		if result.Result.IsBoom {
			return 1
		} else {
			return -1
		}
	} else {
		if result.Result.IsBoom {
			return -1
		} else {
			return len(result.Cell)
		}
	}
}

// calculateScoreWithContext applies double-score multipliers
func calculateScoreWithContext(message Request, result ChangeCell, doubleScoreActive bool, inDoubleScoreZone bool) int {
	base := scoreCalculatorBase(message, result)
	if base <= 0 {
		return base // don't multiply negative scores
	}
	multiplier := 1
	if doubleScoreActive {
		multiplier *= 2
	}
	if inDoubleScoreZone {
		multiplier *= 2
	}
	return base * multiplier
}

// scoreCalculator is kept for backward compatibility
func scoreCalculator(message Request, result ChangeCell) int {
	return scoreCalculatorBase(message, result)
}

func clearScoreBoard(board *ScoreBoard, handler *database.DBHandler, nameCache *utils.NameCache) {
	for name, score := range board.Board {
		userId, _ := nameCache.GetId(name)
		err := handler.AddMedal(userId, score)
		if err != nil {
			return
		}
	}
	scoreBoard.clear()
}
