<script lang="ts" setup>
import Login from '@/components/Login.vue';
import Board from '@/components/Board.vue';
import {ref} from "vue";

const jwt = localStorage.getItem('jwt')?.replace('20240704', '')

const isExpired = (jwt: string | undefined) => {
  if (!jwt || jwt == 'undefined') return true
  const payload = JSON.parse(atob(jwt.split('.')[1]))
  if (payload.exp === undefined) return true
  return Date.now() > payload.exp * 1000
}

const showLogin = ref<boolean>(isExpired(jwt))

</script>
<template>
  <div class="common-layout">
    <el-container>
      <el-header></el-header>
      <el-main>
        <Board v-if="!showLogin" v-model="showLogin"></Board>
        <Login v-if="showLogin" v-model="showLogin"></Login>
      </el-main>
      <el-footer></el-footer>
    </el-container>
  </div>

</template>