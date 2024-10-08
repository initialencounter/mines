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

func scoreCalculator(message Request, result ChangeCell) int {
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
