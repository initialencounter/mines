<script lang="ts" setup>
import {ref} from "vue";

const scoreTipList = ref<{
  name: string
}[]>([])
const tips = (score: number) => {
  console.log(score)
  if (score === 0) return
  const tipText = '积分' + (score > 0 ? '+' + score : score)
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
      <div class="animated-div">
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
  animation: moveAndFade 1s ease-out;
}

</style>