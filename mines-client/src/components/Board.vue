<script lang="ts" setup>
import type {
  ActivePropEffect,
  Cell,
  DetectorResult,
  Minefield,
  PropDropInfo,
  PropSlot,
  ScoreBoard as ScoreBoardType,
  ShieldProtect,
} from '@/types'
import { ElMessage } from 'element-plus'
import { Howl } from 'howler'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { wsClient } from '@/api/websocket'
import PropBar from '@/components/PropBar.vue'
import ScoreBoard from '@/components/ScoreBoard.vue'
import ScoreTip from '@/components/ScoreTip.vue'

// ========== 动作接口 ==========
interface Action { a: number, r: number, c: number }

// ========== 常量 ==========
const emit = defineEmits<{ logout: [] }>()
const cellSize = 24
const COOLDOWN_PER_ERROR = 0.5
const MAX_HINTS = 5

// ========== 快捷键 (localStorage 持久化) ==========
const KEYBINDS_STORAGE_KEY = 'mines-keybinds'
const defaultKeybinds = { flagMode: 'F', detector: 'D', xjbd: 'X' }

function loadKeybinds() {
  try {
    const saved = localStorage.getItem(KEYBINDS_STORAGE_KEY)
    return saved ? { ...defaultKeybinds, ...JSON.parse(saved) } : { ...defaultKeybinds }
  }
  catch { return { ...defaultKeybinds } }
}
const keybinds = ref(loadKeybinds())

// ========== 主题 (localStorage 持久化) ==========
type ThemeName = 'wom' | 'chocolate'
const THEME_STORAGE_KEY = 'mines-theme'
function loadTheme(): ThemeName {
  try { return localStorage.getItem(THEME_STORAGE_KEY) === 'chocolate' ? 'chocolate' : 'wom' }
  catch { return 'wom' }
}
const currentTheme = ref<ThemeName>(loadTheme())
function toggleTheme() {
  currentTheme.value = currentTheme.value === 'wom' ? 'chocolate' : 'wom'
  localStorage.setItem(THEME_STORAGE_KEY, currentTheme.value)
}

// ========== 玩家光标显示 (localStorage 持久化) ==========
type PlayerCursorMode = 'full' | 'avatar' | 'off'
const PLAYER_CURSOR_MODE_KEY = 'mines-player-cursor-mode'
function loadPlayerCursorMode(): PlayerCursorMode {
  try {
    const saved = localStorage.getItem(PLAYER_CURSOR_MODE_KEY)
    return saved === 'avatar' ? 'avatar' : saved === 'off' ? 'off' : 'full'
  }
  catch { return 'full' }
}
const playerCursorMode = ref<PlayerCursorMode>(loadPlayerCursorMode())
function togglePlayerCursorMode() {
  const modes: PlayerCursorMode[] = ['full', 'avatar', 'off']
  const idx = modes.indexOf(playerCursorMode.value)
  playerCursorMode.value = (modes[(idx + 1) % 3] as PlayerCursorMode)
  localStorage.setItem(PLAYER_CURSOR_MODE_KEY, playerCursorMode.value)
}
const showPlayerCursors = computed(() => playerCursorMode.value !== 'off')
const showPlayerNames = computed(() => playerCursorMode.value === 'full')

// ========== 音效 ==========
const openSound = new Howl({ src: ['./audio/open.mp3'], volume: 0.5 })
const flagSound = new Howl({ src: ['./audio/flag.mp3'], volume: 0.5 })
const boomSound = new Howl({ src: ['./audio/boom.mp3'], volume: 0.2 })

// ========== 响应式状态 ==========
const minefield = ref<Minefield>({
  Width: 0,
  Height: 0,
  Cells: 0,
  Mines: 0,
  Cell: [],
  Zones: [],
  First: false,
  StartTimeStamp: 0,
})

const timeWatcher = ref('00:000')
const scoreBoard = ref<ScoreBoardType>({})
const flagMode = ref(false)
const spaceHeld = ref(false)
const effectiveFlagMode = computed(() => flagMode.value !== spaceHeld.value)

// 提示
const hintCount = ref(0)
const serverSafeCells = ref(0)

// CD 冷却系统
const cdEndTime = ref(0)
const cdRemaining = ref(0)
const isBlocked = computed(() => cdRemaining.value > 0)
const cdTimer = ref<number | null>(null)
const serverErrorCount = ref(0)
const puddingCount = ref(0)
const cdTotal = ref(0)
const cdPercent = computed(() =>
  cdTotal.value > 0 ? Math.max(0, (cdRemaining.value / cdTotal.value) * 100) : 0,
)

const isInGame = ref(true)
const currentUserId = ref<string>('')
const currentUserName = ref<string>('')

// 道具系统
const myProps = ref<Record<string, PropSlot>>({})
const myPropsList = computed(() => Object.values(myProps.value))
const usingPropId = ref<number | null>(null)
const activeEffects = ref<Record<string, ActivePropEffect>>({})

// 护盾/双倍状态
const doubleScoreActive = computed(() => !!activeEffects.value[1001])

// ========== 棋盘鼠标指针 ==========
const boardCursor = computed(() => {
  if (isBlocked.value)
    return 'not-allowed'
  if (usingPropId.value === 101)
    return `url('/assets/prop101.png') 64 64, crosshair`
  if (usingPropId.value === 102)
    return `url('/assets/prop102.png') 64 64, crosshair`
  if (effectiveFlagMode.value)
    return 'default'
  return 'pointer'
})
const boardStyle = computed(() => ({
  gridTemplateColumns: `repeat(${minefield.value.Width}, ${cellSize}px)`,
  gridTemplateRows: `repeat(${minefield.value.Height}, ${cellSize}px)`,
  cursor: boardCursor.value,
}))

