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
        </el-aside>
        <el-main class="main-container">
          <div class="owner-info-window">
            <div class="my-homepage-title"><h2>我的主页</h2></div>
            <p class="owner-prefix">用户id：{{ userInfo.uuid }}</p>
            <p class="owner-prefix">昵称：{{ userInfo.nickname }}</p>
            <p class="owner-prefix">电话：{{ userInfo.telephone }}</p>
            <p class="owner-prefix">邮箱：{{ userInfo.email }}</p>
            <p class="owner-prefix">性别：{{ userInfo.gender === 0 ? '男' : '女' }}</p>
            <p class="owner-prefix">生日：{{ userInfo.birthday }}</p>
            <p class="owner-prefix">个性签名：{{ userInfo.signature }}</p>
            <p class="owner-prefix">加入时间：{{ userInfo.created_at }}</p>
            <div class="owner-opt">
              <p class="owner-prefix">头像：</p>
              <img style="width: 80px; height: 80px; border-radius: 4px" :src="userInfo.avatar" />
            </div>
            <div class="edit-window">
              <el-button class="edit-btn" @click="showUpdateModal">编辑</el-button>
            </div>
          </div>
        </el-main>
      </el-container>
    </div>

    <UpdateUserInfoModal v-model:isVisible="updateVisible" @success="loadUserInfo" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useStore } from 'vuex'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ChatDotRound, User, HomeFilled, Setting } from '@element-plus/icons-vue'
import UpdateUserInfoModal from '../../components/UpdateUserInfoModal.vue'

const store = useStore()
const router = useRouter()

const userInfo = computed(() => store.state.userInfo)
const isAdmin = computed(() => userInfo.value.uuid && userInfo.value.uuid.substring(0, 4) === '1000')
const updateVisible = ref(false)

onMounted(() => {
  loadUserInfo()
})

const loadUserInfo = async () => {
  try {
    const response = await axios.get(store.state.backendUrl + '/user/getUserInfo', {
      params: { uuid: userInfo.value.uuid }
    })
    if (response.data.code === 200) {
      if (response.data.data.avatar && !response.data.data.avatar.startsWith('http')) {
        response.data.data.avatar = store.state.backendUrl + response.data.data.avatar
      }
      store.commit('setUserInfo', response.data.data)
    }
  } catch (error) {
    console.error('加载用户信息失败:', error)
  }
}

const showUpdateModal = () => {
  updateVisible.value = true
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
  width: 80px;
  background-color: #2e2e2e;
}

.nav-container {
  display: flex;
  flex-direction: column;
  align-items: center;
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
  margin-bottom: 10px;
}

.nav-item:hover {
  background-color: #3e3e3e;
}

.main-container {
  display: flex;
  justify-content: center;
}

.owner-info-window {
  padding: 30px;
  background-color: #fff;
  border-radius: 8px;
  width: 500px;
}

.my-homepage-title {
  text-align: center;
  margin-bottom: 20px;
}

.owner-prefix {
  margin-bottom: 10px;
  color: #333;
}

.owner-opt {
  display: flex;
  align-items: center;
  margin-top: 15px;
}

.edit-window {
  margin-top: 20px;
  text-align: center;
}

.edit-btn {
  width: 200px;
}
</style>