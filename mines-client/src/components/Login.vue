<template>
  <div class="login-wrapper">
    <el-form ref="ruleFormRef"
             v-loading="loading"
             :model="ruleForm"
             :rules="rules"
             :size="formSize"
             label-width="auto"
             status-icon
             class="login-form"
    >
      <el-form-item>
        <h1 class="login-title">{{ modes.register ? "注册模式" : (modes.login ? "亲，请登录" : "找回密码，若忘记用户名，请联系管理员！") }}</h1>
      </el-form-item>
      <el-form-item label="用户名" prop="userName">
        <el-input v-model="ruleForm.userName"/>
      </el-form-item>
      <el-form-item v-if="!modes.verify" :label="(modes.reset?'新密码':'密码')" prop="password">
        <el-input v-model="ruleForm.password" type="password"/>
      </el-form-item>
      <el-form-item v-if="!modes.login && !modes.verify" label="确认密码" prop="checkPass">
        <el-input v-model="ruleForm.checkPass" type="password"/>
      </el-form-item>
      <el-form-item v-if="modes.register || modes.verify" label="email" prop="email">
        <el-input v-model="ruleForm.email"/>
      </el-form-item>
      <el-form-item v-if="modes.reset" label="验证码" prop="code">
        <el-input v-model="ruleForm.code"/>
      </el-form-item>
      <el-form-item>
        <div class="config-button">
          <el-button v-if="modes.verify" type="primary" @click="submitForm(ruleFormRef)">发送验证码</el-button>
          <el-button v-if="!modes.verify" type="primary" @click="submitForm(ruleFormRef)">
            {{ modes.register ? "注册" : (modes.login ? "登录" : "重置密码") }}
          </el-button>
          <el-button type="danger" @click="resetForm(ruleFormRef)">清空</el-button>
          <el-button type="info" @click="visitorModeLogin">游客</el-button>
          <p></p>
          <el-button v-if="!modes.login" type="primary" @click="switchMode('login')">登录</el-button>
          <el-button v-if="!modes.register" type="primary" @click="switchMode('register')">注册</el-button>
          <el-button v-if="!modes.verify && !modes.reset" type="primary" @click="switchMode('verify')">忘记密码</el-button>
        </div>
      </el-form-item>
    </el-form>
  </div>
</template>


<script lang="ts" setup>
import {reactive, ref} from 'vue'
import axios from "axios";
import {host, port} from "@/utils";

import {
  type ComponentSize,
  ElMessage,
  ElMessageBox,
  type FormInstance,
  type FormRules
} from 'element-plus'

const modes = ref<{ login: boolean, register: boolean, reset: boolean, verify: boolean }>({login: true, register: false, reset: false, verify: false})
type Mode = 'login' | 'register' | 'reset' | 'verify'

const loading = ref(false)
const showLogin = defineModel<boolean>({required: true})

interface RuleForm {
  userName: string
  password: string
  checkPass: string
  email: string
  code: string
}

const formSize = ref<ComponentSize>('default')
const ruleFormRef = ref<FormInstance>()
const ruleForm = reactive<RuleForm>({
  userName: '',
  password: '',
  checkPass: '',
  email: '',
  code: '',
})

const validatePass = (rule: any, value: string, callback: any) => {
  if (value === '') {
    callback(new Error('请再次输入密码！'))
  } else if (value !== ruleForm.password) {
    callback(new Error('两次输入密码不一致！'))
  } else {
    callback()
  }
}

const rules = reactive<FormRules<RuleForm>>({
  userName: [
    {required: true, message: '请输入用户名！', trigger: 'blur'},
    {min: 3, max: 16, message: '长度应该为 3 - 16', trigger: 'blur'},
  ],
  password: [
    {required: true, message: '请输入密码！', trigger: 'blur'},
    {min: 1, max: 16, message: '长度应该为 1 - 16', trigger: 'blur'},
  ],
  checkPass: [
    {required: true, validator: validatePass, trigger: 'blur'},
    {min: 1, max: 16, message: '长度应该为 1 - 16', trigger: 'blur'},
  ],
  email: [
    {required: true, message: '请输入邮箱！', trigger: 'blur'},
    {min: 5, max: 64, message: '长度应该为 5 - 64', trigger: 'blur'},
  ],
  code: [
    {required: true, message: '请输入验证码！', trigger: 'blur'},
    {min: 6, max: 6, message: '长度应该为 6', trigger: 'blur'},
  ],
})

const submitForm = async (formEl: FormInstance | undefined) => {
  if (!formEl) return
  await formEl.validate((valid) => {
    if (valid) {
      let mode = modes.value.register ? 'register' : (modes.value.login ? 'login' : (modes.value.verify ? 'verify' : 'reset'))
      postForm(mode as Mode)
    } else {
      ElMessage.error('error submit!')
    }
  })
}

const resetForm = (formEl: FormInstance | undefined) => {
  if (!formEl) return
  formEl.resetFields()
}

async function postForm(mode: Mode) {
  loading.value = true
  const {userName, password, email, code} = ruleForm
  try {
    let config = {
      method: 'post',
      url: `//${host}:${port}/${mode}`,
      headers: {
        'Content-Type': 'application/json',
        'Accept': '*/*',
      },
      data: {
        user: userName,
        pass: password,
        code: code,
        email: email
      }
    };
    const response = await axios(config);
    loading.value = false
    let messageText = mode === 'login' ? `欢迎回来，${userName}！` : (mode === 'register' ? `${userName}，欢迎加入！` : response.data)
    ElMessage({
      message: messageText,
      type: 'success',
      plain: true,
    })
    if (mode === 'verify') {
      switchMode('reset')
    }
    if (mode === 'reset') {
      switchMode('login')
    }
    if(mode === 'login'|| mode === 'register') {
      const token = response.data.token;
      localStorage.setItem('jwt', '20240704' + token);
      localStorage.setItem('userId', response.data.id);
      localStorage.setItem('userName', userName)
      showLogin.value = false
    }
  } catch (error: any) {
    loading.value = false
    await ElMessageBox.alert(error.response.data, 'oops!', {
      confirmButtonText: 'OK',
      type: 'error',
    });
  }
}

const switchMode = (mode: Mode) => {
  for (const key of Object.keys(modes.value) as Mode[]) {
    modes.value[key] = (key === mode);
  }
}

document.onkeydown = (e) => {
  if (e.key === 'Enter') {
    submitForm(ruleFormRef.value)
  }
}

const visitorModeLogin = () => {
  ruleForm.userName = 'visitor'
  ruleForm.password = 'visitor'
  postForm('login')
}

</script>

<style scoped>
.login-wrapper {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 80vh;
}

.login-form {
  background: rgba(255, 255, 255, 0.04);
  backdrop-filter: blur(16px);
  border-radius: 14px;
  padding: 2rem;
  border: 1px solid rgba(255, 255, 255, 0.07);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
  max-width: 28rem;
  width: 100%;
}

html:not(.dark) .login-form {
  background: rgba(255, 255, 255, 0.85);
  border: 1px solid rgba(0, 0, 0, 0.08);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
}

.login-title {
  font-size: 1.3rem;
  font-weight: 700;
  color: #d0d0d0;
  text-align: center;
}

html:not(.dark) .login-title {
  color: #334155;
}

.config-button {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
}
</style>
