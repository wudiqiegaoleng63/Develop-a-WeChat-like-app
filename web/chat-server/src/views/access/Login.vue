<template>
  <div class="login-wrap">
    <div class="login-window" :style="{ boxShadow: 'var(--el-box-shadow-dark)' }">
      <h2 class="login-item">登录</h2>
      <el-form :model="loginData" label-width="70px">
        <el-form-item
          prop="telephone"
          label="账号"
          :rules="[{ required: true, message: '此项为必填项', trigger: 'blur' }]"
        >
          <el-input v-model="loginData.telephone" />
        </el-form-item>
        <el-form-item
          prop="password"
          label="密码"
          :rules="[{ required: true, message: '此项为必填项', trigger: 'blur' }]"
        >
          <el-input type="password" v-model="loginData.password" />
        </el-form-item>
      </el-form>
      <div class="login-button-container">
        <el-button type="primary" class="login-btn" @click="handleLogin">登录</el-button>
      </div>
      <div class="go-register-button-container">
        <button class="go-register-btn" @click="handleRegister">注册</button>
        <button class="go-sms-btn" @click="handleSmsLogin">验证码登录</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive } from 'vue'
import { useStore } from 'vuex'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const store = useStore()
const router = useRouter()

const loginData = reactive({
  telephone: '',
  password: ''
})

const checkTelephoneValid = () => {
  const regex = /^1([38][0-9]|14[579]|5[^4]|16[6]|7[1-35-8]|9[189])\d{8}$/
  return regex.test(loginData.telephone)
}

const handleLogin = async () => {
  try {
    if (!loginData.telephone || !loginData.password) {
      ElMessage.error('请填写完整登录信息。')
      return
    }
    if (!checkTelephoneValid()) {
      ElMessage.error('请输入有效的手机号码。')
      return
    }
    const response = await axios.post(store.state.backendUrl + '/login', loginData)
    console.log(response)
    if (response.data.code == 200) {
      if (response.data.data.status == 1) {
        ElMessage.error('该账号已被封禁，请联系管理员。')
        return
      }
      ElMessage.success(response.data.message)
      if (!response.data.data.avatar.startsWith('http')) {
        response.data.data.avatar = store.state.backendUrl + response.data.data.avatar
      }
      store.commit('setUserInfo', response.data.data)
      router.push('/sessionlist')
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('登录失败: ' + error.message)
  }
}

const handleRegister = () => {
  router.push('/register')
}

const handleSmsLogin = () => {
  router.push('/smslogin')
}
</script>

<style scoped>
.login-wrap {
  width: 100%;
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background-color: #f5f5f5;
}

.login-window {
  width: 400px;
  padding: 30px;
  background-color: #fff;
  border-radius: 8px;
}

.login-item {
  text-align: center;
  margin-bottom: 20px;
}

.login-button-container {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

.login-btn {
  width: 200px;
}

.go-register-button-container {
  display: flex;
  justify-content: space-between;
  margin-top: 20px;
}

.go-register-btn,
.go-sms-btn {
  background: none;
  border: none;
  color: #409eff;
  cursor: pointer;
  font-size: 14px;
}

.go-register-btn:hover,
.go-sms-btn:hover {
  color: #66b1ff;
}
</style>