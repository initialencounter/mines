interface Cell {
    Id: number
    Mines: number
    IsMine: boolean
    IsOpen: boolean
    IsFlagged: boolean
}

interface Minefield {
    Width: number
    Height: number
    Cells: number
    Mines: number
    Cell: Cell[]
    First: boolean
    StartTimeStamp: number
}

export type {Minefield, Cell};