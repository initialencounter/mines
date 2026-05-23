<template>
  <div class="game-area">
    <div class="game-left">
      <ScoreBoard :score-board="scoreBoard" :current-game="true" class="score-panel" />
      <ScoreBoard :score-board="totalScoreBoard" :current-game="false" class="score-panel" />
    </div>
    <div class="game-center">
      <div class="timeWatcher">{{ timeWatcher }}</div>
      <div
        :style="{
          gridTemplateColumns: `repeat(${minefield.Width}, ${cellSize}px)`,
          gridTemplateRows: `repeat(${minefield.Height}, ${cellSize}px)`,
        }"
        class="board"
      >
        <div
          v-for="(cell, index) in minefield.Cell"
          :key="index"
          :style="{ backgroundImage: `url(${getImageSrc(cell)})` }"
          class="cell"
          @mousedown="(event) => handleClick(event, index)"
        ></div>
      </div>
      <ScoreTip ref="scoreTip" class="scoreTipParent" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import axios from "axios";
import { host, port } from "@/utils";
import { ElMessage, ElMessageBox } from "element-plus";
import type {
  Cell,
  Minefield,
  RequestType,
  Response,
  ScoreBoard as ScoreBoardType,
} from "@/types";
import ScoreBoard from "@/components/ScoreBoard.vue";
import ScoreTip from "@/components/ScoreTip.vue";
import { Howl } from "howler";

const props = defineProps<{
  flagMode: boolean;
}>();

const cellSize = 24;
const minefield = ref<Minefield>({
  Width: 5,
  Height: 4,
  Cells: 20,
  Mines: 5,
  Cell: [],
  First: false,
  StartTimeStamp: 0,
});

const timeWatcher = ref("00:0");
let startTimeStamp = 0;
document.oncontextmenu = () => false;
const userId = localStorage.getItem("userId");
const token = (localStorage.getItem("jwt") ?? "").replace("20240704", "");
const userName = localStorage.getItem("userName");
const scoreBoard = ref<ScoreBoardType>({});
const totalScoreBoard = ref<ScoreBoardType>({});
const scoreTip = ref<InstanceType<typeof ScoreTip>>();
const isEnd = ref(false);
const openSound = new Howl({
  src: ["/src/assets/audio/open.mp3"],
  volume: 0.5,
});
const flagSound = new Howl({
  src: ["/src/assets/audio/flag.mp3"],
  volume: 0.5,
});

const getRank = async () => {
  let config = {
    method: "post",
    url: `//${host}:${port}/getRank`,
    headers: {
      "Content-Type": "application/xml",
      Accept: "*/*",
    },
  };
  return (await axios(config)).data;
};

const reConnect = () => {
  return new WebSocket(`${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${host}:${port}/ws/${userId}?token=${token}`);
};

const getBoard = async () => {
  let config = {
    method: "post",
    url: `//${host}:${port}/getMinefield`,
    headers: {
      "Content-Type": "application/xml",
      Accept: "*/*",
    },
  };
  minefield.value = (await axios(config)).data;
  totalScoreBoard.value = await getRank();
};

const getNewGame = async () => {
  let config = {
    method: "post",
    url: `//${host}:${port}/newGame`,
    headers: {
      "Content-Type": "application/xml",
      Accept: "*/*",
    },
  };
  await axios(config);
};

let ws = reConnect();

ws.onopen = getBoard;
ws.onclose = () => {
  function sleep(number: number) {
    return new Promise((resolve) => {
      setTimeout(resolve, number);
    });
  }

  sleep(1000);
  ws = reConnect();
  getBoard();
};

const getNearbyFlaggedCount = (nearbyCells: number[]) => {
  let count = 0;
  for (let i = 0; i < nearbyCells.length; i++) {
    if (
      minefield.value.Cell[nearbyCells[i]].IsFlagged ||
      minefield.value.Cell[nearbyCells[i]].IsMine
    ) {
      count++;
    }
  }
  return count;
};

const doFlag = (index: number, now: number): number[] => {
  const cell = minefield.value.Cell[index];
  let nearbyCells = getNearbyCells(index);
  const openCells: number[] = [];
  if (cell.IsOpen && !cell.IsMine) {
    let flagCount = getNearbyFlaggedCount(nearbyCells);
    if (flagCount < 1) {
      return [];
    }
    if (flagCount === cell.Mines) {
      for (let i of nearbyCells) {
        if (
          !minefield.value.Cell[i].IsOpen &&
          !minefield.value.Cell[i].IsFlagged &&
          !minefield.value.Cell[i].IsMine
        ) {
          openCells.push(...doOpen(i));
        }
      }
      return openCells;
    } else {
      return [];
    }
  } else {
    let data: RequestType = {
      Ids: [index],
      IsFlag: true,
      TimeStamp: now,
    };
    flagSound.play();
    ws.send(JSON.stringify(data));
    return [];
  }
};

const doOpen = (index: number): number[] => {
  const cell = minefield.value.Cell[index];
  let nearbyCells = getNearbyCells(index);
  const openCells: number[] = [];
  if (cell.IsOpen && !cell.IsMine) {
    let flagCount = getNearbyFlaggedCount(nearbyCells);
    if (flagCount < 1) {
      return [];
    }
    if (flagCount === cell.Mines) {
      for (let i of nearbyCells) {
        if (
          !minefield.value.Cell[i].IsOpen &&
          !minefield.value.Cell[i].IsFlagged &&
          !minefield.value.Cell[i].IsMine
        ) {
          openCells.push(...doOpen(i));
        }
      }
    }
    return openCells;
  }
  openCells.push(index);
  cell.IsOpen = !cell.IsOpen;
  if (cell.Mines === 0) {
    for (let i = 0; i < nearbyCells.length; i++) {
      openCells.push(...doOpen(nearbyCells[i]));
    }
  }
  return openCells;
};

