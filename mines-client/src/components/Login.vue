<template>
  <div class="login-form">
    <template v-if="true">
      <div class="el-input__wrapper">
        <input v-model="shared.userId" class="el-input__inner" placeholder="用户名"></input>
        <svg class="k-icon" viewBox="0 0 448 512" xmlns="http://www.w3.org/2000/svg">
          <path
              d="M224 256c70.7 0 128-57.3 128-128S294.7 0 224 0 96 57.3 96 128s57.3 128 128 128zm89.6 32h-16.7c-22.2 10.2-46.9 16-72.9 16s-50.6-5.8-72.9-16h-16.7C60.2 288 0 348.2 0 422.4V464c0 26.5 21.5 48 48 48h352c26.5 0 48-21.5 48-48v-41.6c0-74.2-60.2-134.4-134.4-134.4z"
              fill="currentColor"></path>
        </svg>
      </div>

      <div class="el-input__wrapper">
        <input v-model="shared.password"
               class="el-input__inner"
               placeholder="密码"
               @keypress.enter.stop="login"></input>
        <svg class="k-icon" viewBox="0 0 448 512" xmlns="http://www.w3.org/2000/svg">
          <path
              d="M400 224h-24v-72C376 68.2 307.8 0 224 0S72 68.2 72 152v72H48c-26.5 0-48 21.5-48 48v192c0 26.5 21.5 48 48 48h352c26.5 0 48-21.5 48-48V272c0-26.5-21.5-48-48-48zm-104 0H152v-72c0-39.7 32.3-72 72-72s72 32.3 72 72v72z"
              fill="currentColor"></path>
        </svg>
      </div>
      <ElButton class="ElButton" @click="login">登录</ElButton>
    </template>
  </div>
</template>

<script lang="ts" setup>
import {ElButton, ElMessage} from 'element-plus'
import {ref, defineModel} from 'vue'
import axios from "axios";
import {useRouter} from "vue-router";

const router = useRouter();
let host = window.location.hostname
let port = window.location.port

const loggedIn = defineModel<boolean>({ required: true })

const shared = ref({
  showPass: false,
  userId: '',
  password: '',
})

async function login() {
  const {userId, password} = shared.value
  try {
    let config = {
      method: 'post',
      url: `http://${host}:${port}/login?user=${userId}&pass=${password}`,
      headers: {
        'Content-Type': 'application/xml',
        'Accept': '*/*',
      }
    };
    const response = await axios(config);
    const token = response.data.token;
    loggedIn.value = true;
    localStorage.setItem('jwt', token);
  } catch (error) {
    console.log(error)
    ElMessage.error({
      message: 'Invalid username or password',
      type: 'error',
      customClass: 'customClass',
      offset: 100
    });
  }
}

</script>

<style>

.login-form {
  width: 20rem;
  height: 12rem;
  border-radius: 5px;
  border: lightgrey 1px solid;
  justify-content: center;
  align-items: center;
  display: block;
}

h1 {
  font-size: 1.5rem;
  margin: 2.5rem auto;
  cursor: default;
}

.el-input__wrapper {
  top: 2rem;
  position: relative;
  width: 100%;
  display: flex;
  align-items: center;
  border-radius: 5px;
}

.el-input__inner {
  position: relative;
  display: block;
  margin: 0.5rem;
  height: 2rem;
  width: 8rem;
  left: 20%;
  --el-input-inner-height: calc(var(--el-input-height, 32px) - 2px);
  flex-grow: 1;
  color: var(--el-input-text-color, var(--el-text-color-regular));
  font-size: inherit;
  line-height: var(--el-input-inner-height);
  padding: 0;
  outline: 0;
  border: none;
  background: 0 0;
  box-sizing: border-box;
}

.el-input__wrapper:hover {
  border: #b1b1b1 2px solid;
  border-radius: 5px;
}

.control {
  margin: 2.5rem auto;
}

.k-icon {
  position: absolute;
  left: 2rem;
  top: 1rem;
  margin: 0.5rem;
  transform: translateY(-50%);
  height: 1.2rem;
  color: var(--el-input-placeholder-color);
}

.ElButton {
  position: absolute;
  top: 75%;
  left: 40%;
  font-size: 14px;
  font-weight: bolder;
  line-height: 20px;
  -webkit-appearance: none;
  -moz-appearance: none;
  appearance: none;
  -webkit-user-select: none;
  user-select: none;
  border-radius: .4em;
  cursor: pointer;
  padding: .4em 1em;
  display: inline-block;
  background: #595959;
}

.ElButton:hover {
  background-color: rgba(151, 150, 150, 0.23);
}

.customClass {
  color: red;
  height: 10px;
  width: 10px;
  top: 50px !important; /* 距离顶部 50 像素 */
  left: 50% !important; /* 水平居中 */
}


</style>