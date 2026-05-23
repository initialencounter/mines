<template>
  <div class="game-area">
    <div class="game-left">
      <ScoreBoard :score-board="scoreBoard" :current-game="true" class="score-panel" />
      <ScoreBoard :score-board="totalScoreBoard" :current-game="false" class="score-panel" />
    </div>
    <div class="game-center">
      <PropBar
        :prop-state="propBarState"
        :active-prop="activeProp"
        @update:active-prop="activeProp = $event"
      />
      <div class="timeWatcher">用时:{{ timeWatcher }}</div>
      <div class="board-container">
        <div
          :style="{
            gridTemplateColumns: `repeat(${minefield.Width}, ${cellSize}px)`,
            gridTemplateRows: `repeat(${minefield.Height}, ${cellSize}px)`,
          }"
          class="board"
        >
          <!-- Zone overlays -->
          <div
            v-for="(zone, zi) in zones"
            :key="'z'+zi"
            class="zone-overlay"
            :class="zone.Type"
            :style="zoneStyle(zone)"
          />
          <!-- Cells -->
          <div
            v-for="(cell, index) in minefield.Cell"
            :key="index"
            class="cell-wrapper"
          >
            <div
              :style="{ backgroundImage: `url(${getImageSrc(cell)})` }"
              class="cell"
              :class="{
                'detector-mine': detectorMineCells.has(index),
                'detector-safe': detectorSafeCells.has(index),
              }"
              @mousedown="(event) => handleClick(event, index)"
            ></div>
          </div>
        </div>
      </div>
      <ScoreTip ref="scoreTip" class="scoreTipParent" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted, ref } from "vue";
import axios from "axios";
import { host, port } from "@/utils";
import { ElMessage, ElMessageBox } from "element-plus";
import type {
  Cell,
  Minefield,
  RequestType,
  Response,
  ScoreBoard as ScoreBoardType,
  Zone as ZoneType,
  PropBarUpdate,
  DetectorResult,
  PropEffectInfo,
  ShieldProtect,
} from "@/types";
import ScoreBoard from "@/components/ScoreBoard.vue";
import ScoreTip from "@/components/ScoreTip.vue";
import PropBar from "@/components/PropBar.vue";
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

// Zone state
const zones = ref<ZoneType[]>([]);

// Detector highlight state
const detectorMineCells = ref<Set<number>>(new Set());
const detectorSafeCells = ref<Set<number>>(new Set());
const detectorTimer = ref<number | null>(null);

