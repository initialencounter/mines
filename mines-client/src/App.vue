<script lang="ts" setup>
import { ref } from "vue";
import { useDark, useToggle } from "@vueuse/core";
import { Moon, Sunny } from '@element-plus/icons-vue'
import Login from '@/components/Login.vue';
import Board from '@/components/Board.vue';
import Footer from "@/components/Footer.vue";

const jwt = localStorage.getItem('jwt')?.replace('20240704', '')

const isExpired = (jwt: string | undefined) => {
  if (!jwt || jwt == 'undefined') return true
  const payload = JSON.parse(atob(jwt.split('.')[1]))
  if (payload.exp === undefined) return true
  return Date.now() > payload.exp * 1000
}

const showLogin = ref<boolean>(isExpired(jwt))
const flagMode = ref(false)
const boardRef = ref<InstanceType<typeof Board>>()

const isDark = useDark();
const toggleDark = useToggle(isDark);

function logout() {
  localStorage.removeItem("jwt");
  localStorage.removeItem("userId");
  showLogin.value = true;
}

function reset() {
  boardRef.value?.reset();
}
</script>

<template>
  <div class="app-shell">
    <div v-if="!showLogin" class="app-header">
      <span class="app-title">Mines Client</span>
      <div class="header-actions">
        <el-button
          size="small"
          :style="{ background: flagMode ? '#5282b8' : '#5c8f4b' }"
          @click="flagMode = !flagMode"
        >
          {{ flagMode ? "标记" : "挖开" }}模式
        </el-button>
        <el-button size="small" plain @click="reset">重置</el-button>
        <el-button type="danger" size="small" plain @click="logout">退出登录</el-button>
        <div class="icon-btn" @click="toggleDark()">
          <el-icon v-if="isDark"><Moon /></el-icon>
          <el-icon v-else><Sunny /></el-icon>
        </div>
      </div>
    </div>
    <el-container>
      <el-main>
        <Board v-if="!showLogin" ref="boardRef" :flag-mode="flagMode" />
        <Login v-if="showLogin" v-model="showLogin" />
      </el-main>
      <el-footer>
        <Footer></Footer>
      </el-footer>
    </el-container>
  </div>
</template>

<style>
.icon-btn {
  cursor: pointer;
  width: 1.4rem;
  height: 1.4rem;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.5);
  transition: color 0.2s, transform 0.2s;
}

.icon-btn:hover {
  color: rgba(255, 255, 255, 0.85);
  transform: scale(1.15);
}

html:not(.dark) .icon-btn {
  color: #64748b;
}

html:not(.dark) .icon-btn:hover {
  color: #334155;
}
</style>
