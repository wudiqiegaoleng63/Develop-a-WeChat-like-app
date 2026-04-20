<template>
  <div class="register-wrap">
    <div class="register-window" :style="{ boxShadow: 'var(--el-box-shadow-dark)' }">
      <h2 class="register-item">注册</h2>
      <el-form :model="registerData" label-width="70px">
        <el-form-item
          prop="nickname"
          label="昵称"
          :rules="[
            { required: true, message: '此项为必填项', trigger: 'blur' },
            { min: 3, max: 10, message: '昵称长度在 3 到 10 个字符', trigger: 'blur' }
          ]"
        >
          <el-input v-model="registerData.nickname" />
        </el-form-item>
        <el-form-item
          prop="telephone"
          label="账号"
          :rules="[{ required: true, message: '此项为必填项', trigger: 'blur' }]"
        >
          <el-input v-model="registerData.telephone" />
        </el-form-item>
        <el-form-item
          prop="password"
          label="密码"
          :rules="[{ required: true, message: '此项为必填项', trigger: 'blur' }]"
        >
          <el-input type="password" v-model="registerData.password" />
        </el-form-item>
        <el-form-item
          prop="sms_code"
          label="验证码"
          :rules="[{ required: true, message: '此项为必填项', trigger: 'blur' }]"
        >
          <el-input v-model="registerData.sms_code" style="max-width: 200px">
            <template #append>
              <el-button @click="sendSmsCode" style="background-color: rgb(229, 132, 132); color: #ffffff">
                点击发送
              </el-button>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
      <div class="register-button-container">
        <el-button type="primary" class="register-btn" @click="handleRegister">注册</el-button>
      </div>
      <div class="go-login-button-container">
        <button class="go-sms-login-btn" @click="handleSmsLogin">验证码登录</button>
        <button class="go-password-login-btn" @click="handleLogin">密码登录</button>
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

const registerData = reactive({
  nickname: '',
  telephone: '',
  password: '',
  sms_code: ''
})

const checkTelephoneValid = () => {
  const regex = /^1([38][0-9]|14[579]|5[^4]|16[6]|7[1-35-8]|9[189])\d{8}$/
  return regex.test(registerData.telephone)
}

const sendSmsCode = async () => {
  if (!registerData.telephone || !registerData.nickname || !registerData.password) {
    ElMessage.error('请填写完整注册信息。')
    return
  }
  if (!checkTelephoneValid()) {
    ElMessage.error('请输入有效的手机号码。')
    return
  }
  const req = { telephone: registerData.telephone }
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

const handleRegister = async () => {
  try {
    if (!registerData.nickname || !registerData.telephone || !registerData.password || !registerData.sms_code) {
      ElMessage.error('请填写完整注册信息。')
      return
    }
    if (registerData.nickname.length < 3 || registerData.nickname.length > 10) {
      ElMessage.error('昵称长度在 3 到 10 个字符。')
      return
    }
    if (!checkTelephoneValid()) {
      ElMessage.error('请输入有效的手机号码。')
      return
    }
    const response = await axios.post(store.state.backendUrl + '/register', registerData)
    if (response.data.code == 200) {
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
    ElMessage.error('注册失败: ' + error.message)
  }
}

const handleLogin = () => {
  router.push('/login')
}

const handleSmsLogin = () => {
  router.push('/smslogin')
}
</script>

<style scoped>
.register-wrap {
  width: 100%;
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background-color: #f5f5f5;
}

.register-window {
  width: 400px;
  padding: 30px;
  background-color: #fff;
  border-radius: 8px;
}

.register-item {
  text-align: center;
  margin-bottom: 20px;
}

.register-button-container {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

.register-btn {
  width: 200px;
}

.go-login-button-container {
  display: flex;
  justify-content: space-between;
  margin-top: 20px;
}

.go-sms-login-btn,
.go-password-login-btn {
  background: none;
  border: none;
  color: #409eff;
  cursor: pointer;
  font-size: 14px;
}

.go-sms-login-btn:hover,
.go-password-login-btn:hover {
  color: #66b1ff;
}
</style>