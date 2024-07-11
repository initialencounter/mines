<template>
  <div class="timeWatcher">{{ timeWatcher }}</div>
  <el-button class="logout-button" @click="logout">退出登录</el-button>
  <div class="board" :style="{
    gridTemplateColumns: `repeat(${minefield.Width}, ${cellSize}px)`,
    gridTemplateRows: `repeat(${minefield.Height}, ${cellSize}px)`
  }">
    <div v-for="(cell, index) in minefield.Cell" :key="index"
         :style="{backgroundImage: `url(${getImageSrc(cell)})`}"
         class="cell"
         @mousedown="(event) => handleClick(event,index)">
    </div>
  </div>
</template>

<script lang="ts" setup>
import {h, ref} from 'vue'
import axios from 'axios'
import {host, port} from "@/utils";
import {ElMessage, ElMessageBox, ElNotification} from "element-plus";
import type {Cell, Minefield, Response} from "@/types";

const cellSize = 24
const minefield = ref<Minefield>({
  Width: 5,
  Height: 4,
  Cells: 20,
  Mines: 5,
  Cell: [],
  First: false,
  StartTimeStamp: 0,
})

const showLogin = defineModel<boolean>({required: true})
const timeWatcher = ref('00:000')
let startTimeStamp = 0
document.oncontextmenu = () => false;
const userId = localStorage.getItem('userId')
const token = (localStorage.getItem('jwt') ?? '').replace('20240704', '')

const reConnect = () => {
  return new WebSocket(`ws://${host}:${port}/ws/${userId}?token=${token}`);
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
  minefield.value = (await axios(config)).data
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
let intervalFlag: number
const handleClick = (event: MouseEvent, index: number) => {
  let now = new Date().getTime()
  if (!timer) {
    timer = true
    intervalFlag = setInterval(() => {
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
  let isNotLastRow = y < height - 1; // 不在最后一排


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
    if (data.NewPlayer) {
      ElNotification({
        title: 'Success',
        message: h('i', { style: 'color: teal' }, data.UserId + ' 加入了游戏！'),
      })
    }
    if (data.PlayerQuit) {
      ElNotification({
        title: 'Success',
        message: h('i', { style: 'color: teal' }, data.UserId + ' 离开了游戏！'),
      })
      return
    }
  for (let i = 0; i < data.ChangeCell.Cell.length; i++) {
    minefield.value.Cell[data.ChangeCell.Cell[i].Id] = data.ChangeCell.Cell[i]
  }
  startTimeStamp = data.StartTimeStamp
  if (data.ChangeCell.Result.IsWin) {
    if(timer){
      clearInterval(intervalFlag)
      timer = false
    }
    ElMessageBox.confirm(
        `${decodeURIComponent(data.UserId)}结束了比赛！用时：${msToTime(data.TimeStamp - data.StartTimeStamp)}，再来一局？`,
        'Success',
        {
          confirmButtonText: 'OK',
          cancelButtonText: 'Cancel',
          type: 'success',
        }
    )
        .then(async () => {
          await getNewGame()
          await getBoard()
        })
        .catch(() => {
          ElMessage({
            type: 'info',
            message: '取消了再来一局！',
          })
        })

  }
}

function logout() {
  localStorage.removeItem('jwt');
  localStorage.removeItem('userId');
  // Redirect to login page or refresh the page
  showLogin.value = true;
}

</script>

<style scoped>
.board {
  display: grid;
  position: absolute;
  top: 80px;
  left: 50px;
}

.cell {
  background-size: cover;
}

.cell:hover {
  background-color: gray;
  color: white;
}

.timeWatcher {
  position: absolute;
  top: 20px; /* Adjust as needed */
  left: 20px;
  font-size: 26px;
  font-weight: bold;
  color: #00bd7e;
}

.logout-button {
  position: absolute;
  top: 20px; /* Adjust as needed */
  right: 20px; /* Adjust as needed */
  z-index: 100; /* Ensure it's above other content */
}
</style>
