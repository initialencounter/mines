<template>
  <div class="board">
    <div class="timeWatcher">{{ timeWatcher }}</div>
    <div v-for="(cell, index) in minefield.Cell" :key="index"
         :style="{position: 'absolute', top: Math.floor(index/minefield.Width)*94+'px', left: (index%minefield.Width)*94+'px',
         backgroundImage: `url(${getImageSrc(cell)})`}"
         class="cell"
         @mousedown="(event) => handleClick(event,index)">
    </div>
  </div>
</template>

<script lang="ts" setup>
import {ref} from 'vue'
import axios from 'axios'
let host = window.location.hostname
let port = window.location.port
console.log(host)
type Cell = { Id: number, Mines: number, IsMine: boolean, IsOpen: boolean, IsFlagged: boolean }
type Result = {
  Cell: Cell[],
  Result: {
    IsWin: boolean,
    IsBoom: boolean,
    RemainCells: boolean,
    Message: string,
    TimeStamp: number,
  },
}

type Response = {
  ChangeCell: Result
  TimeStamp: number
  StartTimeStamp: number
}

type Minefield = {
  Width: number
  Height: number
  Cells: number
  Mines: number
  Cell: Cell[]
  First: boolean
  StartTimeStamp: number
}

const minefield = ref<Minefield>({
  Width: 5,
  Height: 4,
  Cells: 20,
  Mines: 5,
  Cell: [],
  First: false,
  StartTimeStamp: 0,
})

const isWin = ref(false)
const timeWatcher = ref('00:000')
let startTimeStamp = 0
document.oncontextmenu = () => false;

const reConnect = () => {
  const token = localStorage.getItem('jwt')
  return new WebSocket(`ws://${host}:${port}/ws/101?token=${token}`);
}
const getBoard = async () => {
  let config = {
    method: 'post',
    url: `http://${host}:${port}/getMinefield`,
    headers: {
      'Content-Type': 'application/xml',
      'Accept': '*/*',
    }
  };
  let board = (await axios(config)).data
  console.log(board)
  minefield.value = board
}

const getNewGame = async () => {
  let config = {
    method: 'post',
    url: `http://${host}:${port}/newGame`,
    headers: {
      'Content-Type': 'application/xml',
      'Accept': '*/*',
    }
  };
  await axios(config)
}

let ws = reConnect()

ws.onopen = getBoard
ws.onclose = () => {
  function sleep(number: number) {
    return new Promise((resolve) => {
      setTimeout(resolve, number);
    });
  }

  sleep(1000)
  ws = reConnect()
  getBoard()
}
const getNearbyUnFlaggedCells = (nearbyCells: number[]) => {
  let nearbyUnFlaggedCells = []
  for (let i = 0; i < nearbyCells.length; i++) {
    if (!minefield.value.Cell[nearbyCells[i]].IsFlagged && !minefield.value.Cell[nearbyCells[i]].IsOpen) {
      nearbyUnFlaggedCells.push(nearbyCells[i])
    }
  }
  return nearbyUnFlaggedCells
}

const getNearbyFlaggedCount = (nearbyCells: number[]) => {
  let count = 0
  for (let i = 0; i < nearbyCells.length; i++) {
    if (minefield.value.Cell[nearbyCells[i]].IsFlagged || minefield.value.Cell[nearbyCells[i]].IsOpen && minefield.value.Cell[nearbyCells[i]].IsMine) {
      count++
    }
  }
  return count
}

const doFlag = (index: number) => {
  const cell = minefield.value.Cell[index]
  let nearbyCells = getNearbyCells(index)
  if (cell.IsOpen && !cell.IsMine) {
    let flagCount = getNearbyFlaggedCount(nearbyCells)
    if (flagCount < 1) {
      return
    }
    if (flagCount === cell.Mines) {
      let unFlaggedCells = getNearbyUnFlaggedCells(nearbyCells)
      for (let i of unFlaggedCells) {
        doOpen(i, 0)
      }
    }
  } else {
    cell.IsFlagged = !cell.IsFlagged;
  }
}

