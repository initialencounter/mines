<script lang="ts" setup>
import { ElMessage } from 'element-plus'
import { ref } from 'vue'

const emit = defineEmits<{
  login: [data: { token: string; id: number; name: string }]
}>()

type Tab = 'login' | 'register' | 'forgot'
const activeTab = ref<Tab>('login')

// ========== 登录表单 ==========
const loginUser = ref('')
const loginPass = ref('')
const loginLoading = ref(false)

async function doLogin() {
  if (!loginUser.value || !loginPass.value) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loginLoading.value = true
  try {
    const resp = await fetch('/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user: loginUser.value, pass: loginPass.value }),
    })
    if (!resp.ok) {
      const msg = await resp.text()
      ElMessage.error(msg || '登录失败')
      return
    }
    const data = await resp.json()
    saveAndEmit(data)
  } catch {
    ElMessage.error('网络错误')
  } finally {
    loginLoading.value = false
  }
}

// ========== 注册表单 ==========
const regUser = ref('')
const regPass = ref('')
const regEmail = ref('')
const regLoading = ref(false)

async function doRegister() {
  if (!regUser.value || !regPass.value || !regEmail.value) {
    ElMessage.warning('请填写所有字段')
    return
  }
  regLoading.value = true
  try {
    const resp = await fetch('/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user: regUser.value, pass: regPass.value, email: regEmail.value }),
    })
    if (!resp.ok) {
      const msg = await resp.text()
      ElMessage.error(msg || '注册失败')
      return
    }
    const data = await resp.json()
    saveAndEmit(data)
  } catch {
    ElMessage.error('网络错误')
  } finally {
    regLoading.value = false
  }
}

// ========== 忘记密码 ==========
const forgotUser = ref('')
const forgotEmail = ref('')
const forgotCode = ref('')
const forgotNewPass = ref('')
const forgotStep = ref<'email' | 'reset'>('email')
const forgotLoading = ref(false)
const forgotSending = ref(false)

async function doSendCode() {
  if (!forgotUser.value || !forgotEmail.value) {
    ElMessage.warning('请输入用户名和邮箱')
    return
  }
  forgotSending.value = true
  try {
    const resp = await fetch('/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user: forgotUser.value, email: forgotEmail.value }),
    })
    if (!resp.ok) {
      const msg = await resp.text()
      ElMessage.error(msg || '发送失败')
      return
    }
    ElMessage.success('验证码已发送到邮箱')
    forgotStep.value = 'reset'
  } catch {
    ElMessage.error('网络错误')
  } finally {
    forgotSending.value = false
  }
}

async function doResetPassword() {
  if (!forgotUser.value || !forgotNewPass.value || !forgotCode.value) {
    ElMessage.warning('请填写所有字段')
    return
  }
  forgotLoading.value = true
  try {
    const resp = await fetch('/reset', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user: forgotUser.value, pass: forgotNewPass.value, code: forgotCode.value }),
    })
    if (!resp.ok) {
      const msg = await resp.text()
      ElMessage.error(msg || '重置失败')
      return
    }
    ElMessage.success('密码重置成功，请登录')
    // Switch to login tab and pre-fill username
    loginUser.value = forgotUser.value
    forgotStep.value = 'email'
    activeTab.value = 'login'
  } catch {
    ElMessage.error('网络错误')
  } finally {
    forgotLoading.value = false
  }
}

// ========== 游客登录 ==========
const guestLoading = ref(false)

async function doGuestLogin() {
  guestLoading.value = true
  try {
    const resp = await fetch('/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user: 'visitor', pass: 'visitor' }),
    })
    if (!resp.ok) {
      const msg = await resp.text()
      ElMessage.error(msg || '游客登录失败')
      return
    }
    const data = await resp.json()
    saveAndEmit(data)
  } catch {
    ElMessage.error('网络错误')
  } finally {
    guestLoading.value = false
  }
}

// ========== 保存 & 通知 ==========
function saveAndEmit(data: { token: string; id: number }) {
  localStorage.setItem('token', data.token)
  localStorage.setItem('uid', String(data.id))
  emit('login', { token: data.token, id: data.id, name: loginUser.value || regUser.value })
}
</script>

<template>
  <el-dialog :model-value="true" width="400px" :show-close="false" :close-on-click-modal="false" :close-on-press-escape="false" center>
    <template #header>
      <div class="dialog-tabs">
        <span class="tab" :class="{ active: activeTab === 'login' }" @click="activeTab = 'login'">登录</span>
        <span class="tab-divider">|</span>
        <span class="tab" :class="{ active: activeTab === 'register' }" @click="activeTab = 'register'">注册</span>
        <span class="tab-divider">|</span>
        <span class="tab" :class="{ active: activeTab === 'forgot' }" @click="activeTab = 'forgot'">忘记密码</span>
      </div>
    </template>

    <!-- ===== 登录 ===== -->
    <div v-if="activeTab === 'login'" class="form-body">
      <el-input v-model="loginUser" placeholder="用户名" size="large" @keyup.enter="doLogin" />
      <el-input v-model="loginPass" type="password" placeholder="密码" size="large" show-password @keyup.enter="doLogin" />
      <el-button type="primary" size="large" :loading="loginLoading" @click="doLogin" style="width: 100%">
        登录
      </el-button>
      <el-button size="large" :loading="guestLoading" @click="doGuestLogin" style="width: 100%">
        游客登录
      </el-button>
    </div>

    <!-- ===== 注册 ===== -->
    <div v-if="activeTab === 'register'" class="form-body">
      <el-input v-model="regUser" placeholder="用户名" size="large" @keyup.enter="doRegister" />
      <el-input v-model="regPass" type="password" placeholder="密码" size="large" show-password @keyup.enter="doRegister" />
      <el-input v-model="regEmail" placeholder="邮箱" size="large" @keyup.enter="doRegister" />
      <el-button type="primary" size="large" :loading="regLoading" @click="doRegister" style="width: 100%">
        注册
      </el-button>
    </div>

    <!-- ===== 忘记密码 ===== -->
    <div v-if="activeTab === 'forgot'" class="form-body">
      <template v-if="forgotStep === 'email'">
        <el-input v-model="forgotUser" placeholder="用户名" size="large" />
        <el-input v-model="forgotEmail" placeholder="注册邮箱" size="large" />
        <el-button type="warning" size="large" :loading="forgotSending" @click="doSendCode" style="width: 100%">
          发送验证码
        </el-button>
      </template>
      <template v-else>
        <el-input v-model="forgotCode" placeholder="验证码" size="large" />
        <el-input v-model="forgotNewPass" type="password" placeholder="新密码" size="large" show-password />
        <el-button type="primary" size="large" :loading="forgotLoading" @click="doResetPassword" style="width: 100%">
          重置密码
        </el-button>
        <el-button size="small" @click="forgotStep = 'email'" style="width: 100%">返回上一步</el-button>
      </template>
    </div>
  </el-dialog>
</template>

<style scoped>
.dialog-tabs {
  display: flex; justify-content: center; gap: 8px; align-items: center; width: 100%;
}
.tab {
  cursor: pointer; color: #888; font-size: 14px; padding: 4px 8px; transition: color 0.2s; user-select: none;
}
.tab:hover { color: #c0c0c0; }
.tab.active { color: #409eff; font-weight: 700; }
.tab-divider { color: #444; font-size: 12px; }
.form-body {
  display: flex; flex-direction: column; gap: 14px; padding: 8px 0;
}
</style>
