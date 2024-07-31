<template>
  <div class="timeWatcher">{{ timeWatcher }}</div>
  <el-button class="logout-button" @click="logout">退出登录</el-button>
  <el-button :style="{background:flagMode?'#5282b8':'#5c8f4b'}" class="flag-switch-button"
             @click="flagMode = !flagMode">{{ flagMode ? "标记" : "挖开" }}模式
  </el-button>
  <div :style="{
    gridTemplateColumns: `repeat(${minefield.Width}, ${cellSize}px)`,
    gridTemplateRows: `repeat(${minefield.Height}, ${cellSize}px)`
  }" class="board">
    <div v-for="(cell, index) in minefield.Cell" :key="index"
         :style="{backgroundImage: `url(${getImageSrc(cell)})`}"
         class="cell"
         @mousedown="(event) => handleClick(event,index)">
    </div>
  </div>
  <ScoreBoard :scoreBoard="scoreBoard" class="scoreBoard"></ScoreBoard>
  <ScoreBoard :scoreBoard="totalScoreBoard" class="TotalBoard"></ScoreBoard>
  <ScoreTip ref="scoreTip" class="scoreTipParent"></ScoreTip>
</template>

<script lang="ts" setup>
import {ref} from 'vue'
import axios from 'axios'
import {host, port} from "@/utils";
import {ElMessage, ElMessageBox} from "element-plus";
import type {Cell, Minefield, RequestType, Response, ScoreBoard as ScoreBoardType} from "@/types";
import ScoreBoard from "@/components/ScoreBoard.vue";
import ScoreTip from "@/components/ScoreTip.vue";
import {Howl} from 'howler';

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
const userName = localStorage.getItem('userName')
const scoreBoard = ref<ScoreBoardType>({})
const totalScoreBoard = ref<ScoreBoardType>({})
const scoreTip = ref<InstanceType<typeof ScoreTip>>()
const isEnd = ref(false)
const openSound = new Howl({
  src: ['/src/assets/audio/open.mp3'],
  volume: 0.5
})
const flagSound = new Howl({
  src: ['/src/assets/audio/flag.mp3'],
  volume: 0.5
})
const flagMode = ref(false)
const getRank = async () => {
  let config = {
    method: 'post',
    url: `http://${host}:${port}/getRank`,
    headers: {
      'Content-Type': 'application/xml',
      'Accept': '*/*',
    }
  };
  return (await axios(config)).data
}

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
  totalScoreBoard.value = await getRank()
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

const getNearbyFlaggedCount = (nearbyCells: number[]) => {
  let count = 0
  for (let i = 0; i < nearbyCells.length; i++) {
    if (minefield.value.Cell[nearbyCells[i]].IsFlagged || minefield.value.Cell[nearbyCells[i]].IsMine) {
      count++
    }
  }
  return count
}

const doFlag = (index: number, now: number): number[] => {
  const cell = minefield.value.Cell[index]
  let nearbyCells = getNearbyCells(index)
  const openCells: number[] = []
  if (cell.IsOpen && !cell.IsMine) {
    let flagCount = getNearbyFlaggedCount(nearbyCells)
    if (flagCount < 1) {
      return []
    }
    if (flagCount === cell.Mines) {
      for (let i of nearbyCells) {
        if (!minefield.value.Cell[i].IsOpen && !minefield.value.Cell[i].IsFlagged && !minefield.value.Cell[i].IsMine) {
          openCells.push(...doOpen(i))
        }
      }
      return openCells
    } else {
      return []
    }
  } else {
    let data: RequestType = {
      Ids: [index],
      IsFlag: true,
      TimeStamp: now
    }
    flagSound.play()
    ws.send(JSON.stringify(data));
    return []
  }
}

const doOpen = (index: number): number[] => {
  const cell = minefield.value.Cell[index]
  let nearbyCells = getNearbyCells(index)
  const openCells: number[] = []
  if (cell.IsOpen && !cell.IsMine) {
    let flagCount = getNearbyFlaggedCount(nearbyCells)
    if (flagCount < 1) {
      return []
    }
    if (flagCount === cell.Mines) {
      for (let i of nearbyCells) {
        if (!minefield.value.Cell[i].IsOpen && !minefield.value.Cell[i].IsFlagged && !minefield.value.Cell[i].IsMine) {
          openCells.push(...doOpen(i))
        }
      }
    }
    return openCells
  }
  openCells.push(index)
  cell.IsOpen = !cell.IsOpen;
  if (cell.Mines === 0) {
    for (let i = 0; i < nearbyCells.length; i++) {
      openCells.push(...doOpen(nearbyCells[i]))
    }
  }
  return openCells
}

