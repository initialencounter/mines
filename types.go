package main

import "sync"

// --- 扫雷核心数据结构 ---

// Cell 表示雷区中的一个格子
type Cell struct {
	Id        int
	Mines     int  // 周围雷的数量
	IsMine    bool // 是否是雷
	IsOpen    bool // 是否已翻开
	IsFlagged bool // 是否被标记（插旗）
	PropID    int  `json:"-"` // 格子上的道具 ID（不暴露给客户端）
}

// ZoneType 区域类型
type ZoneType int

const (
	ZoneDoubleScore ZoneType = 0
	ZoneNoFlag      ZoneType = 1
)

func (z ZoneType) MarshalJSON() ([]byte, error) {
	if z == ZoneNoFlag {
		return []byte(`"noFlag"`), nil
	}
	return []byte(`"doubleScore"`), nil
}

// Zone 表示雷区中的特殊区域
type Zone struct {
	StartRow int      `json:"StartRow"`
	StartCol int      `json:"StartCol"`
	EndRow   int      `json:"EndRow"`
	EndCol   int      `json:"EndCol"`
	Type     ZoneType `json:"Type"`
}

type Cells = []Cell

// Minefield 扫雷地图
type Minefield struct {
	mu             sync.Mutex
	Width          int
	Height         int
	Cells          int
	Mines          int
	Cell           Cells
	First          bool // 是否是首次操作（首次操作后才布雷，保证安全区）
	StartTimeStamp int64
	EndTimeStamp   int64
	IsWind         bool // 是否已胜利
	Zones          []Zone
	PropCounts     map[int]int
}

// --- 请求/响应数据结构 ---

// Result 操作结果（boom/win/继续）
type Result struct {
	IsWin       bool
	IsBoom      bool
	RemainCells int
	Message     string
}

// ChangeCell 单次操作翻开的格子集合
type ChangeCell struct {
	Result Result
	Cell   Cells
}

// Request 客户端发来的操作请求
type Request struct {
	Ids        []int  `json:"Ids"`
	IsFlag     bool   `json:"IsFlag"`
	TimeStamp  int64  `json:"TimeStamp"`
	ActionType string `json:"ActionType"`
	PropID     int    `json:"PropID"`
	TargetCell int    `json:"TargetCell"`
}

// PropDropInfo 道具掉落信息
type PropDropInfo struct {
	PropID   int    `json:"PropID"`
	PropName string `json:"PropName"`
	Count    int    `json:"Count"`
}

// DetectorResult 探测仪扫描结果
type DetectorResult struct {
	CenterCell int   `json:"CenterCell"`
	MineCells  []int `json:"MineCells"`
	SafeCells  []int `json:"SafeCells"`
}

// PropSlot 道具栏中的一个格位
type PropSlot struct {
	PropID int    `json:"PropID"`
	Name   string `json:"Name"`
	Count  int    `json:"Count"`
}

// PropBarUpdate 道具栏状态更新
type PropBarUpdate struct {
	Inventory            []PropSlot `json:"Inventory"`
	DoubleScoreActive    bool       `json:"DoubleScoreActive"`
	DoubleScoreRemaining int        `json:"DoubleScoreRemaining"`
	ShieldCount          int        `json:"ShieldCount"`
}

// ZoneData 区域信息（扁平化后发给客户端）
type ZoneData struct {
	StartRow int    `json:"StartRow"`
	StartCol int    `json:"StartCol"`
	EndRow   int    `json:"EndRow"`
	EndCol   int    `json:"EndCol"`
	Type     string `json:"Type"`
}

// InitMessage 新玩家连接时的初始化消息
type InitMessage struct {
	MessageType    string         `json:"MessageType"`
	Minefield      *Minefield     `json:"Minefield"`
	ScoreBoard     map[string]int `json:"ScoreBoard"`
	StartTimeStamp int64          `json:"StartTimeStamp"`
	SafeCells      int            `json:"SafeCells"`
	UserName       string         `json:"UserName"`
}

// PropEffectInfo 道具使用广播（通知其他玩家有人用了道具）
type PropEffectInfo struct {
	PropID     int    `json:"PropID"`
	UserName   string `json:"UserName"`
	TargetCell int    `json:"TargetCell"`
}

// ShieldProtect 护盾保护结果
type ShieldProtect struct {
	ShieldCount int  `json:"ShieldCount"`
	CellID      int  `json:"CellID"`
	WasMine     bool `json:"WasMine"`
}

// Response 服务端推送的响应消息
type Response struct {
	PlayerQuit     bool            `json:"PlayerQuit"`
	NewPlayer      bool            `json:"NewPlayer"`
	UserName       string          `json:"UserName"`
	ChangeCell     ChangeCell      `json:"ChangeCell"`
	TimeStamp      int64           `json:"TimeStamp"`
	StartTimeStamp int64           `json:"StartTimeStamp"`
	EarnScore      int             `json:"EarnScore"`
	ScoreBoard     map[string]int  `json:"ScoreBoard"`
	SafeCells      int             `json:"SafeCells,omitempty"`
	MessageType    string          `json:"MessageType,omitempty"`
	PropDrop       *PropDropInfo   `json:"PropDrop,omitempty"`
	DetectorResult *DetectorResult `json:"DetectorResult,omitempty"`
	PropBarUpdate  *PropBarUpdate  `json:"PropBarUpdate,omitempty"`
	ZoneInfo       []ZoneData      `json:"ZoneInfo,omitempty"`
	PropEffect     *PropEffectInfo `json:"PropEffect,omitempty"`
	ShieldProtect  *ShieldProtect  `json:"ShieldProtect,omitempty"`
}
