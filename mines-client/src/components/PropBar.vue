<script lang="ts" setup>
import type { ActivePropEffect, PropSlot } from '@/types'
import { onMounted, onUnmounted, ref } from 'vue'

const props = defineProps<{
  props: PropSlot[]
  usingPropId: number | null
  cdRemaining: number
  activeEffects: Record<number, ActivePropEffect>
}>()

defineEmits<{
  useProp: [propId: number]
}>()

const now = ref(Date.now())
let timer: number | null = null

onMounted(() => { timer = window.setInterval(() => { now.value = Date.now() }, 100) })
onUnmounted(() => {
  if (timer)
    clearInterval(timer)
})

// Prop type: 101=detector(active), 102=xjbd(active), 1001=doubleScore(passive), 1002=shield(passive)
function isActiveProp(propId: number): boolean {
  return propId === 101 || propId === 102
}

function isAutoActive(propId: number): boolean {
  const e = props.activeEffects[propId]
  return !!e && (e.startTime + e.remainingMs) > now.value
}

function getAutoRemaining(propId: number): number {
  const e = props.activeEffects[propId]
  if (!e)
    return 0
  return Math.max(0, (e.startTime + e.remainingMs - now.value) / 1000)
}

function getBadgeValue(prop: PropSlot): string | number {
  if (!isActiveProp(prop.PropID) && isAutoActive(prop.PropID)) {
    return `${getAutoRemaining(prop.PropID).toFixed(1)}s`
  }
  return prop.Count
}

function getTimerPercent(propId: number): number {
  const e = props.activeEffects[propId]
  if (!e || e.remainingMs <= 0)
    return 0
  const remaining = Math.max(0, e.startTime + e.remainingMs - now.value)
  return (remaining / e.remainingMs) * 100
}

const propDisplayNames: Record<number, string> = { 101: '探测仪', 102: '雷之奥义', 1001: '双倍', 1002: '护盾' }

const propDescs: Record<number, string> = {
  101: '显示5x5范围的雷,持续10秒。点击后选择格子使用',
  102: '自动完成7x7区域,雷自动标记。点击后选择格子使用',
  1001: '10秒内得分加倍。获得时自动使用',
  1002: '不限时间,踩雷时消耗一个护盾,可叠加。获得时自动使用',
}

const iconMap: Record<number, string> = {
  101: '/assets/prop101.png',
  102: '/assets/prop102.png',
  1001: '/assets/prop1001.png',
  1002: '/assets/prop1002.png',
}
</script>

<template>
  <div class="prop-bar">
    <div class="prop-bar-inner">
      <div class="prop-label">
        道具栏
      </div>
      <div
        v-for="prop in props.props" :key="prop.PropID" class="prop-item"
        :class="{
          'prop-active': isActiveProp(prop.PropID) && usingPropId === prop.PropID,
          'prop-disabled': cdRemaining > 0,
          'prop-auto': !isActiveProp(prop.PropID),
          'prop-countdown': isAutoActive(prop.PropID),
        }"
        :title="propDescs[prop.PropID] || ''"
        @click="isActiveProp(prop.PropID) ? $emit('useProp', prop.PropID) : null"
      >
        <el-badge :value="getBadgeValue(prop)" :offset="[-4, 0]">
          <img
            :src="iconMap[prop.PropID] || ''" class="prop-icon"
            :class="{ 'prop-icon-pulse': isAutoActive(prop.PropID) }"
          >
        </el-badge>
        <div class="prop-name">
          {{ propDisplayNames[prop.PropID] || prop.Name }}
        </div>
        <div v-if="isAutoActive(prop.PropID)" class="prop-timer-bar">
          <div class="prop-timer-fill" :style="{ width: `${getTimerPercent(prop.PropID)}%` }" />
        </div>
      </div>
      <span v-if="props.props.length === 0" class="prop-empty">暂无道具</span>
    </div>
  </div>
</template>

<style scoped>
.prop-bar {
  display: flex;
  justify-content: center;
}
.prop-bar-inner {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(255, 255, 255, 0.04);
  backdrop-filter: blur(10px);
  padding: 6px 20px;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.07);
}
.prop-label {
  font-size: 12px;
  color: #888;
  margin-right: 6px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 1px;
}
.prop-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 8px;
  transition: all 0.2s;
  border: 1.5px solid transparent;
}
.prop-item:hover {
  background: rgba(255, 255, 255, 0.08);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}
.prop-active {
  border-color: #fbbf24;
  background: rgba(251, 191, 36, 0.15);
  box-shadow: 0 0 16px rgba(251, 191, 36, 0.2);
}
.prop-disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.prop-auto {
  border-style: dashed;
  border-color: rgba(255, 255, 255, 0.1);
}
.prop-icon {
  width: 40px;
  height: 40px;
  object-fit: contain;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.3));
}
.prop-name {
  font-size: 11px;
  color: #c0c0c0;
  margin-top: 3px;
  white-space: nowrap;
  font-weight: 500;
}
.prop-empty {
  font-size: 12px;
  color: #666;
  padding: 0 10px;
  font-style: italic;
}
.prop-countdown {
  border-color: rgba(96, 165, 250, 0.5) !important;
  border-style: solid !important;
  background: rgba(96, 165, 250, 0.08);
}
.prop-icon-pulse {
  animation: icon-pulse 0.8s ease-in-out infinite alternate;
}
@keyframes icon-pulse {
  from {
    filter: drop-shadow(0 0 3px rgba(96, 165, 250, 0.4));
  }
  to {
    filter: drop-shadow(0 0 10px rgba(96, 165, 250, 0.8));
  }
}
.prop-timer-bar {
  width: 100%;
  height: 3px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 2px;
  margin-top: 4px;
  overflow: hidden;
}
.prop-timer-fill {
  height: 100%;
  background: linear-gradient(90deg, #60a5fa, #34d399);
  border-radius: 2px;
  transition: width 0.1s linear;
}
</style>