const sendOpenList = async (openList: number[], now: number) => {
  let data: RequestType = {
    Ids: openList,
    IsFlag: false,
    TimeStamp: now
  }
  ws.send(JSON.stringify(data));
}

let timer = false
let intervalFlag: number
const handleClick = (event: MouseEvent, index: number) => {
  let now = new Date().getTime()
  if (!timer) {
    timer = true
    if(isEnd.value){
      return
    }
    intervalFlag = setInterval(() => {
      let now1 = new Date().getTime()
      timeWatcher.value = msToTime(now1 - startTimeStamp)
    }, 1)
  }
  let openCells: number[]
  flagSound.stop()
  openSound.stop()

  const isRightClick = event.button === 2
  const shouldFlag = isRightClick !== flagMode.value

  if (shouldFlag) {
    openCells = doFlag(index, now)
    if (openCells.length > 0) {
      flagSound.play()
    }
  } else {
    openCells = doOpen(index)
    if (openCells.length > 0) {
      openSound.play()
    }
  }

  if (openCells.length > 0) {
    sendOpenList(openCells, now)
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
  if (data.UserName === userName && data.EarnScore) {
    if (scoreTip.value) {
      scoreTip.value.tips(data.EarnScore)
    }
  }
  if (data.NewPlayer && data.UserName != userName) {
    ElMessage({
      type: 'success',
      message: data.UserName + ' 加入了游戏！',
    })
  }
  if (data.PlayerQuit) {
    ElMessage({
      type: 'success',
      message: data.UserName + ' 离开了游戏！',
    })
    return
  }
  for (let i = 0; i < data.ChangeCell.Cell.length; i++) {
    minefield.value.Cell[data.ChangeCell.Cell[i].Id] = data.ChangeCell.Cell[i]
  }
  startTimeStamp = data.StartTimeStamp ?? {name1: 0, name2: 2}
  scoreBoard.value = data.ScoreBoard
  if (data.ChangeCell.Result.IsWin) {
    if (timer) {
      clearInterval(intervalFlag)
      timer = false
    }
    isEnd.value = true
    let confirm = await ElMessageBox.confirm(
        `${decodeURIComponent(data.UserName)}结束了比赛！用时：${msToTime(data.TimeStamp - data.StartTimeStamp)}，再来一局？`,
        'Success',
        {
          confirmButtonText: 'OK',
          cancelButtonText: 'Cancel',
          type: 'success',
        }
    )
    if (confirm === 'confirm') {
      isEnd.value = false
      await getNewGame()
      await getBoard()
    }
  }
}

function logout() {
  localStorage.removeItem('jwt');
  localStorage.removeItem('userId');
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
  position: fixed;
  top: 20px; /* Adjust as needed */
  left: 20px;
  font-size: 26px;
  font-weight: bold;
  color: #00bd7e;
  z-index: 100;
}

.logout-button {
  position: absolute;
  top: 20px; /* Adjust as needed */
  right: 20px; /* Adjust as needed */
  z-index: 100; /* Ensure it's above other content */
}

.flag-switch-button {
  position: absolute;
  top: 20px; /* Adjust as needed */
  right: 120px; /* Adjust as needed */
  z-index: 100; /* Ensure it's above other content */
}

.scoreBoard {
  position: fixed;
  top: 60px; /* Adjust as needed */
  right: 205px; /* Adjust as needed */
  z-index: 100; /* Ensure it's above other content */
}

.TotalBoard {
  position: fixed;
  top: 60px; /* Adjust as needed */
  right: 20px; /* Adjust as needed */
  z-index: 100; /* Ensure it's above other content */
}

.scoreTipParent {
  position: fixed;
  top: 30px; /* Adjust as needed */
  right: 50%; /* Adjust as needed */
  z-index: 999; /* Ensure it's above other content */
}

</style>
