<script setup lang="ts">
import type {ScoreBoard as ScoreBoardType} from "@/types";

interface ScoreEntry {
  name: string
  score: number
}

defineProps<{
  scoreBoard: ScoreBoardType
}>()
function convertScoreBoardToArray(scoreBoard: ScoreBoardType): ScoreEntry[] {
  if (!scoreBoard) return [];
  return Object.entries(scoreBoard).map(([key, value]) => ({
    name: key,
    score: value
  }));
}

</script>

<template>
  <el-table
      :header-cell-style="{color: '#ffffff',fontSize: '16px', fontWeight: 600, backgroundColor: 'rgb(68,39,120)',border: '1px solid #5151510',opacity: '1'}"
      :row-style="{color: '#000', fontSize: '14px', backgroundColor: '#bbb', padding: '4px'}"
      :data="convertScoreBoardToArray(scoreBoard)"
      :default-sort="{ prop: 'score', order: 'descending'}"
      style="width: 180px"
  >
    <el-table-column prop="name" label="Name" sortable width="100" />
    <el-table-column prop="score" label="Score" width="80" />
  </el-table>
</template>

<style scoped>

</style>
