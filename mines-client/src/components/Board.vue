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
        <!-- CD 冷却遮罩 -->
        <div v-if="isBlocked" class="cd-overlay-dialog">
          <div class="cd-overlay-card">
            <span class="cd-overlay-title">冷却中</span>
            <div class="cd-progress-bar">
              <div class="cd-progress-fill" :style="{ width: `${cdPercent}%` }" />
            </div>
            <span class="cd-overlay-time">{{ cdRemaining.toFixed(1) }}s</span>
          </div>
        </div>
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
            v-memo="[cell.IsOpen, cell.IsFlagged, cell.IsMine, cell.Mines, overlayVersion]"
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
            />
            <div v-if="noFlagZoneSet.has(index)" class="no-flag-zone-overlay" />
            <div v-if="doubleScoreZoneSet.has(index)" class="double-score-zone-overlay" />
          </div>
        </div>
      </div>
      <ScoreTip ref="scoreTip" class="scoreTipParent" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref } from "vue";
import axios from "axios";
import { host, port } from "@/utils";
import { ElMessage, ElMessageBox } from "element-plus";
import type {
  Cell,
  Minefield,
  RequestType,
  Response,
  ScoreBoard as ScoreBoardType,
  PropBarUpdate,
  DetectorResult,
} from "@/types";
import ScoreBoard from "@/components/ScoreBoard.vue";
import ScoreTip from "@/components/ScoreTip.vue";
import PropBar from "@/components/PropBar.vue";
import { Howl } from "howler";

const props = defineProps<{
  flagMode: boolean;
}>();

// ========== 常量 ==========
const cellSize = 24;
const COOLDOWN_PER_ERROR = 0.5;

// ========== 音效 ==========
const openSound = new Howl({ src: ["/src/assets/audio/open.mp3"], volume: 0.5 });
const flagSound = new Howl({ src: ["/src/assets/audio/flag.mp3"], volume: 0.5 });
const boomSound = new Howl({ src: ["/src/assets/audio/boom.mp3"], volume: 0.2 });

// ========== 响应式状态 ==========
const minefield = ref<Minefield>({
  Width: 5,
  Height: 4,
  Cells: 20,
  Mines: 5,
  Cell: [],
  First: false,
  StartTimeStamp: 0,
});

// Zone state — 预计算 Set，模板 O(1) 查找
const noFlagZoneSet = ref<Set<number>>(new Set());
const doubleScoreZoneSet = ref<Set<number>>(new Set());
const overlayVersion = ref(0);

function rebuildZoneSets() {
  const w = minefield.value.Width;
  const zones = minefield.value.Zones;
  const nfSet = new Set<number>();
  const dsSet = new Set<number>();
  if (zones) {
    for (const z of zones) {
      for (let r = z.StartRow; r <= z.EndRow; r++) {
        for (let c = z.StartCol; c <= z.EndCol; c++) {
          const idx = r * w + c;
          if (z.Type === "noFlag") nfSet.add(idx);
          else if (z.Type === "doubleScore") dsSet.add(idx);
        }
      }
    }
  }
  noFlagZoneSet.value = nfSet;
  doubleScoreZoneSet.value = dsSet;
  overlayVersion.value++;
}

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

// CD 冷却系统
const cdEndTime = ref(0);
const cdRemaining = ref(0);
const cdTotal = ref(0);
const isBlocked = computed(() => cdRemaining.value > 0);
const cdPercent = computed(() =>
  cdTotal.value > 0 ? Math.max(0, (cdRemaining.value / cdTotal.value) * 100) : 0,
);
const cdTimer = ref<number | null>(null);
const errorCount = ref(0);

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

