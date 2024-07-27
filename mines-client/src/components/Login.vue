<template>
  <div class="common-layout">
    <el-container>
      <el-header></el-header>
      <el-main>
        <el-form v-if="!registerMode" ref="ruleFormRef"
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
            <h1>亲，请登录</h1>
          </el-form-item>
          <el-form-item label="用户名" prop="userName">
            <el-input v-model="ruleForm.userName"/>
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input v-model="ruleForm.password" type="password"/>
          </el-form-item>
          <el-form-item>
            <div class="config-button">
              <el-button type="primary" @click="submitForm(ruleFormRef)">登录</el-button>
              <el-button type="danger" @click="resetForm(ruleFormRef)">清除输入</el-button>
              <el-button type="info">忘记密码</el-button>
            </div>
          </el-form-item>
          <el-form-item>
            <div class="config-button">
              <el-button type="primary" @click="switchMode">没有账号！点我注册！</el-button>
            </div>
          </el-form-item>
        </el-form>
        <el-form v-if="registerMode" ref="ruleFormRef"
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
            <h1>注册模式</h1>
          </el-form-item>
          <el-form-item label="用户名" prop="userName">
            <el-input v-model="ruleForm.userName"/>
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input v-model="ruleForm.password" type="password"/>
          </el-form-item>
          <el-form-item label="确认密码" prop="checkPass">
            <el-input v-model="ruleForm.checkPass" type="password"/>
          </el-form-item>
          <el-form-item label="email" prop="email">
            <el-input v-model="ruleForm.email"/>
          </el-form-item>
          <el-form-item>
            <div class="config-button">
              <el-button type="primary" @click="submitForm(ruleFormRef)">注册</el-button>
              <el-button type="danger" @click="resetForm(ruleFormRef)">清除输入</el-button>
            </div>
          </el-form-item>
          <el-form-item>
            <div class="config-button">
              <el-button type="primary" @click="login">已有账号！点我登录！</el-button>
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

import {type ComponentSize, ElMessage, ElNotification, type FormInstance, type FormRules} from 'element-plus'

const registerMode = ref(false)
const loading = ref(false)
const showLogin = defineModel<boolean>({required: true})

interface RuleForm {
  userName: string
  password: string
  checkPass: string
  email: string
}

const formSize = ref<ComponentSize>('default')
const ruleFormRef = ref<FormInstance>()
const ruleForm = reactive<RuleForm>({
  userName: '',
  password: '',
  checkPass: '',
  email: '',
})

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
    {required: true, message: '请输入再次密码！', trigger: 'blur'},
    {min: 1, max: 16, message: '长度应该为 1 - 16', trigger: 'blur'},
  ],
  email: [
    {required: true, message: '请输入邮箱！', trigger: 'blur'},
    {min: 5, max: 64, message: '长度应该为 5 - 64', trigger: 'blur'},
  ],
})

const submitForm = async (formEl: FormInstance | undefined) => {
  if (!formEl) return
  await formEl.validate((valid) => {
    if (valid) {
      if (registerMode.value)
        register()
      else {
        login()
      }
    } else {
      ElMessage.error('error submit!')
    }
  })
}

const resetForm = (formEl: FormInstance | undefined) => {
  if (!formEl) return
  formEl.resetFields()
}

async function login() {
  loading.value = true
  const {userName: userName, password} = ruleForm
  try {
    let config = {
      method: 'post',
      url: `http://${host}:${port}/login`,
      headers: {
        'Content-Type': 'application/json',
        'Accept': '*/*',
      },
      data: {
        user: userName,
        pass: password,
      }
    };
    const response = await axios(config);
    loading.value = false
    ElMessage({
      message: `${userName}，欢迎回来！`,
      type: 'success',
      plain: true,
    })
    showLogin.value = false
    const token = response.data.token;
    localStorage.setItem('jwt', '20240704' + token);
    localStorage.setItem('userId', response.data.id);
    localStorage.setItem('userName', userName)
  } catch (error: any) {
    loading.value = false
    ElNotification({
      title: 'Error',
      message: error.response.data,
      type: 'error',
      duration: 0,
    })
  }
}

const register = async () => {
  loading.value = true
  const {userName, password, email} = ruleForm
  if (password !== ruleForm.checkPass) {
    ElMessage.error('两次密码不一致！')
    return
  }
  try {
    let config = {
      method: 'post',
      url: `http://${host}:${port}/register`,
      headers: {
        'Content-Type': 'application/json',
        'Accept': '*/*',
      },
      data: {
        user: userName,
        pass: password,
        email: email
      }
    };
    const response = await axios(config);
    loading.value = false
    ElMessage({
      message: `${userName}，欢迎加入！`,
      type: 'success',
      plain: true,
    })
    showLogin.value = false
    const token = response.data.token;
    localStorage.setItem('jwt', '20240704' + token);
    localStorage.setItem('userId', response.data.id);
    localStorage.setItem('userName', userName)
  } catch (error: any) {
    loading.value = false
    ElNotification({
      title: 'Error',
      message: error.response.data,
      type: 'error',
      duration: 0,
    })
  }
}
const switchMode = () => {
  registerMode.value = !registerMode.value
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