// ========== 预计算区域/效果集合 ==========
const noFlagZoneSet = ref<Set<number>>(new Set())
const highScoreZoneSet = ref<Set<number>>(new Set())
const detectorRangeSet = ref<Set<number>>(new Set())
const xjbdRangeSet = ref<Set<number>>(new Set())
const overlayVersion = ref(0)

function rebuildZoneSets() {
  const w = minefield.value.Width
  const nfSet = new Set<number>()
  const hsSet = new Set<number>()
  for (const z of minefield.value.Zones) {
    for (let r = z.StartRow; r <= z.EndRow; r++) {
      for (let c = z.StartCol; c <= z.EndCol; c++) {
        const idx = r * w + c
        if (z.Type === 'noFlag')
          nfSet.add(idx)
        else if (z.Type === 'doubleScore')
          hsSet.add(idx)
      }
    }
  }
  noFlagZoneSet.value = nfSet
  highScoreZoneSet.value = hsSet
  overlayVersion.value++
}

function rebuildEffectSets() {
  const w = minefield.value.Width
  const h = minefield.value.Height

  const detSet = new Set<number>()
  const detector = activeEffects.value[101]
  if (detector?.centerCol !== undefined && detector?.centerRow !== undefined) {
    for (let r = Math.max(0, detector.centerRow - 2); r <= Math.min(h - 1, detector.centerRow + 2); r++) {
      for (let c = Math.max(0, detector.centerCol - 2); c <= Math.min(w - 1, detector.centerCol + 2); c++) {
        detSet.add(r * w + c)
      }
    }
  }
  detectorRangeSet.value = detSet

  const xjbdSet = new Set<number>()
  const xjbd = activeEffects.value[102]
  if (xjbd?.centerCol !== undefined && xjbd?.centerRow !== undefined) {
    for (let r = Math.max(0, xjbd.centerRow - 3); r <= Math.min(h - 1, xjbd.centerRow + 3); r++) {
      for (let c = Math.max(0, xjbd.centerCol - 3); c <= Math.min(w - 1, xjbd.centerCol + 3); c++) {
        xjbdSet.add(r * w + c)
      }
    }
  }
  xjbdRangeSet.value = xjbdSet
  overlayVersion.value++
}

// ========== 邻近格子缓存 ==========
let nearbyCache: number[][] = []
function buildNearbyCache() {
  const w = minefield.value.Width
  const h = minefield.value.Height
  const total = w * h
  nearbyCache = Array.from({ length: total })
  for (let i = 0; i < total; i++) {
    const x = i % w; const y = Math.floor(i / w)
    const nearby: number[] = []
    if (y > 0)
      nearby.push(i - w)
    if (y < h - 1)
      nearby.push(i + w)
    if (x > 0) {
      nearby.push(i - 1)
      if (y > 0)
        nearby.push(i - w - 1)
      if (y < h - 1)
        nearby.push(i + w - 1)
    }
    if (x < w - 1) {
      nearby.push(i + 1)
      if (y > 0)
        nearby.push(i - w + 1)
      if (y < h - 1)
        nearby.push(i + w + 1)
    }
    nearbyCache[i] = nearby
  }
}

// 结算
const showResultDialog = ref(false)
const resultList = ref<{ name: string, score: number }[]>([])

// 计时器
let startTimeStamp = 0
let timerRunning = false
let intervalFlag: number
let effectUpdateTimer: number | null = null

// 玩家光标追踪
interface PlayerCursor { name: string, cellIndex: number }
const playerCursors = ref<Record<string, PlayerCursor>>({})
interface TrailParticle { id: number, name: string, fromIndex: number, toIndex: number }
const trails = ref<TrailParticle[]>([])
let trailIdSeq = 0
const initializedCursors = ref<Set<string>>(new Set())

const scoreTip = ref<InstanceType<typeof ScoreTip> | null>(null)

// Detector mine cells (for overlay in template)
const detectorMineCells = ref<Set<number>>(new Set())