// ========== 邻近格子缓存 ==========
let nearbyCache: number[][] = [];
function buildNearbyCache() {
  const w = minefield.value.Width;
  const h = minefield.value.Height;
  const total = w * h;
  nearbyCache = Array.from({ length: total });
  for (let i = 0; i < total; i++) {
    const x = i % w;
    const y = Math.floor(i / w);
    const nearby: number[] = [];
    if (y > 0) nearby.push(i - w);
    if (y < h - 1) nearby.push(i + w);
    if (x > 0) {
      nearby.push(i - 1);
      if (y > 0) nearby.push(i - w - 1);
      if (y < h - 1) nearby.push(i + w - 1);
    }
    if (x < w - 1) {
      nearby.push(i + 1);
      if (y > 0) nearby.push(i - w + 1);
      if (y < h - 1) nearby.push(i + w + 1);
    }
    nearbyCache[i] = nearby;
  }
}

function getNearbyCells(cell: number): number[] {
  return nearbyCache[cell] || [];
}

// ========== 键盘快捷键 ==========
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
  getBoard();
  startCdTimer();
});
onUnmounted(() => {
  window.removeEventListener("keydown", onKeydown);
  if (detectorTimer.value) clearTimeout(detectorTimer.value);
  if (cdTimer.value) clearInterval(cdTimer.value);
  if (intervalFlag) clearInterval(intervalFlag);
});

// ========== HTTP API ==========
const getRank = async () => {
  const config = {
    method: "post",
    url: `//${host}:${port}/getRank`,
    headers: { "Content-Type": "application/xml", Accept: "*/*" },
  };
  return (await axios(config)).data;
};

const getBoard = async () => {
  const config = {
    method: "post",
    url: `//${host}:${port}/getMinefield`,
    headers: { "Content-Type": "application/xml", Accept: "*/*" },
  };
  minefield.value = (await axios(config)).data;
  if (minefield.value.Zones) {
    rebuildZoneSets();
  }
  buildNearbyCache();
  totalScoreBoard.value = await getRank();
};

const getNewGame = async () => {
  const config = {
    method: "post",
    url: `//${host}:${port}/newGame`,
    headers: { "Content-Type": "application/xml", Accept: "*/*" },
  };
  await axios(config);
};

// ========== WebSocket ==========
let ws: WebSocket;
let reconnectTimer: number | null = null;

function setupWebSocket() {
  ws = new WebSocket(
    `${location.protocol === "https:" ? "wss:" : "ws:"}//${host}:${port}/ws/${userId}?token=${token}`,
  );

  ws.onopen = () => {
    getBoard();
  };

  ws.onclose = () => {
    // 延迟重连并重新绑定全部 handler
    if (reconnectTimer) clearTimeout(reconnectTimer);
    reconnectTimer = window.setTimeout(() => {
      setupWebSocket();
    }, 2000);
  };

  ws.onmessage = onWsMessage;
}

setupWebSocket();