const doOpen = (index: number, now: number) => {
  const cell = minefield.value.Cell[index]
  let nearbyCells = getNearbyCells(index)
  if (cell.IsOpen && !cell.IsMine) {
    let flagCount = getNearbyFlaggedCount(nearbyCells)
    if (flagCount < 1) {
      return
    }
    if (flagCount === cell.Mines) {
      let unFlaggedCells = getNearbyUnFlaggedCells(nearbyCells)
      for (let i of unFlaggedCells) {
        doOpen(i, now)
      }
    }
  }
  cell.IsOpen = !cell.IsOpen;
  if (ws && ws.readyState === WebSocket.OPEN) {
    let data = {
      Id: index,
      TimeStamp: now
    }
    ws.send(JSON.stringify(data));
  }
  if (cell.Mines === 0) {
    for (let i = 0; i < nearbyCells.length; i++) {
      doOpen(nearbyCells[i], now)
    }
  }
}

let timer = false
const handleClick = (event: MouseEvent, index: number) => {
  let now = new Date().getTime()
  if (!timer) {
    timer = true
    setInterval(() => {
      let now1 = new Date().getTime()
      timeWatcher.value = msToTime(now1 - startTimeStamp)
    }, 1)
  }
  if (event.button === 2) {
    doFlag(index)
  } else if (event.button === 0) {
    doOpen(index, now)
  }
}

const getImageSrc = (cell: Cell) => {
  let mines = cell.Mines
  if (cell.IsOpen) {
    if (cell.IsMine && cell.IsOpen) {
      return `/src/assets/themes/wom/flag.png`
    }
    if (cell.Mines === 9) {
      return `/src/assets/themes/wom/closed.png`
    }
    return `/src/assets/themes/wom/type${mines}.png`
  }
  if (cell.IsFlagged) {
    return `/src/assets/themes/wom/flag.png`
  }
  return `/src/assets/themes/wom/closed.png`
}

const getNearbyCells = (cell: number) => {
  let nearbyCells = [];                                  //center
  let width = minefield.value.Width;
  let height = minefield.value.Height;
  let x = cell % width;
  let y = Math.floor(cell / width);

  let isNotFirstRow = y > 0; // 不在第一排
  let isNotLastRow = y < height - 1; // 不咋最后一排


  if (isNotFirstRow) nearbyCells.push(cell - width);      //up
  if (isNotLastRow) nearbyCells.push(cell + width);      //down

  if (x > 0) //if cell isn't on first column
  {
    nearbyCells.push(cell - 1);                               //left

    if (isNotFirstRow) nearbyCells.push(cell - width - 1); //up left
    if (isNotLastRow) nearbyCells.push(cell + width - 1); //down left
  }

  if (x < width - 1) //if cell isn't on last column
  {
    nearbyCells.push(cell + 1);                               //right

    if (isNotFirstRow) nearbyCells.push(cell - width + 1); //up right
    if (isNotLastRow) nearbyCells.push(cell + width + 1); //down right
  }

  return nearbyCells;
}

function msToTime(duration: number): string {
  const milliseconds = duration % 1000;
  const seconds = Math.floor((duration / 1000));
  const secondsStr = (seconds < 10) ? "0" + seconds : seconds;

  return `${secondsStr}:${milliseconds}`;
}

ws.onmessage = async (event) => {
  const data: Response = JSON.parse(event.data);
  for (let i = 0; i < data.ChangeCell.Cell.length; i++) {
    minefield.value.Cell[data.ChangeCell.Cell[i].Id] = data.ChangeCell.Cell[i]
  }
  startTimeStamp = data.StartTimeStamp
  if (data.ChangeCell.Result.IsWin) {
    let newGame = confirm(`你赢了！用时：${msToTime(data.TimeStamp - data.StartTimeStamp)}，再来一局？`)
    if (newGame) {
      await getNewGame()
      await getBoard()
    }
  }
}

</script>

<style scoped>
.board {
  position: relative;
  left: 10px;
  top: 10px;
}

.timeWatcher {
  position: absolute;
  top: -100px;
  left: 20px;
  font-size: 48px;
  font-weight: bold;
  color: #00bd7e;
}

.cell {
  position: absolute;
  height: 94px;
  width: 94px;
  justify-content: center;
  align-items: center;
  background-color: lightgray;
  border: 1px solid gray;
  cursor: pointer;
  font-size: 18px;
  font-weight: bold;
  user-select: none;
}

.cell:hover {
  background-color: gray;
  color: white;
}
</style>