const sendOpenList = async (openList: number[], now: number) => {
  let data: RequestType = {
    Ids: openList,
    IsFlag: false,
    TimeStamp: now,
  };
  ws.send(JSON.stringify(data));
};

let timer = false;
let intervalFlag: number;
const handleClick = (event: MouseEvent, index: number) => {
  if (event.button === 1) {
    reset();
    return
  }
  let now = new Date().getTime();
  if (!timer) {
    timer = true;
    if (isEnd.value) {
      return;
    }
    intervalFlag = setInterval(() => {
      let now1 = new Date().getTime();
      timeWatcher.value = msToTime(now1 - startTimeStamp);
    }, 1);
  }
  let openCells: number[];
  flagSound.stop();
  openSound.stop();

  const isRightClick = event.button === 2;
  const shouldFlag = isRightClick !== props.flagMode;

  if (shouldFlag) {
    openCells = doFlag(index, now);
    if (openCells.length > 0) {
      flagSound.play();
    }
  } else {
    openCells = doOpen(index);
    if (openCells.length > 0) {
      openSound.play();
    }
  }

  if (openCells.length > 0) {
    sendOpenList(openCells, now);
  }
};

const getImageSrc = (cell: Cell) => {
  let mines = cell.Mines;
  if (cell.IsOpen) {
    if (cell.IsMine && cell.IsOpen) {
      return `/src/assets/themes/wom/flag.png`;
    }
    if (cell.Mines === 9) {
      return `/src/assets/themes/wom/closed.png`;
    }
    return `/src/assets/themes/wom/type${mines}.png`;
  }
  if (cell.IsFlagged) {
    return `/src/assets/themes/wom/flag.png`;
  }
  return `/src/assets/themes/wom/closed.png`;
};

const getNearbyCells = (cell: number) => {
  let nearbyCells = [];
  let width = minefield.value.Width;
  let height = minefield.value.Height;
  let x = cell % width;
  let y = Math.floor(cell / width);

  let isNotFirstRow = y > 0;
  let isNotLastRow = y < height - 1;

  if (isNotFirstRow) nearbyCells.push(cell - width);
  if (isNotLastRow) nearbyCells.push(cell + width);

  if (x > 0) {
    nearbyCells.push(cell - 1);
    if (isNotFirstRow) nearbyCells.push(cell - width - 1);
    if (isNotLastRow) nearbyCells.push(cell + width - 1);
  }

  if (x < width - 1) {
    nearbyCells.push(cell + 1);
    if (isNotFirstRow) nearbyCells.push(cell - width + 1);
    if (isNotLastRow) nearbyCells.push(cell + width + 1);
  }

  return nearbyCells;
};

function msToTime(duration: number): string {
  const milliseconds = duration % 10;
  const seconds = Math.floor(duration / 1000);
  const secondsStr = seconds < 10 ? "0" + seconds : seconds;

  return `${secondsStr}:${milliseconds}`;
}

ws.onmessage = async (event) => {
  const data: Response = JSON.parse(event.data);
  if (data.UserName === userName && data.EarnScore) {
    if (scoreTip.value) {
      scoreTip.value.tips(data.EarnScore);
    }
  }
  if (data.NewPlayer && data.UserName != userName) {
    ElMessage({
      type: "success",
      message: data.UserName + " 加入了游戏！",
    });
  }
  if (data.PlayerQuit) {
    ElMessage({
      type: "success",
      message: data.UserName + " 离开了游戏！",
    });
    return;
  }
  for (let i = 0; i < data.ChangeCell.Cell.length; i++) {
    minefield.value.Cell[data.ChangeCell.Cell[i].Id] = data.ChangeCell.Cell[i];
  }
  startTimeStamp = data.StartTimeStamp ?? { name1: 0, name2: 2 };
  scoreBoard.value = data.ScoreBoard;
  if (data.ChangeCell.Result.IsWin) {
    if (timer) {
      clearInterval(intervalFlag);
      timer = false;
    }
    isEnd.value = true;
    let confirm = await ElMessageBox.confirm(
      `${decodeURIComponent(data.UserName)}结束了比赛！用时：${msToTime(
        data.TimeStamp - data.StartTimeStamp
      )}，再来一局？`,
      "Success",
      {
        confirmButtonText: "OK",
        cancelButtonText: "Cancel",
        type: "success",
      }
    );
    if (confirm === "confirm") {
      reset();
    }
  }
};

async function reset() {
  isEnd.value = false;
  await getNewGame();
  await getBoard();
}

defineExpose({ reset });
</script>

<style scoped>
.game-area {
  display: flex;
  justify-content: center;
  align-items: flex-start;
  height: 100%;
  padding: 20px;
  gap: 20px;
}

.game-left {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex-shrink: 0;
}

.game-center {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.board {
  display: grid;
}

.cell {
  background-size: cover;
}

.timeWatcher {
  font-size: 26px;
  font-weight: bold;
  color: #00bd7e;
}
</style>