// ========== WebSocket 消息处理 ==========
async function onWsMessage(event: MessageEvent) {
  const data: Response = JSON.parse(event.data);

  // 个性化消息
  if (data.MessageType === "detectorResult" && data.DetectorResult) {
    applyDetectorHighlights(data.DetectorResult);
    if (data.PropBarUpdate) propBarState.value = data.PropBarUpdate;
    ElMessage.success("探测仪已使用");
    return;
  }

  if (data.MessageType === "personal") {
    if (data.PropBarUpdate) propBarState.value = data.PropBarUpdate;
    if (data.PropDrop) {
      const names: Record<number, string> = {
        101: "探测仪", 102: "雷之奥义", 1001: "双倍积分", 1002: "护盾",
      };
      ElMessage.success(
        `获得道具: ${names[data.PropDrop.PropID] || data.PropDrop.PropName} (+${data.PropDrop.Count})`,
      );
    }
    if (data.ShieldProtect) {
      ElMessage.warning(`护盾保护！剩余 ${data.ShieldProtect.ShieldCount} 个`);
    }
    return;
  }

  if (data.MessageType === "propEffect" && data.PropEffect) {
    const names: Record<number, string> = { 101: "探测仪", 102: "雷之奥义" };
    ElMessage.info(`${data.UserName} 使用了${names[data.PropEffect.PropID] || "道具"}`);
  }

  // Zone info
  if (data.ZoneInfo) {
    minefield.value.Zones = data.ZoneInfo;
    rebuildZoneSets();
  }

  // Score tip animation
  if (data.UserName === userName && data.EarnScore) {
    if (scoreTip.value) {
      const isDouble = propBarState.value.DoubleScoreActive;
      scoreTip.value.tips(data.EarnScore, isDouble);
    }
  }
  if (data.NewPlayer && data.UserName !== userName) {
    ElMessage({ type: "success", message: data.UserName + " 加入了游戏！" });
  }
  if (data.PlayerQuit) {
    ElMessage({ type: "success", message: data.UserName + " 离开了游戏！" });
    return;
  }

  // 更新 Cell
  for (let i = 0; i < data.ChangeCell.Cell.length; i++) {
    minefield.value.Cell[data.ChangeCell.Cell[i].Id] = data.ChangeCell.Cell[i];
  }
  startTimeStamp = data.StartTimeStamp || 0;
  scoreBoard.value = data.ScoreBoard;

  // 结算
  if (data.ChangeCell.Result.IsWin) {
    if (timerRunning) {
      clearInterval(intervalFlag);
      timerRunning = false;
    }
    isEnd.value = true;
    let confirm = await ElMessageBox.confirm(
      `${decodeURIComponent(data.UserName)}结束了比赛！用时：${msToTime(
        data.TimeStamp - data.StartTimeStamp,
      )}，再来一局？`,
      "Success",
      {
        confirmButtonText: "OK",
        cancelButtonText: "Cancel",
        type: "success",
      },
    );
    if (confirm === "confirm") {
      reset();
    }
  }
}

// ========== 计时器 ==========
let timerRunning = false;
let intervalFlag: number;

function startTimer() {
  if (!timerRunning) {
    timerRunning = true;
    intervalFlag = window.setInterval(() => {
      timeWatcher.value = msToTime(Date.now() - startTimeStamp);
    }, 100);
  }
}

// ========== 冷却系统 ==========
function applyCooldown() {
  const now = Date.now();
  const penalty = 2.5 + (errorCount.value + 1) * COOLDOWN_PER_ERROR;
  cdTotal.value = penalty;
  cdEndTime.value = Math.max(cdEndTime.value, now) + penalty * 1000;
  cdRemaining.value = (cdEndTime.value - now) / 1000;
  errorCount.value++;
  startCdTimer();
}

function startCdTimer() {
  if (cdTimer.value) return;
  cdTimer.value = window.setInterval(() => {
    const remaining = Math.max(0, (cdEndTime.value - Date.now()) / 1000);
    cdRemaining.value = remaining;
    if (remaining <= 0) {
      cdRemaining.value = 0;
      if (cdTimer.value) {
        clearInterval(cdTimer.value);
        cdTimer.value = null;
      }
    }
  }, 100);
}

// ========== 点击处理 ==========
const handleClick = (event: MouseEvent, index: number) => {
  if (event.button === 1) {
    reset();
    return;
  }

  if (isEnd.value) return;

  // 冷却中
  if (isBlocked.value) {
    ElMessage.warning(`冷却中，${cdRemaining.value.toFixed(1)}s 后恢复`);
    return;
  }

  // Prop mode
  if (activeProp.value !== null) {
    const propId = activeProp.value;
    activeProp.value = null;
    const actionType = propId === 101 ? "useDetector" : "useXJBD";
    const data: RequestType = {
      Ids: [],
      IsFlag: false,
      TimeStamp: Date.now(),
      ActionType: actionType,
      PropID: propId,
      TargetCell: index,
    };
    ws.send(JSON.stringify(data));
    return;
  }

  const now = Date.now();
  startTimer();

  flagSound.stop();
  openSound.stop();

  const isRightClick = event.button === 2;
  const shouldFlag = isRightClick !== props.flagMode;

  if (shouldFlag) {
    if (noFlagZoneSet.value.has(index)) {
      ElMessage.warning("该区域禁止标记");
      return;
    }
    doFlag(index, now);
  } else {
    doOpen(index, now);
  }
};

