<template>
  <div class="common-layout">
    <el-container>
      <el-header></el-header>
      <el-main>
        <el-form ref="ruleFormRef"
                 v-loading="loading"
                 :model="ruleForm"
                 :rules="rules"
                 :size="formSize"
                 class="demo-ruleForm"
                 label-width="auto"
                 status-icon
                 style="max-width: 500px"
        >
          <el-form-item>
            <h1>{{ modes.register ? "注册模式" : (modes.login ? "亲，请登录" : "找回密码，若忘记用户名，请联系管理员！") }}</h1>
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
              <el-button type="danger" @click="resetForm(ruleFormRef)">清除输入</el-button>
              <el-button type="info" @click="visitorModeLogin">游客模式</el-button>
            </div>
          </el-form-item>
          <el-form-item>
            <div class="config-button">
              <el-button v-if="!modes.login" type="primary" @click="switchMode('login')">已有账号！点我登录！</el-button>
              <el-button v-if="!modes.register" type="primary" @click="switchMode('register')">没有账号！点我注册！</el-button>
              <el-button v-if="!modes.verify && !modes.reset" type="primary" @click="switchMode('verify')">忘记密码！点我找回！</el-button>
            </div>
          </el-form-item>
        </el-form>
      </el-main>
      <el-footer></el-footer>
    </el-container>
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
  ElNotification,
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
      url: `http://${host}:${port}/${mode}`,
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

<style>

.el-form {
  background: rgba(121, 187, 255, 0.18);
  padding: 5rem;
  margin: 5rem;
  border-radius: 0.8rem;
}

.el-item {
  margin: 1rem;
}

.config-button {
  left: 20%;
  display: flex;
  justify-content: center;
}
</style>