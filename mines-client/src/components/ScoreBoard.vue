<script setup lang="ts">
import type { ScoreBoard as ScoreBoardType } from '@/types'
import { computed } from 'vue'

const props = defineProps<{ scoreBoard: ScoreBoardType }>()

const sortedEntries = computed(() => {
  return Object.entries(props.scoreBoard)
    .map(([name, score]) => ({ name, score }))
    .sort((a, b) => b.score - a.score)
})

function getRankClass(idx: number): string {
  if (idx === 0)
    return 'rank-1'
  if (idx === 1)
    return 'rank-2'
  if (idx === 2)
    return 'rank-3'
  return ''
}

function getRankIcon(idx: number): string {
  if (idx === 0)
    return '🥇'
  if (idx === 1)
    return '🥈'
  if (idx === 2)
    return '🥉'
  return String(idx + 1)
}
</script>

<template>
  <div class="scoreboard">
    <div class="scoreboard-title">
      <span class="title-icon">🏆</span> 玩家排行
    </div>
    <div
      v-for="(entry, idx) in sortedEntries" :key="entry.name"
      class="score-row" :class="getRankClass(idx)"
    >
      <div class="rank-num">
        {{ getRankIcon(idx) }}
      </div>
      <div class="player-info">
        <div class="player-name">
          {{ entry.name }}
        </div>
      </div>
      <div class="player-score">
        {{ entry.score }}分
      </div>
    </div>
  </div>
</template>

<style scoped>
.scoreboard {
  background: rgba(255, 255, 255, 0.04);
  backdrop-filter: blur(16px);
  border-radius: 14px;
  padding: 12px 8px;
  border: 1px solid rgba(255, 255, 255, 0.07);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
}
.scoreboard-title {
  font-weight: 700;
  font-size: 14px;
  color: #c0c0c0;
  margin-bottom: 10px;
  text-align: center;
  letter-spacing: 2px;
}
.title-icon {
  margin-right: 4px;
}
.score-row {
  display: flex;
  align-items: center;
  padding: 6px 8px;
  border-radius: 8px;
  margin-bottom: 3px;
  gap: 8px;
  font-size: 13px;
  transition:
    background 0.2s,
    transform 0.15s;
}
.score-row:hover {
  background: rgba(255, 255, 255, 0.05);
  transform: translateX(2px);
}
.rank-1 {
  background: rgba(255, 200, 40, 0.12);
  border: 1px solid rgba(255, 200, 40, 0.18);
}
.rank-2 {
  background: rgba(180, 180, 200, 0.08);
  border: 1px solid rgba(180, 180, 200, 0.1);
}
.rank-3 {
  background: rgba(210, 140, 80, 0.08);
  border: 1px solid rgba(210, 140, 80, 0.1);
}
.rank-num {
  width: 24px;
  text-align: center;
  font-size: 14px;
  flex-shrink: 0;
}
.player-info {
  flex: 1;
  min-width: 0;
}
.player-name {
  font-weight: 600;
  color: #e0e0e0;
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.player-score {
  color: #60a5fa;
  font-weight: 700;
  font-size: 13px;
  flex-shrink: 0;
}
</style>