// ========== 操作逻辑 ==========
function doFlag(index: number, now: number) {
  const cell = minefield.value.Cell[index];
  const nearbyCells = getNearbyCells(index);

  if (cell.IsOpen && !cell.IsMine) {
    // 双击已打开格子 → 快速打开
    const flagCount = getNearbyFlaggedCount(nearbyCells);
    if (flagCount < 1 || flagCount !== cell.Mines) return;
    const openCells: number[] = [];
    for (const i of nearbyCells) {
      if (
        !minefield.value.Cell[i].IsOpen &&
        !minefield.value.Cell[i].IsFlagged &&
        !minefield.value.Cell[i].IsMine
      ) {
        openCells.push(...collectOpen(i));
      }
    }
    if (openCells.length > 0) {
      openSound.play();
      sendOpenList(openCells, now);
    }
    return;
  }

  if (cell.IsOpen || cell.IsFlagged) return;

  flagSound.play();

  // 标记错误 → 冷却
  if (!cell.IsMine) {
    if (propBarState.value.ShieldCount > 0) {
      // 护盾保护
      ElMessage.warning(`标记错误！护盾保护 (剩余 ${propBarState.value.ShieldCount - 1} 个)`);
    } else {
      boomSound.play();
      applyCooldown();
      ElMessage({ message: "标记错误", type: "info", duration: 800 });
    }
  }

  const data: RequestType = { Ids: [index], IsFlag: true, TimeStamp: now };
  ws.send(JSON.stringify(data));
}

function doOpen(index: number, now: number) {
  const cell = minefield.value.Cell[index];
  const nearbyCells = getNearbyCells(index);

  if (cell.IsOpen && !cell.IsMine) {
    // 双击已打开格子 → 快速打开
    const flagCount = getNearbyFlaggedCount(nearbyCells);
    if (flagCount < 1 || flagCount !== cell.Mines) return;
    const openCells: number[] = [];
    for (const i of nearbyCells) {
      if (
        !minefield.value.Cell[i].IsOpen &&
        !minefield.value.Cell[i].IsFlagged &&
        !minefield.value.Cell[i].IsMine
      ) {
        openCells.push(...collectOpen(i));
      }
    }
    if (openCells.length > 0) {
      openSound.play();
      sendOpenList(openCells, now);
    }
    return;
  }

  if (cell.IsOpen || cell.IsFlagged) return;

  openSound.play();

  // 踩雷 → 冷却
  if (cell.IsMine) {
    if (propBarState.value.ShieldCount > 0) {
      ElMessage.warning(`踩雷！护盾保护 (剩余 ${propBarState.value.ShieldCount - 1} 个)`);
    } else {
      boomSound.play();
      applyCooldown();
      ElMessage({ message: "踩雷", type: "info", duration: 800 });
    }
  }

  const openCells = collectOpen(index);
  sendOpenList(openCells, now);
}

/** 收集要打开的格子（含 0 值连锁展开），返回 index 列表 */
function collectOpen(index: number): number[] {
  const result: number[] = [];
  const visited = new Set<number>();
  const queue: number[] = [index];

  while (queue.length > 0) {
    const i = queue.shift()!;
    if (visited.has(i)) continue;
    const cell = minefield.value.Cell[i];
    if (cell.IsOpen || cell.IsFlagged) continue;
    visited.add(i);
    result.push(i);
    if (cell.Mines === 0) {
      for (const nb of getNearbyCells(i)) {
        if (!visited.has(nb)) queue.push(nb);
      }
    }
  }
  return result;
}

function getNearbyFlaggedCount(nearbyCells: number[]) {
  let count = 0;
  for (const i of nearbyCells) {
    if (minefield.value.Cell[i].IsFlagged || minefield.value.Cell[i].IsMine) {
      count++;
    }
  }
  return count;
}

function sendOpenList(openList: number[], now: number) {
  const data: RequestType = { Ids: openList, IsFlag: false, TimeStamp: now };
  ws.send(JSON.stringify(data));
}

