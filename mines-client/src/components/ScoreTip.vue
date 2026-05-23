<script lang="ts" setup>
import {ref} from "vue";

const scoreTipList = ref<{
  name: string
}[]>([])
const tips = (score: number, doubleScore = false) => {
  if (score === 0) return
  const displayScore = doubleScore ? score : score
  const tipText = '积分' + (displayScore > 0 ? '+' + displayScore : displayScore) + (doubleScore ? ' x2' : '')
  if(scoreTipList.value.length > 10) scoreTipList.value=[]
  scoreTipList.value.push({name: tipText})
}

defineExpose({
  tips
})

</script>

<template>
  <div>
    <div v-for="(item) of scoreTipList">
      <div class="animated-div" :class="{ 'double-tip': item.name.includes('x2') }">
        {{ item.name }}
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes moveAndFade {
  from {
    transform: translateY(0);
    opacity: 1;
  }
  to {
    transform: translateY(-20px);
    opacity: 0;
  }
}

.animated-div {
  position: absolute;
  top: 0;
  color: #2b8a2d;
  width: 10rem;
  opacity: 0;
  animation: moveAndFade 2s ease-out;
}
.double-tip {
  color: #fbbf24;
  font-size: 17px;
  text-shadow: 0 0 14px rgba(251, 191, 36, 0.6);
}

</style>