// Prop state
const activeProp = ref<number | null>(null);
const propBarState = ref<PropBarUpdate>({
  Inventory: [],
  DoubleScoreActive: false,
  DoubleScoreRemaining: 0,
  ShieldCount: 0,
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

// Keyboard shortcuts for props
function hasProp(propId: number): boolean {
  return propBarState.value.Inventory.some((s) => s.PropID === propId && s.Count > 0);
}
function onKeydown(e: KeyboardEvent) {
  const tag = (e.target as HTMLElement)?.tagName;
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
  if (e.metaKey || e.ctrlKey || e.altKey) return;

  const key = e.key.toUpperCase();
  if (key === "D" && hasProp(101)) {
    e.preventDefault();
    activeProp.value = activeProp.value === 101 ? null : 101;
  }
  if (key === "X" && hasProp(102)) {
    e.preventDefault();
    activeProp.value = activeProp.value === 102 ? null : 102;
  }
}
onMounted(() => {
  window.addEventListener("keydown", onKeydown);
});
onUnmounted(() => {
  window.removeEventListener("keydown", onKeydown);
  if (detectorTimer.value) clearTimeout(detectorTimer.value);
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
  if (minefield.value.Zones) {
    zones.value = minefield.value.Zones;
  }
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

const isInNoFlagZone = (index: number): boolean => {
  const w = minefield.value.Width;
  const row = Math.floor(index / w);
  const col = index % w;
  return zones.value.some(
    (z) =>
      z.Type === "noFlag" &&
      row >= z.StartRow &&
      row <= z.EndRow &&
      col >= z.StartCol &&
      col <= z.EndCol
  );
};

const isInDoubleScoreZone = (index: number): boolean => {
  const w = minefield.value.Width;
  const row = Math.floor(index / w);
  const col = index % w;
  return zones.value.some(
    (z) =>
      z.Type === "doubleScore" &&
      row >= z.StartRow &&
      row <= z.EndRow &&
      col >= z.StartCol &&
      col <= z.EndCol
  );
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

  // Prop mode: handle prop usage
  if (activeProp.value !== null) {
    const propId = activeProp.value;
    activeProp.value = null;
    const actionType = propId === 101 ? "useDetector" : "useXJBD";
    const data: RequestType = {
      Ids: [],
      IsFlag: false,
      TimeStamp: new Date().getTime(),
      ActionType: actionType,
      PropID: propId,
      TargetCell: index,
    };
    ws.send(JSON.stringify(data));
    return;
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
    // Check no-flag zone
    if (isInNoFlagZone(index)) {
      ElMessage.warning("该区域禁止标记");
      return;
    }
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

function applyDetectorHighlights(result: DetectorResult) {
  detectorMineCells.value = new Set(result.MineCells);
  detectorSafeCells.value = new Set(result.SafeCells);
  if (detectorTimer.value) clearTimeout(detectorTimer.value);
  detectorTimer.value = window.setTimeout(() => {
    detectorMineCells.value = new Set();
    detectorSafeCells.value = new Set();
  }, 10000);
}

function zoneStyle(zone: ZoneType) {
  const left = zone.StartCol * cellSize;
  const top = zone.StartRow * cellSize;
  const width = (zone.EndCol - zone.StartCol + 1) * cellSize;
  const height = (zone.EndRow - zone.StartRow + 1) * cellSize;
  return {
    left: `${left}px`,
    top: `${top}px`,
    width: `${width}px`,
    height: `${height}px`,
  };
}

ws.onmessage = async (event) => {
  const data: Response = JSON.parse(event.data);

  // Handle personalized messages
  if (data.MessageType === "detectorResult" && data.DetectorResult) {
    applyDetectorHighlights(data.DetectorResult);
    if (data.PropBarUpdate) {
      propBarState.value = data.PropBarUpdate;
    }
    ElMessage.success("探测仪已使用");
    return;
  }

  if (data.MessageType === "personal") {
    if (data.PropBarUpdate) {
      propBarState.value = data.PropBarUpdate;
    }
    if (data.PropDrop) {
      const names: Record<number, string> = { 101: "探测仪", 102: "雷之奥义", 1001: "双倍积分", 1002: "护盾" };
      ElMessage.success(`获得道具: ${names[data.PropDrop.PropID] || data.PropDrop.PropName} (+${data.PropDrop.Count})`);
    }
    if (data.ShieldProtect) {
      ElMessage.warning(`护盾保护！剩余 ${data.ShieldProtect.ShieldCount} 个`);
    }
    return;
  }

  if (data.MessageType === "propEffect" && data.PropEffect) {
    const names: Record<number, string> = { 101: "探测仪", 102: "雷之奥义" };
    ElMessage.info(`${data.UserName} 使用了${names[data.PropEffect.PropID] || "道具"}`);
    // Fall through to process ChangeCell if present
  }

  // Zone info (first connect)
  if (data.ZoneInfo) {
    zones.value = data.ZoneInfo;
  }

  // Score tip animation
  if (data.UserName === userName && data.EarnScore) {
    if (scoreTip.value) {
      const isDouble = propBarState.value.DoubleScoreActive;
      scoreTip.value.tips(data.EarnScore, isDouble);
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
  zones.value = [];
  propBarState.value = {
    Inventory: [],
    DoubleScoreActive: false,
    DoubleScoreRemaining: 0,
    ShieldCount: 0,
  };
  activeProp.value = null;
  detectorMineCells.value = new Set();
  detectorSafeCells.value = new Set();
  if (detectorTimer.value) {
    clearTimeout(detectorTimer.value);
    detectorTimer.value = null;
  }
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

.board-container {
  position: relative;
}

.board {
  display: grid;
  position: relative;
}

/* Zone overlays */
.zone-overlay {
  position: absolute;
  pointer-events: none;
  z-index: 5;
  border-radius: 4px;
  box-shadow: inset 0 0 16px rgba(0, 0, 0, 0.15);
}
.zone-overlay.doubleScore {
  background: rgba(255, 200, 50, 0.35);
  border: 2px solid rgba(255, 200, 50, 0.7);
  animation: zone-glow-yellow 2s ease-in-out infinite alternate;
}
.zone-overlay.noFlag {
  background: rgba(255, 60, 50, 0.4);
  border: 2px solid rgba(255, 60, 50, 0.75);
  animation: zone-glow-red 2s ease-in-out infinite alternate;
}
@keyframes zone-glow-yellow {
  from {
    background: rgba(255, 200, 50, 0.25);
    border-color: rgba(255, 200, 50, 0.5);
  }
  to {
    background: rgba(255, 200, 50, 0.45);
    border-color: rgba(255, 200, 50, 0.85);
  }
}
@keyframes zone-glow-red {
  from {
    background: rgba(255, 60, 50, 0.3);
    border-color: rgba(255, 60, 50, 0.55);
  }
  to {
    background: rgba(255, 60, 50, 0.5);
    border-color: rgba(255, 60, 50, 0.9);
  }
}

.cell-wrapper {
  position: relative;
  width: 24px;
  height: 24px;
}

.cell {
  width: 100%;
  height: 100%;
  background-size: cover;
}

/* Detector highlights */
.cell.detector-mine::after {
  content: '';
  position: absolute;
  inset: 0;
  background: rgba(255, 0, 0, 0.5);
  pointer-events: none;
  z-index: 7;
  animation: detector-pulse 0.8s ease-in-out infinite alternate;
}
.cell.detector-safe::after {
  content: '';
  position: absolute;
  inset: 0;
  background: rgba(0, 230, 120, 0.35);
  pointer-events: none;
  z-index: 7;
  animation: detector-pulse 0.8s ease-in-out infinite alternate;
}
@keyframes detector-pulse {
  from { opacity: 0.6; }
  to { opacity: 1.0; }
}

.timeWatcher {
  font-size: 26px;
  font-weight: bold;
  color: #00bd7e;
}
</style>
