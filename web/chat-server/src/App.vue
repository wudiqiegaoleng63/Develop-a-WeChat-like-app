<template>
  <router-view />
</template>

<script setup>
import { onMounted, onBeforeUnmount } from 'vue'
import { useStore } from 'vuex'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElNotification } from 'element-plus'

const store = useStore()
const router = useRouter()

onMounted(() => {
  // 从sessionStorage获取用户信息
  const userInfoStr = sessionStorage.getItem('userInfo')
  if (userInfoStr) {
    const userInfo = JSON.parse(userInfoStr)
    store.commit('setUserInfo', userInfo)

    // 初始化WebSocket连接
    initWebSocket()
  }
})

onBeforeUnmount(() => {
  if (store.state.socket) {
    store.state.socket.close()
  }
})

const initWebSocket = () => {
  const wsUrl = store.state.wsUrl + '/wss?client_id=' + store.state.userInfo.uuid

  try {
    const socket = new WebSocket(wsUrl)
    store.state.socket = socket

    socket.onopen = () => {
      console.log('WebSocket连接已建立')
    }

    socket.onmessage = (event) => {
      const message = JSON.parse(event.data)
      console.log('收到WebSocket消息:', message)

      // 处理不同类型的消息
      if (message.type === 3) {
        // 音视频消息
        const avData = JSON.parse(message.av_data)
        if (avData.messageId === 'PROXY' && avData.type === 'start_call') {
          ElNotification({
            title: '通话请求',
            message: `收到来自${message.send_name}的通话请求`,
            type: 'warning'
          })
        }
      }

      // 触发自定义事件供其他组件监听
      window.dispatchEvent(new CustomEvent('ws-message', { detail: message }))
    }

    socket.onclose = () => {
      console.log('WebSocket连接已关闭')
      store.state.socket = null
    }

    socket.onerror = (error) => {
      console.error('WebSocket连接错误:', error)
      ElMessage.error('WebSocket连接失败')
    }
  } catch (error) {
    console.error('初始化WebSocket失败:', error)
  }
}

// 获取用户信息
const getUserInfo = () => {
  if (!store.state.userInfo.uuid) return

  axios.get(store.state.backendUrl + '/user/getUserInfo', {
    params: { uuid: store.state.userInfo.uuid }
  }).then(res => {
    if (res.data.code === 200) {
      store.commit('setUserInfo', res.data.data)
    }
  }).catch(err => {
    console.error('获取用户信息失败:', err)
  })
}
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
</style>