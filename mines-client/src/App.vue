<script lang="ts" setup>
import {ref} from 'vue'
import Board from './components/Board.vue'
import Login from './components/Login.vue'

const loggedIn = ref<boolean>(false)
loggedIn.value = !!localStorage.getItem('jwt');

</script>

<template>
  <main class="container">
    <Board v-if="loggedIn"></Board>
    <Login v-if="!loggedIn" class="board" v-model="loggedIn"></Login>
  </main>
</template>

<style scoped>
.container {
  position: relative;
  place-items: center; /* 垂直居中 */
  height: 100vh; /* 父容器高度设为视口高度 */
}

.board {
  position: absolute;
  top: 20%;
  left: 20%;
  transform: translate(-50%, -50%); /* 通过 transform 平移实现居中 */
}
</style>
