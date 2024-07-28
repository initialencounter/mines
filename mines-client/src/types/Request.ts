import type {Cell} from "@/types/Mines";

interface Result {
    Cell: Cell[]
    Result: {
        IsWin: boolean
        IsBoom: boolean
        RemainCells: boolean
        Message: string
        TimeStamp: number
    },
}

interface ScoreBoard {
    [key: string]: number
}

interface Response  {
    PlayerQuit: boolean
    NewPlayer: boolean
    UserName: string
    ChangeCell: Result
    TimeStamp: number
    StartTimeStamp: number
    EarnScore: number
    ScoreBoard: ScoreBoard
}

interface RequestType {
    Ids: number[],
    IsFlag: boolean,
    TimeStamp: number
}

export type {Response, Result, ScoreBoard, RequestType};