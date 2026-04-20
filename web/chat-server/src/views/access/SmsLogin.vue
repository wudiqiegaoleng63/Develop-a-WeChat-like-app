<template>
  <div class="login-wrap">
    <div class="login-window" :style="{ boxShadow: 'var(--el-box-shadow-dark)' }">
      <h2 class="login-item">短信验证码登录</h2>
      <el-form :model="loginData" label-width="70px">
        <el-form-item
          prop="telephone"
          label="账号"
          :rules="[{ required: true, message: '此项为必填项', trigger: 'blur' }]"
        >
          <el-input v-model="loginData.telephone" />
        </el-form-item>
        <el-form-item
          prop="sms_code"
          label="验证码"
          :rules="[{ required: true, message: '此项为必填项', trigger: 'blur' }]"
        >
          <el-input v-model="loginData.sms_code" style="max-width: 200px">
            <template #append>
              <el-button @click="sendSmsCode" style="background-color: rgb(229, 132, 132); color: #ffffff">
                点击发送
              </el-button>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
      <div class="login-button-container">
        <el-button type="primary" class="login-btn" @click="handleSmsLogin">登录</el-button>
      </div>
      <div class="go-register-button-container">
        <button class="go-register-btn" @click="handleRegister">注册</button>
        <button class="go-password-btn" @click="handleLogin">密码登录</button>
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
  sms_code: ''
})

const checkTelephoneValid = () => {
  const regex = /^1([38][0-9]|14[579]|5[^4]|16[6]|7[1-35-8]|9[189])\d{8}$/
  return regex.test(loginData.telephone)
}

const sendSmsCode = async () => {
  if (!loginData.telephone) {
    ElMessage.error('请输入手机号。')
    return
  }
  if (!checkTelephoneValid()) {
    ElMessage.error('请输入有效的手机号码。')
    return
  }
  const req = { telephone: loginData.telephone }
  try {
    const rsp = await axios.post(store.state.backendUrl + '/user/sendSmsCode', req)
    console.log(rsp)
    if (rsp.data.code == 200) {
      ElMessage.success(rsp.data.message)
    } else if (rsp.data.code == 400) {
      ElMessage.warning(rsp.data.message)
    } else {
      ElMessage.error(rsp.data.message)
    }
  } catch (error) {
    ElMessage.error('发送验证码失败: ' + error.message)
  }
}

const handleSmsLogin = async () => {
  try {
    if (!loginData.telephone || !loginData.sms_code) {
      ElMessage.error('请填写完整登录信息。')
      return
    }
    if (!checkTelephoneValid()) {
      ElMessage.error('请输入有效的手机号码。')
      return
    }
    const response = await axios.post(store.state.backendUrl + '/user/smsLogin', loginData)
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

const handleLogin = () => {
  router.push('/login')
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
.go-password-btn {
  background: none;
  border: none;
  color: #409eff;
  cursor: pointer;
  font-size: 14px;
}

.go-register-btn:hover,
.go-password-btn:hover {
  color: #66b1ff;
}
</style>