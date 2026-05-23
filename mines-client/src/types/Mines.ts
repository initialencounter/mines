interface Cell {
    Id: number
    Mines: number
    IsMine: boolean
    IsOpen: boolean
    IsFlagged: boolean
}

interface Zone {
    StartRow: number
    StartCol: number
    EndRow: number
    EndCol: number
    Type: "doubleScore" | "noFlag"
}

interface Minefield {
    Width: number
    Height: number
    Cells: number
    Mines: number
    Cell: Cell[]
    First: boolean
    StartTimeStamp: number
    Zones?: Zone[]
}

interface PropSlot {
    PropID: number
    Name: string
    Count: number
}

interface PropBarUpdate {
    Inventory: PropSlot[]
    DoubleScoreActive: boolean
    DoubleScoreRemaining: number
    ShieldCount: number
}

interface DetectorResult {
    CenterCell: number
    MineCells: number[]
    SafeCells: number[]
}

interface PropEffectInfo {
    PropID: number
    UserName: string
    TargetCell: number
}

interface ShieldProtect {
    ShieldCount: number
    CellID: number
    WasMine: boolean
}

export type { Minefield, Cell, Zone, PropSlot, PropBarUpdate, DetectorResult, PropEffectInfo, ShieldProtect };
