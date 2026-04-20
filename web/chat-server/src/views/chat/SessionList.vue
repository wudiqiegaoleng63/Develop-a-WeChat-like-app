<template>
  <div class="chat-wrap">
    <div class="chat-window">
      <el-container class="chat-window-container">
        <el-aside class="aside-container">
          <div class="nav-container">
            <div class="nav-item" @click="goToSession">
              <el-icon><ChatDotRound /></el-icon>
            </div>
            <div class="nav-item" @click="goToContact">
              <el-icon><User /></el-icon>
            </div>
            <div class="nav-item" @click="goToOwnInfo">
              <el-icon><HomeFilled /></el-icon>
            </div>
            <el-dropdown placement="top" trigger="click">
              <div class="nav-item">
                <el-icon><Setting /></el-icon>
              </div>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item v-if="isAdmin" @click="goToManager">进入管理</el-dropdown-item>
                  <el-dropdown-item @click="logout">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
          <div class="session-list-container">
            <el-scrollbar height="calc(100vh - 100px)">
              <div v-for="session in sessionList" :key="session.session_id" class="session-item" @click="openChat(session)">
                <img :src="session.avatar" class="session-avatar" />
                <div class="session-info">
                  <span class="session-name">{{ session.name }}</span>
                  <span class="session-content">{{ session.last_content }}</span>
                </div>
              </div>
            </el-scrollbar>
          </div>
        </el-aside>
        <el-main class="main-container">
          <div class="welcome-content">
            <h2>欢迎使用 Kama Chat Server</h2>
            <p>请选择左侧会话开始聊天</p>
          </div>
        </el-main>
      </el-container>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useStore } from 'vuex'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { ChatDotRound, User, HomeFilled, Setting } from '@element-plus/icons-vue'

const store = useStore()
const router = useRouter()

const sessionList = ref([])
const userInfo = computed(() => store.state.userInfo)
const isAdmin = computed(() => userInfo.value.uuid && userInfo.value.uuid.substring(0, 4) === '1000')

onMounted(() => {
  loadSessionList()
})

const loadSessionList = async () => {
  try {
    const response = await axios.get(store.state.backendUrl + '/session/getUserSessionList', {
      params: { user_uuid: userInfo.value.uuid }
    })
    if (response.data.code === 200) {
      sessionList.value = response.data.data || []
      sessionList.value.forEach(session => {
        if (session.avatar && !session.avatar.startsWith('http')) {
          session.avatar = store.state.backendUrl + session.avatar
        }
      })
    }
  } catch (error) {
    console.error('加载会话列表失败:', error)
  }
}

const openChat = (session) => {
  router.push('/chat/' + session.session_id)
}

const goToSession = () => {
  router.push('/sessionlist')
}

const goToContact = () => {
  router.push('/contactlist')
}

const goToOwnInfo = () => {
  router.push('/owninfo')
}

const goToManager = () => {
  router.push('/manager')
}

const logout = () => {
  store.commit('cleanUserInfo')
  router.push('/login')
}
</script>

<style scoped>
.chat-wrap {
  width: 100%;
  height: 100vh;
  background-color: #f5f5f5;
}

.chat-window {
  width: 100%;
  height: 100%;
}

.chat-window-container {
  height: 100%;
}

.aside-container {
  width: 300px;
  background-color: #2e2e2e;
}

.nav-container {
  display: flex;
  justify-content: space-around;
  padding: 15px 10px;
  background-color: #1e1e1e;
}

.nav-item {
  width: 40px;
  height: 40px;
  display: flex;
  justify-content: center;
  align-items: center;
  cursor: pointer;
  color: #fff;
  border-radius: 4px;
}

.nav-item:hover {
  background-color: #3e3e3e;
}

.session-list-container {
  padding: 10px;
}

.session-item {
  display: flex;
  align-items: center;
  padding: 10px;
  cursor: pointer;
  border-radius: 4px;
  margin-bottom: 5px;
}

.session-item:hover {
  background-color: #3e3e3e;
}

.session-avatar {
  width: 45px;
  height: 45px;
  border-radius: 4px;
  margin-right: 10px;
}

.session-info {
  flex: 1;
  overflow: hidden;
}

.session-name {
  font-size: 14px;
  color: #fff;
  display: block;
}

.session-content {
  font-size: 12px;
  color: #aaa;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.main-container {
  display: flex;
  justify-content: center;
  align-items: center;
}

.welcome-content {
  text-align: center;
  color: #666;
}
</style>