// ========== 图片 ==========
function getImageSrc(cell: Cell) {
  const mines = cell.Mines;
  if (cell.IsOpen) {
    if (cell.IsMine) return "/src/assets/themes/wom/flag.png";
    if (cell.Mines === 9) return "/src/assets/themes/wom/closed.png";
    return `/src/assets/themes/wom/type${mines}.png`;
  }
  if (cell.IsFlagged) return "/src/assets/themes/wom/flag.png";
  return "/src/assets/themes/wom/closed.png";
}

// ========== 计时器格式化 ==========
function msToTime(duration: number): string {
  const milliseconds = duration % 10;
  const seconds = Math.floor(duration / 1000);
  const secondsStr = seconds < 10 ? "0" + seconds : String(seconds);
  return `${secondsStr}:${milliseconds}`;
}

// ========== 探测仪高亮 ==========
function applyDetectorHighlights(result: DetectorResult) {
  detectorMineCells.value = new Set(result.MineCells);
  detectorSafeCells.value = new Set(result.SafeCells);
  if (detectorTimer.value) clearTimeout(detectorTimer.value);
  detectorTimer.value = window.setTimeout(() => {
    detectorMineCells.value = new Set();
    detectorSafeCells.value = new Set();
  }, 10000);
}

// ========== 重置 ==========
async function reset() {
  isEnd.value = false;
  minefield.value.Zones = undefined;
  noFlagZoneSet.value = new Set();
  doubleScoreZoneSet.value = new Set();
  overlayVersion.value++;
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
  cdEndTime.value = 0;
  cdRemaining.value = 0;
  cdTotal.value = 0;
  errorCount.value = 0;
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

/* ===== CD 冷却遮罩 ===== */
.cd-overlay-dialog {
  position: absolute;
  inset: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  border-radius: 12px;
}
.cd-overlay-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 24px 36px;
  background: rgba(20, 20, 30, 0.92);
  border-radius: 12px;
  border: 1px solid rgba(255, 100, 100, 0.25);
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.6);
}
.cd-overlay-title {
  font-size: 18px;
  font-weight: 700;
  color: #f66;
}
.cd-progress-bar {
  width: 200px;
  height: 8px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.1);
  overflow: hidden;
}
.cd-progress-fill {
  height: 100%;
  border-radius: 4px;
  background: linear-gradient(90deg, #f66, #fa0);
  transition: width 0.1s linear;
}
.cd-overlay-time {
  font-size: 24px;
  font-weight: 700;
  color: #fff;
  font-variant-numeric: tabular-nums;
}

/* ===== 棋盘 ===== */
.board {
  display: grid;
  position: relative;
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

/* ===== Zone 覆盖层 (per-cell) ===== */
.no-flag-zone-overlay {
  position: absolute;
  inset: 0;
  background: rgba(255, 50, 50, 0.45);
  pointer-events: none;
  z-index: 5;
  border-radius: 2px;
}
.double-score-zone-overlay {
  position: absolute;
  inset: 0;
  background: rgba(255, 200, 50, 0.3);
  pointer-events: none;
  z-index: 4;
  border-radius: 2px;
}

/* ===== 探测仪高亮 ===== */
.cell.detector-mine::after {
  content: "";
  position: absolute;
  inset: 0;
  background: rgba(255, 0, 0, 0.5);
  pointer-events: none;
  z-index: 7;
  animation: detector-pulse 0.8s ease-in-out infinite alternate;
}
.cell.detector-safe::after {
  content: "";
  position: absolute;
  inset: 0;
  background: rgba(0, 230, 120, 0.35);
  pointer-events: none;
  z-index: 7;
  animation: detector-pulse 0.8s ease-in-out infinite alternate;
}
@keyframes detector-pulse {
  from {
    opacity: 0.6;
  }
  to {
    opacity: 1;
  }
}

.timeWatcher {
  font-size: 26px;
  font-weight: bold;
  color: #00bd7e;
}
</style>
