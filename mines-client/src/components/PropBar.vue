<script lang="ts" setup>
import type { PropBarUpdate, PropSlot } from "@/types";
import { computed, onMounted, onUnmounted, ref } from "vue";

const props = defineProps<{
  propState: PropBarUpdate;
  activeProp: number | null;
}>();

const emit = defineEmits<{
  "use-prop": [propId: number];
  "update:activeProp": [propId: number | null];
}>();

const now = ref(Date.now());
let timer: number | null = null;

onMounted(() => {
  timer = window.setInterval(() => {
    now.value = Date.now();
  }, 100);
});

onUnmounted(() => {
  if (timer) {
    clearInterval(timer);
    timer = null;
  }
});

interface UnifiedProp {
  id: number;
  name: string;
  count: number;
  isActive: boolean;
}

const unifiedProps = computed<UnifiedProp[]>(() => {
  const list: UnifiedProp[] = [];

  for (const slot of props.propState.Inventory) {
    if (slot.PropID === 1002) continue; // shield handled separately via ShieldCount
    list.push({
      id: slot.PropID,
      name: getDisplayName(slot),
      count: slot.Count,
      isActive: slot.PropID === 101 || slot.PropID === 102,
    });
  }

  // Shield from ShieldCount
  if (props.propState.ShieldCount > 0) {
    list.push({
      id: 1002,
      name: "护盾",
      count: props.propState.ShieldCount,
      isActive: false,
    });
  }

  return list;
});

const iconMap: Record<number, string> = {
  101: "/src/assets/prop101.png",
  102: "/src/assets/prop102.png",
  1001: "/src/assets/prop1001.png",
  1002: "/src/assets/prop1002.png",
};

function getDisplayName(slot: PropSlot): string {
  const names: Record<number, string> = {
    101: "探测仪",
    102: "雷之奥义",
    1001: "双倍",
    1002: "护盾",
  };
  return names[slot.PropID] || slot.Name || `道具#${slot.PropID}`;
}

function getPropDesc(prop: UnifiedProp): string {
  const descs: Record<number, string> = {
    101: "显示5x5范围的雷，点击后选择格子使用 (快捷键 D)",
    102: "自动完成7x7区域，雷自动标记，安全格自动打开 (快捷键 X)",
    1001: "获得时自动使用，10秒内得分加倍，可叠加",
    1002: "踩雷或标记错误时消耗一个护盾，可叠加",
  };
  return descs[prop.id] || "";
}

function getBadgeValue(prop: UnifiedProp): string | number {
  if (prop.id === 1001 && props.propState.DoubleScoreActive) {
    return `${props.propState.DoubleScoreRemaining}s`;
  }
  return prop.count;
}

function isDoubleScoreActive(): boolean {
  return props.propState.DoubleScoreActive;
}

function onPropClick(prop: UnifiedProp) {
  if (!prop.isActive) return;
  if (props.activeProp === prop.id) {
    emit("update:activeProp", null);
  } else {
    emit("update:activeProp", prop.id);
  }
}
</script>

<template>
  <div class="prop-bar">
    <div class="prop-bar-inner">
      <span class="prop-label">道具</span>
      <div
        v-for="prop in unifiedProps"
        :key="prop.id"
        class="prop-item"
        :class="{
          'prop-active': prop.isActive && activeProp === prop.id,
          'prop-auto': !prop.isActive,
          'prop-countdown': prop.id === 1001 && isDoubleScoreActive(),
        }"
        :title="getPropDesc(prop)"
        @click="onPropClick(prop)"
      >
        <el-badge :value="getBadgeValue(prop)" :offset="[-4, 0]">
          <img :src="iconMap[prop.id]" class="prop-icon">
        </el-badge>
        <div class="prop-name">
          {{ prop.name }}
        </div>
        <div
          v-if="prop.id === 1001 && isDoubleScoreActive()"
          class="prop-timer-bar"
        >
          <div class="prop-timer-fill" />
        </div>
      </div>
      <span v-if="unifiedProps.length === 0" class="prop-empty">暂无道具</span>
    </div>
  </div>
</template>

<style scoped>
.prop-bar {
  display: flex;
  justify-content: center;
  margin-bottom: 8px;
}
.prop-bar-inner {
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(255, 255, 255, 0.04);
  backdrop-filter: blur(10px);
  padding: 6px 16px;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.07);
}
html:not(.dark) .prop-bar-inner {
  background: rgba(255, 255, 255, 0.85);
  border: 1px solid rgba(0, 0, 0, 0.08);
}
.prop-label {
  font-size: 12px;
  color: #888;
  margin-right: 4px;
  font-weight: 600;
}
html:not(.dark) .prop-label {
  color: #64748b;
}
.prop-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 8px;
  transition: all 0.2s;
  border: 1.5px solid transparent;
}
.prop-item:hover {
  background: rgba(255, 255, 255, 0.08);
}
html:not(.dark) .prop-item:hover {
  background: rgba(0, 0, 0, 0.05);
}
.prop-active {
  border-color: #fbbf24;
  background: rgba(251, 191, 36, 0.15);
}
.prop-auto {
  border-style: dashed;
  border-color: rgba(255, 255, 255, 0.1);
}
html:not(.dark) .prop-auto {
  border-color: rgba(0, 0, 0, 0.15);
}
.prop-countdown {
  border-color: rgba(96, 165, 250, 0.5) !important;
  border-style: solid !important;
  background: rgba(96, 165, 250, 0.08);
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
}
html:not(.dark) .prop-name {
  color: #475569;
}
.prop-empty {
  font-size: 12px;
  color: #666;
}
html:not(.dark) .prop-empty {
  color: #94a3b8;
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
  width: 100%;
}
</style>
