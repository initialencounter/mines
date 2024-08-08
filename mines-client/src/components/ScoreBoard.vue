<script setup lang="ts">
import type { ScoreBoard as ScoreBoardType } from "@/types";

interface ScoreEntry {
  name: string;
  score: number;
}

defineProps<{
  scoreBoard: ScoreBoardType;
}>();
function convertScoreBoardToArray(scoreBoard: ScoreBoardType): ScoreEntry[] {
  if (!scoreBoard) return [];
  return Object.entries(scoreBoard).map(([key, value]) => ({
    name: key,
    score: value,
  }));
}
</script>

<template>
  <div>
    <div
      v-for="(cell, index) in convertScoreBoardToArray(scoreBoard)"
      :key="index"
      class="rankCell"
    >
      <div style="width: 2rem">{{ index + 1 }}</div>
      <div style="width: 5rem">{{ cell.name }}</div>
      <div style="width: 0.5rem">{{ cell.score }}</div>
    </div>
  </div>
</template>

<style scoped>
.rankCell {
  display: flex;
}
</style>
