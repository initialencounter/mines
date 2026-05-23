<script setup lang="ts">
import { ref } from 'vue'
import Board from '@/components/Board.vue'
import LoginDialog from '@/components/LoginDialog.vue'

const token = ref(localStorage.getItem('token') || '')
const isLoggedIn = ref(!!token.value)

function onLogin(data: { token: string; id: number; name: string }) {
  token.value = data.token
  isLoggedIn.value = true
}

function onLogout() {
  localStorage.removeItem('token')
  localStorage.removeItem('uid')
  token.value = ''
  isLoggedIn.value = false
}
</script>

<template>
  <LoginDialog v-if="!isLoggedIn" @login="onLogin" />
  <Board v-else :key="token" @logout="onLogout" />
</template>