function getWsUrl(uid: string): string {
  const token = localStorage.getItem('token') || ''
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${location.host}/ws/${uid}?token=${token}`
}

document.oncontextmenu = () => false

// ========== 快捷键 ==========
function onKeydown(e: KeyboardEvent) {
  const tag = (e.target as HTMLElement)?.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT')
    return
  if (e.metaKey || e.ctrlKey || e.altKey)
    return
  const key = e.key.toUpperCase()
  if (e.code === 'Space') { e.preventDefault(); spaceHeld.value = true; return }
  if (key === keybinds.value.flagMode.toUpperCase()) { e.preventDefault(); flagMode.value = !flagMode.value; return }
  if (key === keybinds.value.detector.toUpperCase() && hasProp(101)) { e.preventDefault(); startUseProp(101); return }
  if (key === keybinds.value.xjbd.toUpperCase() && hasProp(102)) { e.preventDefault(); startUseProp(102) }
}
function onKeyup(e: KeyboardEvent) {
  if (e.code === 'Space')
    spaceHeld.value = false
}
function hasProp(propId: number): boolean {
  const p = myProps.value[String(propId)]
  return !!(p && p.Count > 0)
}

// ========== 生命周期 ==========
onMounted(() => {
  const uid = localStorage.getItem('uid')
  if (uid)
    currentUserId.value = uid
  initGame()
  startEffectUpdateLoop()
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('keyup', onKeyup)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('keyup', onKeyup)
  cleanup()
})

function cleanup() {
  wsClient.close()
  if (timerRunning)
    clearInterval(intervalFlag)
  if (cdTimer.value)
    clearInterval(cdTimer.value)
  if (effectUpdateTimer)
    clearInterval(effectUpdateTimer)
  playerCursors.value = {}
  trails.value = []
  initializedCursors.value = new Set()
}

// ========== WebSocket 初始化 ==========
function initGame() {
  // First get initial minefield via HTTP
  fetch('/getMinefield', { method: 'POST' })
    .then(r => r.json())
    .then((data: Minefield) => {
      minefield.value = data
      if (data.StartTimeStamp)
        startTimeStamp = data.StartTimeStamp
      rebuildZoneSets()
      buildNearbyCache()
      startTimer()
    })
    .catch(e => console.error('Failed to get minefield:', e))

  // Then connect WebSocket
  const uid = currentUserId.value
  wsClient.onMessage(handleMessage)
  wsClient.connect(getWsUrl(uid))
}

// ========== 消息路由 ==========
function handleMessage(data: any) {
  const mt = data.MessageType

  if (mt === 'init') {
    handleInit(data)
  }
  else if (mt === 'propEffect') {
    handleRemotePropEffect(data)
  }
  else if (mt === 'personal') {
    handlePersonal(data)
  }
  else if (mt === 'detectorResult') {
    handleDetectorResult(data)
  }
  else if (mt === 'hintResult') {
    handleHintResult(data)
  }
  else if (data.PlayerQuit) {
    handlePlayerQuit(data)
  }
  else if (data.NewPlayer) {
    handlePlayerJoin(data)
  }
  else {
    // Normal action broadcast
    handleAction(data)
  }
}

// ========== init — 初始状态 ==========
function handleInit(data: any) {
  if (data.UserName) {
    currentUserName.value = data.UserName
  }
  if (data.Minefield) {
    minefield.value = data.Minefield
    startTimeStamp = data.StartTimeStamp || Date.now()
    minefield.value.StartTimeStamp = startTimeStamp
    rebuildZoneSets()
    buildNearbyCache()
    startTimer()
  }
  if (data.ScoreBoard) {
    scoreBoard.value = data.ScoreBoard
  }
  if (data.SafeCells !== undefined) {
    serverSafeCells.value = data.SafeCells
  }
}

// ========== 普通操作广播 ==========
function handleAction(data: any) {
  if (data.ChangeCell?.Cell) {
    for (const c of data.ChangeCell.Cell) {
      const cell = minefield.value.Cell[c.Id]
      if (cell)
        Object.assign(cell, c)
    }
    // Check win
    if (data.ChangeCell.Result?.IsWin) {
      handleFinish(data)
    }
    // Count errors from negative score (wrong flag or opened mine)
    const earnScore = data.EarnScore ?? 0
    if (earnScore < 0 && data.UserName === currentUserName.value) {
      applyCooldown()
      boomSound.play()
    }
  }
  if (data.ScoreBoard) {
    const prevSelfScore = scoreBoard.value[currentUserName.value] ?? 0
    scoreBoard.value = data.ScoreBoard
    const newScore = data.ScoreBoard[currentUserName.value] ?? 0
    if (newScore !== prevSelfScore && scoreTip.value) {
      scoreTip.value.tips(newScore - prevSelfScore, doubleScoreActive.value)
    }
  }
  // Track player cursor
  if (data.UserName && data.UserName !== currentUserName.value && data.ChangeCell?.Cell?.length) {
    const lastCell = data.ChangeCell.Cell[data.ChangeCell.Cell.length - 1]
    trackPlayerCursor(data.UserName, lastCell.Id)
  }
  if (data.SafeCells !== undefined)
    serverSafeCells.value = data.SafeCells
}

// ========== 个人消息 (道具栏更新/道具获得/护盾) ==========
function handlePersonal(data: any) {
  if (data.PropBarUpdate)
    applyPropBarUpdate(data.PropBarUpdate)
  if (data.PropDrop)
    handlePropGain(data.PropDrop)
  if (data.ShieldProtect)
    handleShieldProtect(data.ShieldProtect)
  if (data.EarnScore && data.EarnScore > 0 && scoreTip.value) {
    scoreTip.value.tips(data.EarnScore, doubleScoreActive.value)
  }
}

// ========== 服务端探测仪结果 ==========
function handleDetectorResult(data: any) {
  const result = data.DetectorResult as DetectorResult
  if (!result)
    return
  usingPropId.value = null
  // Activate 5x5 visual effect
  const cx = result.CenterCell % minefield.value.Width
  const cy = Math.floor(result.CenterCell / minefield.value.Width)
  activeEffects.value[101] = {
    propId: 101,
    propName: '探测仪',
    remainingMs: 10000,
    startTime: Date.now(),
    centerCol: cx,
    centerRow: cy,
  }
  rebuildEffectSets()
  // Show mine cell info via overlay: mark mine cells so detector-highlight-mine shows
  // Store mine positions for overlay rendering
  detectorMineCells.value = new Set(result.MineCells)
  ElMessage.success(`探测仪: ${result.MineCells.length} 个雷`)
}

// ========== 提示结果 ==========
function handleHintResult(data: any) {
  if (data.ChangeCell?.Cell?.length) {
    const hintCell = data.ChangeCell.Cell[0]
    // Open the hint cell as if the player clicked it
    const r = Math.floor(hintCell.Id / minefield.value.Width)
    const c = hintCell.Id % minefield.value.Width
    doOpen({ a: 0, c, r })
  }
}

// ========== 玩家加入/离开 ==========
function handlePlayerJoin(data: any) {
  if (data.UserName) {
    ElMessage.info(`${data.UserName} 加入了游戏`)
  }
}
function handlePlayerQuit(data: any) {
  if (data.UserName) {
    // Remove from scoreboard
    const newScores = { ...scoreBoard.value }
    delete newScores[data.UserName]
    scoreBoard.value = newScores
    // Remove cursor
    delete playerCursors.value[data.UserName]
    ElMessage.info(`${data.UserName} 离开了游戏`)
  }
}

// ========== 结算 ==========
function handleFinish(data: any) {
  const board = data.ScoreBoard || scoreBoard.value
  const list = Object.entries(board).map(([name, score]) => ({ name, score: score as number }))
  list.sort((a, b) => b.score - a.score)
  resultList.value = list
  showResultDialog.value = true
  if (timerRunning) { clearInterval(intervalFlag); timerRunning = false }
}

// ========== 道具获得 ==========
function handlePropGain(drop: PropDropInfo) {
  const propNames: Record<number, string> = { 101: '探测仪', 102: '雷之奥义', 1001: '双倍积分', 1002: '护盾' }
  ElMessage.success(`获得道具: ${propNames[drop.PropID] || drop.PropName} (+${drop.Count})`)
  // Double score auto-activate
  if (drop.PropID === 1001) {
    activeEffects.value[1001] = {
      propId: 1001,
      propName: '双倍积分',
      remainingMs: 10000,
      startTime: Date.now(),
    }
  }
}

// ========== 护盾保护 ==========
function handleShieldProtect(sp: ShieldProtect) {
  const msg = sp.WasMine ? '踩雷！护盾保护' : '标记错误！护盾保护'
  ElMessage.warning(`${msg} (剩余 ${sp.ShieldCount} 个)`)
}

// ========== 远程道具效果 ==========
function handleRemotePropEffect(data: any) {
  if (!data.PropEffect)
    return
  const ef = data.PropEffect
  if (ef.PropID === 101) {
    // Another player used detector — show visual briefly
    const cx = ef.TargetCell % minefield.value.Width
    const cy = Math.floor(ef.TargetCell / minefield.value.Width)
    activeEffects.value[`remote_101_${ef.UserName}`] = {
      propId: 101,
      propName: '探测仪',
      remainingMs: 3000,
      startTime: Date.now(),
      centerCol: cx,
      centerRow: cy,
    }
    rebuildEffectSets()
  }
  else if (ef.PropID === 102) {
    const cx = ef.TargetCell % minefield.value.Width
    const cy = Math.floor(ef.TargetCell / minefield.value.Width)
    activeEffects.value[`remote_102_${ef.UserName}`] = {
      propId: 102,
      propName: '雷之奥义',
      remainingMs: 1500,
      startTime: Date.now(),
      centerCol: cx,
      centerRow: cy,
    }
    rebuildEffectSets()
  }
}

// ========== 道具栏更新 ==========
function applyPropBarUpdate(update: any) {
  const inventory = update.Inventory as PropSlot[]
  const newProps: Record<string, PropSlot> = {}
  for (const slot of inventory) {
    newProps[String(slot.PropID)] = slot
  }
  myProps.value = newProps

  if (update.DoubleScoreActive) {
    activeEffects.value[1001] = {
      propId: 1001,
      propName: '双倍积分',
      remainingMs: update.DoubleScoreRemaining * 1000,
      startTime: Date.now(),
    }
  }
  else {
    delete activeEffects.value[1001]
  }
  rebuildEffectSets()
}

// ========== 道具使用流程 ==========
function startUseProp(propId: number) {
  if (isBlocked.value) { ElMessage.warning('冷却中，请等待'); return }
  if (usingPropId.value === propId) { usingPropId.value = null; return }
  usingPropId.value = propId
}

// ========== 点击处理 ==========
function handleClick(event: MouseEvent, index: number) {
  if (isBlocked.value) { ElMessage.warning(`冷却中，${cdRemaining.value.toFixed(1)}s 后恢复`); return }

  const cell = minefield.value.Cell[index]
  if (!cell)
    return
  const r = Math.floor(index / minefield.value.Width)
  const c = index % minefield.value.Width

  // 道具模式
  if (usingPropId.value !== null) {
    handlePropUse(usingPropId.value, r, c)
    return
  }

  flagSound.stop()
  openSound.stop()

  const isRightClick = event.button === 2
  const shouldFlag = isRightClick !== effectiveFlagMode.value

  // 红区检查
  if (shouldFlag && isInNoFlagZone(index)) { ElMessage.warning('该区域禁止标记，只能打开'); return }

  if (shouldFlag) {
    flagSound.play()
    if (!cell.IsOpen) {
      // We can't check IsMine client-side for unopened cells — server handles penalty
      doFlag({ a: 1, c, r })
    }
    else {
      doExpand(index)
    }
  }
  else {
    openSound.play()
    if (!cell.IsOpen && !cell.IsFlagged) {
      doOpen({ a: 0, c, r })
    }
    else if (cell.IsOpen) {
      doExpand(index)
    }
  }
}

// ========== 道具使用 (服务端驱动) ==========
function handlePropUse(propId: number, row: number, col: number) {
  const prop = myProps.value[String(propId)]
  if (!prop || prop.Count <= 0) { ElMessage.error('没有该道具'); usingPropId.value = null; return }

  // 发送到服务端
  const actionType = propId === 101 ? 'useDetector' : 'useXJBD'
  wsClient.send({
    Ids: [],
    IsFlag: false,
    TimeStamp: Date.now(),
    ActionType: actionType,
    TargetCell: row * minefield.value.Width + col,
  })

  // 本地扣减
  prop.Count -= 1
  if (prop.Count <= 0)
    delete myProps.value[String(propId)]

  // XJBD: 服务端会返回 full cell changes via broadcast
  // Detector: 服务端会返回 detectorResult via personal message

  const propNames: Record<number, string> = { 101: '探测仪', 102: '雷之奥义' }
  ElMessage.success(`使用 ${propNames[propId]}`)
}

// ========== 操作 ==========
function doOpen(action: Action) {
  const index = action.r * minefield.value.Width + action.c
  const cell = minefield.value.Cell[index]
  if (!cell)
    return
  // Send all cascading cells if 0-value
  const ids = [index]
  if (cell.Mines === 0 && cell.IsOpen) {
    // Already open — this is expand, handled separately
  }
  wsClient.send({ Ids: ids, IsFlag: false, TimeStamp: Date.now() })
}

function doFlag(action: Action) {
  const index = action.r * minefield.value.Width + action.c
  wsClient.send({ Ids: [index], IsFlag: true, TimeStamp: Date.now() })
}

function doExpand(index: number) {
  const cell = minefield.value.Cell[index]
  if (!cell || !cell.IsOpen)
    return

  const nearby = getNearbyCells(index)
  // Count flagged cells (opened mines count as flagged too)
  const flagCount = nearby.filter((n) => {
    const nc = minefield.value.Cell[n]
    if (!nc)
      return false
    return nc.IsFlagged || (nc.IsOpen && nc.IsMine)
  }).length

  if (flagCount === cell.Mines) {
    openSound.play()
    const ids: number[] = []
    nearby.forEach((i) => {
      const nCell = minefield.value.Cell[i]
      if (nCell && !nCell.IsOpen && !nCell.IsFlagged) {
        ids.push(i)
      }
    })
    if (ids.length > 0) {
      wsClient.send({ Ids: ids, IsFlag: false, TimeStamp: Date.now() })
    }
  }
}

function getNearbyCells(cell: number): number[] {
  return nearbyCache[cell] || []
}

// ========== 冷却系统 ==========
function applyCooldown() {
  const now = Date.now()
  serverErrorCount.value++
  const reduction = Math.min(puddingCount.value * 0.01, 0.99)
  const basePenalty = 2.5 + serverErrorCount.value * COOLDOWN_PER_ERROR
  const penalty = basePenalty * (1 - reduction)
  cdTotal.value = penalty
  cdEndTime.value = Math.max(cdEndTime.value, now) + penalty * 1000
  cdRemaining.value = (cdEndTime.value - now) / 1000
  startCdTimer()
}
function startCdTimer() {
  if (cdTimer.value)
    return
  cdTimer.value = window.setInterval(() => {
    const remaining = Math.max(0, (cdEndTime.value - Date.now()) / 1000)
    cdRemaining.value = remaining
    if (remaining <= 0) { cdRemaining.value = 0; if (cdTimer.value) { clearInterval(cdTimer.value); cdTimer.value = null } }
  }, 100)
}

// ========== 效果更新循环 ==========
function startEffectUpdateLoop() {
  effectUpdateTimer = window.setInterval(() => {
    const now = Date.now()
    const expiredIds: string[] = []
    for (const key of Object.keys(activeEffects.value)) {
      const e = activeEffects.value[key]
      if (!e)
        continue
      if (now - e.startTime >= e.remainingMs) {
        expiredIds.push(key)
      }
    }
    for (const id of expiredIds) {
      if (id === '1001') {
        // Double score expired — remove from inventory
        const p = myProps.value['1001']
        if (p) {
          p.Count -= 1; if (p.Count <= 0)
            delete myProps.value['1001']
        }
      }
      if (id.startsWith('remote_')) {
        // Clean up remote effects
      }
      delete activeEffects.value[id]
    }
    if (expiredIds.length > 0)
      rebuildEffectSets()
  }, 500)
}

// ========== 图片 ==========
function getImageSrc(cell: Cell): string {
  const theme = currentTheme.value
  if (cell.IsOpen) {
    if (cell.IsMine)
      return `./themes/${theme}/flag.png`
    const mines = cell.Mines
    if (mines >= 0 && mines <= 8)
      return `./themes/${theme}/type${mines}.png`
    return `./themes/${theme}/closed.png`
  }
  if (cell.IsFlagged)
    return `./themes/${theme}/flag.png`
  return `./themes/${theme}/closed.png`
}

// ========== 提示 ==========
function doHint() {
  if (hintCount.value >= MAX_HINTS) { ElMessage.warning('提示次数已用完'); return }
  wsClient.send({ Ids: [], IsFlag: false, TimeStamp: Date.now(), ActionType: 'hint' })
  hintCount.value++
  if (hintCount.value >= MAX_HINTS)
    ElMessage.info('提示次数已用完')
}

// ========== 区域判断 ==========
function isInNoFlagZone(index: number): boolean {
  return noFlagZoneSet.value.has(index)
}

// ========== 计时器 ==========
function startTimer() {
  if (!timerRunning) {
    timerRunning = true
    intervalFlag = window.setInterval(() => {
      timeWatcher.value = msToTime(Date.now() - startTimeStamp)
    }, 50)
  }
}
function msToTime(duration: number): string {
  const ms = duration % 10
  const seconds = Math.floor(duration / 1000)
  const secondsStr = seconds < 10 ? `0${seconds}` : String(seconds)
  return `${secondsStr}:${ms}`
}

// ========== 玩家光标追踪 ==========
function trackPlayerCursor(userName: string, cellIndex: number) {
  const prev = playerCursors.value[userName]
  if (prev && prev.cellIndex !== cellIndex) {
    const trail: TrailParticle = { id: ++trailIdSeq, name: userName, fromIndex: prev.cellIndex, toIndex: cellIndex }
    trails.value = [...trails.value, trail]
    setTimeout(() => { trails.value = trails.value.filter(t => t.id !== trail.id) }, 600)
  }
  if (!initializedCursors.value.has(userName)) {
    initializedCursors.value = new Set([...initializedCursors.value, userName])
  }
  playerCursors.value = { ...playerCursors.value, [userName]: { name: userName, cellIndex } }
}
function cursorInitialized(uid: string): boolean { return initializedCursors.value.has(uid) }
function playerCursorStyle(cursor: PlayerCursor) {
  const pos = cellIndexToPos(cursor.cellIndex)
  return { left: `${pos.x + cellSize / 2}px`, top: `${pos.y + cellSize / 2}px` }
}
function cellIndexToPos(index: number) {
  const col = index % minefield.value.Width
  const row = Math.floor(index / minefield.value.Width)
  return { x: col * cellSize, y: row * cellSize }
}
function trailStyle(trail: TrailParticle) {
  const from = cellIndexToPos(trail.fromIndex)
  const to = cellIndexToPos(trail.toIndex)
  return { 'left': `${to.x}px`, 'top': `${to.y}px`, '--trail-from-x': `${from.x}px`, '--trail-from-y': `${from.y}px`, '--trail-to-x': `${to.x}px`, '--trail-to-y': `${to.y}px` } as any
}

// ========== 结算 UI ==========
function getRankIcon(rank: number) {
  if (rank === 1)
    return '🥇'
  if (rank === 2)
    return '🥈'
  if (rank === 3)
    return '🥉'
  return ''
}

// ========== 退出登录 ==========
function doLogout() {
  cleanup()
  emit('logout')
}

// ========== 重置 ==========
async function reset() {
  cleanup()
  minefield.value = { Width: 0, Height: 0, Cells: 0, Mines: 0, Cell: [], Zones: [], First: false, StartTimeStamp: 0 }
  scoreBoard.value = {}
  myProps.value = {}
  activeEffects.value = {}
  cdEndTime.value = 0; cdRemaining.value = 0; cdTotal.value = 0
  serverErrorCount.value = 0; puddingCount.value = 0; hintCount.value = 0
  usingPropId.value = null
  timeWatcher.value = '00:000'; timerRunning = false
  isInGame.value = true
  // Call newGame to reset server-side board, then reconnect
  await fetch('/newGame', { method: 'POST' })
  wsClient.onMessage(handleMessage)
  wsClient.connect(getWsUrl(currentUserId.value))
  // Re-fetch minefield
  fetch('/getMinefield', { method: 'POST' }).then(r => r.json()).then((data: Minefield) => {
    minefield.value = data
    if (data.StartTimeStamp)
      startTimeStamp = data.StartTimeStamp
    rebuildZoneSets(); buildNearbyCache(); startTimer()
  })
  startEffectUpdateLoop()
}
</script>

<template>
  <!-- =================== 结算弹窗 =================== -->
  <div v-if="showResultDialog">
    <el-dialog v-model="showResultDialog" width="480px" :show-close="false" :close-on-click-modal="false" :close-on-press-escape="false" center>
      <div style="text-align: center; padding: 0 0 10px 0">
        <div v-if="resultList.length > 0" style="margin-bottom: 18px">
          <div style="display: flex; flex-direction: column; align-items: center">
            <div style="font-size: 3rem">
              🏆
            </div>
            <div style="font-size: 1.3rem; font-weight: bold; margin-top: 8px">
              {{ resultList[0]?.name }}
            </div>
            <div style="color: #fbbf24; font-size: 1.5rem; font-weight: 800; margin-top: 4px">
              {{ resultList[0]?.score }} 分
            </div>
          </div>
        </div>
        <el-divider style="margin: 10px 0" />
        <div
          v-for="(item, idx) in resultList" :key="item.name"
          style="display: flex; align-items: center; justify-content: space-between; margin: 8px 0"
        >
          <div style="display: flex; align-items: center">
            <span v-if="idx < 3" style="font-size: 1.3rem; width: 2.2em; text-align: center">{{ getRankIcon(idx + 1) }}</span>
            <span :style="{ fontWeight: idx < 3 ? 700 : 500, fontSize: idx === 0 ? '1.1rem' : '0.95rem' }">{{ item.name }}</span>
          </div>
          <span style="font-weight: 700; color: #60a5fa">{{ item.score }} 分</span>
        </div>
        <div style="display: flex; justify-content: space-between; margin-top: 18px">
          <el-button type="default" @click="showResultDialog = false">
            返回
          </el-button>
          <el-button type="primary" @click="showResultDialog = false; reset()">
            再来一局
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>

  <!-- =================== 顶部栏 =================== -->
  <div class="topPositionFixed">
    <div class="header-content">
      <el-button class="logout-button" style="width: auto" :disabled="hintCount >= MAX_HINTS" @click="doHint">
        提示 ({{ MAX_HINTS - hintCount }})
      </el-button>
      <el-button
        :style="{ background: effectiveFlagMode ? '#5282b8' : '#5c8f4b', width: 'auto' }"
        class="flag-switch-button"
        @click="flagMode = !flagMode"
      >
        {{ effectiveFlagMode ? '标记' : '挖开' }}模式
        <kbd class="key-hint">{{ keybinds.flagMode }}</kbd>
        <span v-if="spaceHeld" class="space-held-hint">[Space]</span>
      </el-button>

      <el-button class="theme-switch-button" @click="toggleTheme">
        {{ currentTheme === 'wom' ? 'WOM' : '巧克力' }}
      </el-button>

      <el-button class="cursor-mode-button" @click="togglePlayerCursorMode">
        {{ playerCursorMode === 'full' ? '👤 玩家' : playerCursorMode === 'avatar' ? '👤 仅头像' : '👤 隐藏' }}
      </el-button>

      <el-button type="danger" style="width: auto" @click="doLogout">
        退出登录
      </el-button>

      <div class="safe-cells-indicator">
        剩余安全格子 <span class="safe-cells-count">{{ serverSafeCells }}</span>
      </div>
      <div class="timeWatcher">
        {{ timeWatcher }}
      </div>
    </div>
    <ScoreTip ref="scoreTip" class="scoreTipParent" />
  </div>

  <!-- =================== 主体 =================== -->
  <div class="main-layout">
    <div class="left-panel">
      <ScoreBoard v-if="Object.keys(scoreBoard).length > 0" :score-board="scoreBoard" />
    </div>

    <div class="center-panel">
      <!-- 冷却遮罩 -->
      <div v-if="isBlocked" class="cd-overlay-dialog">
        <div class="cd-overlay-card">
          <span class="cd-overlay-title">冷却中</span>
          <div class="cd-progress-bar">
            <div class="cd-progress-fill" :style="{ width: `${cdPercent}%` }" />
          </div>
          <span class="cd-overlay-time">{{ cdRemaining.toFixed(1) }}s</span>
        </div>
      </div>

      <el-scrollbar>
        <div v-if="minefield.Width > 0" :style="boardStyle" class="board">
          <div
            v-for="(cell, index) in minefield.Cell" :key="index"
            v-memo="[cell.IsOpen, cell.IsFlagged, cell.IsMine, cell.Mines, isBlocked, currentTheme, overlayVersion]"
            class="cell-wrapper"
          >
            <div
              :style="{ backgroundImage: `url(${getImageSrc(cell)})` }" class="cell"
              @mousedown="(event: MouseEvent) => handleClick(event, index)"
            />
            <div v-if="isBlocked" class="cd-overlay" />
            <div v-if="noFlagZoneSet.has(index)" class="no-flag-zone-overlay" />
            <div v-if="highScoreZoneSet.has(index)" class="high-score-zone-overlay" />
            <div
              v-if="detectorRangeSet.has(index)" class="detector-highlight"
              :class="{ 'detector-highlight-mine': detectorMineCells.has(index) }"
            />
            <div v-if="xjbdRangeSet.has(index)" class="xjbd-highlight" />
          </div>

          <!-- 其他玩家光标 -->
          <template v-for="(cursor, name) in playerCursors" :key="name">
            <div
              v-if="showPlayerCursors && name !== currentUserName"
              class="player-cursor" :class="{ 'player-cursor--init': !cursorInitialized(name) }"
              :style="playerCursorStyle(cursor)"
            >
              <div class="player-cursor-dot" />
              <span v-if="showPlayerNames" class="player-cursor-name">{{ cursor.name }}</span>
            </div>
          </template>

          <!-- 轨迹粒子 -->
          <template v-if="showPlayerCursors">
            <div v-for="trail in trails" :key="trail.id" class="trail-particle" :style="trailStyle(trail)">
              <div class="trail-dot" />
            </div>
          </template>
        </div>
      </el-scrollbar>
    </div>
  </div>

  <!-- =================== 道具栏 =================== -->
  <div class="prop-bar-fixed">
    <PropBar
      :props="myPropsList" :using-prop-id="usingPropId" :cd-remaining="cdRemaining"
      :active-effects="activeEffects" @use-prop="startUseProp"
    />
  </div>
</template>

<style scoped>
/* ===== 顶部栏 ===== */
.topPositionFixed {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 10px 16px 6px;
  background: rgba(255, 255, 255, 0.04);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}
.header-content {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
  justify-content: center;
}
.flag-switch-button {
  color: #fff !important;
  border: none !important;
  font-weight: 700;
  transition: all 0.25s;
}
.theme-switch-button {
  width: 5.5rem;
  background: #7c5c3c !important;
  color: #f5e6d3 !important;
  border: none !important;
  font-weight: 700;
  font-size: 12px;
  transition: all 0.25s;
}
.theme-switch-button:hover {
  background: #9b7353 !important;
}

.safe-cells-indicator {
  font-size: 14px;
  font-weight: 600;
  color: #a5d6a7;
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  margin-left: 8px;
  padding: 4px 12px;
  background: rgba(165, 214, 167, 0.1);
  border-radius: 6px;
  border: 1px solid rgba(165, 214, 167, 0.15);
  white-space: nowrap;
}
.safe-cells-count {
  color: #66bb6a;
  font-size: 18px;
  font-weight: 800;
}
.timeWatcher {
  font-size: 28px;
  font-weight: 800;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-variant-numeric: tabular-nums;
  color: #5eead4;
  margin-left: 16px;
  text-shadow: 0 0 16px rgba(94, 234, 212, 0.3);
  width: 110px;
  text-align: center;
  flex-shrink: 0;
}
.scoreTipParent {
  height: 0;
  overflow: visible;
  position: relative;
  width: 100%;
  display: flex;
  justify-content: center;
}
.prop-bar-fixed {
  position: fixed;
  bottom: 50px;
  left: 16px;
  z-index: 100;
}

/* ===== 主体 ===== */
.main-layout {
  display: flex;
  justify-content: flex-start;
  gap: 16px;
  height: calc(100vh - 112px);
  max-width: 100vw;
  padding: 10px 12px;
  overflow-x: auto;
}
.left-panel {
  width: 235px;
  flex-shrink: 0;
  overflow-y: auto;
  border-radius: 12px;
}
.center-panel {
  position: relative;
  flex-grow: 1;
  display: flex;
  justify-content: flex-start;
  padding-left: 8px;
  max-width: calc(100% - 275px);
}

/* ===== 冷却遮罩 ===== */
.cd-overlay-dialog {
  position: absolute;
  inset: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  border-radius: 12px;
  pointer-events: all;
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
  --cell-size: 24px;
  position: relative;
  display: grid;
  margin: 0 auto;
  padding: 8px;
  background: rgba(0, 0, 0, 0.35);
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.4),
    inset 0 1px 0 rgba(255, 255, 255, 0.03);
}
.cell-wrapper {
  position: relative;
  width: var(--cell-size);
  height: var(--cell-size);
  transition: transform 0.1s;
}
.cell-wrapper:active {
  transform: scale(0.92);
}
.cell {
  width: 100%;
  height: 100%;
  background-size: cover;
  box-sizing: border-box;
  image-rendering: pixelated;
  border-radius: 2px;
}
.cell:hover {
  filter: brightness(1.25) saturate(1.1);
  z-index: 1;
}

/* ===== 玩家光标 ===== */
.player-cursor {
  position: absolute;
  z-index: 10;
  pointer-events: none;
  display: flex;
  flex-direction: column;
  align-items: center;
  transform: translate(-50%, -50%);
  transition:
    left 0.35s ease,
    top 0.35s ease;
}
.player-cursor--init {
  transition: none;
  animation: cursor-pop-in 0.3s ease-out;
}
.player-cursor-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #ffd700;
  border: 2px solid #fff;
  box-shadow: 0 0 8px rgba(255, 215, 0, 0.6);
}
.player-cursor-name {
  font-size: 9px;
  color: #fff;
  background: rgba(0, 0, 0, 0.75);
  padding: 1px 5px;
  border-radius: 4px;
  white-space: nowrap;
  margin-top: 2px;
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
}

@keyframes cursor-pop-in {
  0% {
    transform: translate(-50%, -50%) scale(0);
    opacity: 0;
  }
  60% {
    transform: translate(-50%, -50%) scale(1.15);
  }
  100% {
    transform: translate(-50%, -50%) scale(1);
    opacity: 1;
  }
}

/* ===== 轨迹粒子 ===== */
.trail-particle {
  position: absolute;
  z-index: 9;
  pointer-events: none;
  animation: trail-fly 0.5s ease-out forwards;
}
.trail-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: rgba(255, 215, 0, 0.5);
}

@keyframes trail-fly {
  0% {
    transform: translate(
      calc(var(--trail-from-x) - var(--trail-to-x, 0px)),
      calc(var(--trail-from-y) - var(--trail-to-y, 0px))
    );
    opacity: 0.8;
  }
  100% {
    transform: translate(0, 0);
    opacity: 0;
  }
}

/* ===== 快捷键提示 ===== */
.key-hint {
  display: inline-block;
  font-size: 10px;
  font-family: inherit;
  padding: 1px 6px;
  margin-left: 6px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.08);
  color: #aaa;
  line-height: 1.5;
  vertical-align: middle;
}

/* ===== 遮罩层 ===== */
.cd-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: repeating-linear-gradient(
    0deg,
    rgba(255, 60, 60, 0.12) 0px,
    rgba(255, 60, 60, 0.12) 2px,
    transparent 2px,
    transparent 8px
  );
  pointer-events: none;
  z-index: 10;
  border-radius: 2px;
}
.no-flag-zone-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 50, 50, 0.45);
  pointer-events: none;
  z-index: 5;
  border-radius: 2px;
}
.high-score-zone-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 200, 50, 0.3);
  pointer-events: none;
  z-index: 4;
  border-radius: 2px;
}
.detector-highlight {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 230, 120, 0.22);
  pointer-events: none;
  z-index: 5;
  border-radius: 2px;
  box-shadow: inset 0 0 8px rgba(0, 230, 120, 0.15);
}
.detector-highlight-mine {
  background: rgba(255, 80, 40, 0.55);
  border: 1.5px dashed rgba(255, 80, 40, 0.9);
  box-shadow: inset 0 0 10px rgba(255, 80, 40, 0.3);
}
.xjbd-highlight {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(140, 60, 255, 0.28);
  pointer-events: none;
  z-index: 6;
  border-radius: 2px;
  animation: xjbd-flash 0.35s ease-in-out infinite alternate;
}
@keyframes xjbd-flash {
  from {
    background: rgba(140, 60, 255, 0.18);
    box-shadow: inset 0 0 6px rgba(140, 60, 255, 0.2);
  }
  to {
    background: rgba(140, 60, 255, 0.45);
    box-shadow: inset 0 0 14px rgba(140, 60, 255, 0.4);
  }
}
</style>
