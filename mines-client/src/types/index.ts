// ========== 格子 ==========
export interface Cell {
  Id: number
  Mines: number // 0-8 数字, 9 = 未打开(反作弊)
  IsMine: boolean
  IsOpen: boolean
  IsFlagged: boolean
}

// ========== 特殊区域 ==========
export interface ZoneRect {
  startRow: number
  startColumn: number
  endRow: number
  endColumn: number
}

export interface ZoneData {
  StartRow: number
  StartCol: number
  EndRow: number
  EndCol: number
  Type: string // "doubleScore" | "noFlag"
}

// ========== 道具 ==========
export interface PropSlot {
  PropID: number
  Name: string
  Count: number
}

export interface PropBarUpdate {
  Inventory: PropSlot[]
  DoubleScoreActive: boolean
  DoubleScoreRemaining: number // seconds
  ShieldCount: number
}

export interface PropDropInfo {
  PropID: number
  PropName: string
  Count: number
}

export interface PropEffectInfo {
  PropID: number
  UserName: string
  TargetCell: number
}

export interface ShieldProtect {
  ShieldCount: number
  CellID: number
  WasMine: boolean
}

// ========== 探测仪结果 ==========
export interface DetectorResult {
  CenterCell: number
  MineCells: number[]
  SafeCells: number[]
}

// ========== 地图数据 ==========
export interface Minefield {
  Width: number
  Height: number
  Cells: number
  Mines: number
  Cell: Cell[]
  Zones: ZoneData[]
  First: boolean
  StartTimeStamp: number
}

// ========== 记分板 ==========
export interface ScoreBoard {
  [userName: string]: number
}

// ========== 激活道具效果 ==========
export interface ActivePropEffect {
  propId: number
  propName: string
  remainingMs: number
  startTime: number
  centerCol?: number
  centerRow?: number
}

// ========== 游戏结果 ==========
export interface GameResult {
  IsWin: boolean
  IsBoom: boolean
  RemainCells: number
  Message: string
}

export interface ChangeCell {
  Result: GameResult
  Cell: Cell[]
}

// ========== WebSocket 请求 ==========
export interface GameRequest {
  Ids: number[]
  IsFlag: boolean
  TimeStamp: number
  ActionType?: string
  PropID?: number
  TargetCell?: number
}

// ========== WebSocket 响应 ==========
export interface GameResponse {
  // 消息类型路由
  MessageType?: string // "init" | "personal" | "propEffect" | "detectorResult" | "hintResult"

  // 初始化消息
  Minefield?: Minefield
  SafeCells?: number
  PlayerList?: PlayerInfo[]

  // 玩家信息
  PlayerQuit?: boolean
  NewPlayer?: boolean
  UserName?: string

  // 操作响应
  ChangeCell?: ChangeCell
  TimeStamp?: number
  StartTimeStamp?: number
  EarnScore?: number
  ScoreBoard?: ScoreBoard

  // 道具相关
  PropDrop?: PropDropInfo
  DetectorResult?: DetectorResult
  PropBarUpdate?: PropBarUpdate
  ZoneInfo?: ZoneData[]
  PropEffect?: PropEffectInfo
  ShieldProtect?: ShieldProtect
}

export interface PlayerInfo {
  UserName: string
  Score: number
}
