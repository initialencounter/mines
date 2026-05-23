import type { Cell, PropBarUpdate, DetectorResult, PropEffectInfo, ShieldProtect } from "@/types/Mines";

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

interface PropDropInfo {
    PropID: number
    PropName: string
    Count: number
}

interface ZoneData {
    StartRow: number
    StartCol: number
    EndRow: number
    EndCol: number
    Type: "doubleScore" | "noFlag"
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
    MessageType?: string
    PropDrop?: PropDropInfo
    DetectorResult?: DetectorResult
    PropBarUpdate?: PropBarUpdate
    ZoneInfo?: ZoneData[]
    PropEffect?: PropEffectInfo
    ShieldProtect?: ShieldProtect
}

interface RequestType {
    Ids: number[],
    IsFlag: boolean,
    TimeStamp: number,
    ActionType?: string,
    PropID?: number,
    TargetCell?: number,
}

export type { Response, Result, ScoreBoard, RequestType, PropDropInfo, ZoneData };
