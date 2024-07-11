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

interface Response  {
    PlayerQuit: boolean
    NewPlayer: boolean
    UserId: string
    ChangeCell: Result
    TimeStamp: number
    StartTimeStamp: number
}

export type {Response, Result};