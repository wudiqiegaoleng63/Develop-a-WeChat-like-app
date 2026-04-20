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
          <div class="contact-list-container">
            <div class="contact-list-header">
              <el-input v-model="searchKey" placeholder="搜索" size="small" style="width: 150px" />
              <el-dropdown placement="bottom" trigger="click">
                <el-icon style="color: #fff; cursor: pointer"><Plus /></el-icon>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item @click="showCreateGroupModal">创建群聊</el-dropdown-item>
                    <el-dropdown-item @click="showAddContactModal">添加用户</el-dropdown-item>
                    <el-dropdown-item @click="showAddGroupModal">添加群聊</el-dropdown-item>
                    <el-dropdown-item @click="showNewContactModal">新的好友</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
            <el-menu default-active="1" class="contact-menu">
              <el-menu-item index="1" @click="showUserContacts">
                <el-icon><User /></el-icon>
                <span>联系人</span>
              </el-menu-item>
              <el-menu-item index="2" @click="showCreatedGroups">
                <el-icon><Grid /></el-icon>
                <span>我创建的群</span>
              </el-menu-item>
              <el-menu-item index="3" @click="showJoinedGroups">
                <el-icon><Grid /></el-icon>
                <span>我加入的群</span>
              </el-menu-item>
            </el-menu>
            <el-scrollbar height="300px">
              <div v-for="item in currentList" :key="item.uuid || item.group_uuid" class="contact-item" @click="openChat(item)">
                <img :src="item.avatar" class="contact-avatar" />
                <span class="contact-name">{{ item.nickname || item.name }}</span>
              </div>
            </el-scrollbar>
          </div>
        </el-aside>
        <el-main class="main-container">
          <div class="welcome-content">
            <h2>通讯录</h2>
          </div>
        </el-main>
      </el-container>
    </div>

    <CreateGroupModal v-model:isVisible="createGroupVisible" @success="loadContacts" />
    <AddContactModal v-model:isVisible="addContactVisible" @success="loadContacts" />
    <AddGroupModal v-model:isVisible="addGroupVisible" @success="loadContacts" />
    <NewContactModal v-model:isVisible="newContactVisible" @success="loadContacts" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useStore } from 'vuex'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { ChatDotRound, User, HomeFilled, Setting, Plus, Grid } from '@element-plus/icons-vue'
import CreateGroupModal from '../../components/CreateGroupModal.vue'
import AddContactModal from '../../components/AddContactModal.vue'
import AddGroupModal from '../../components/AddGroupModal.vue'
import NewContactModal from '../../components/NewContactModal.vue'

const store = useStore()
const router = useRouter()

const searchKey = ref('')
const userContacts = ref([])
const createdGroups = ref([])
const joinedGroups = ref([])
const currentList = ref([])
const currentType = ref('user')

const userInfo = computed(() => store.state.userInfo)
const isAdmin = computed(() => userInfo.value.uuid && userInfo.value.uuid.substring(0, 4) === '1000')

const createGroupVisible = ref(false)
const addContactVisible = ref(false)
const addGroupVisible = ref(false)
const newContactVisible = ref(false)

onMounted(() => {
  loadContacts()
})

const loadContacts = async () => {
  try {
    const response = await axios.get(store.state.backendUrl + '/contact/getUserList', {
      params: { user_uuid: userInfo.value.uuid }
    })
    if (response.data.code === 200) {
      userContacts.value = response.data.data || []
      userContacts.value.forEach(item => {
        if (item.avatar && !item.avatar.startsWith('http')) {
          item.avatar = store.state.backendUrl + item.avatar
        }
      })
    }
  } catch (error) {
    console.error('加载联系人失败:', error)
  }

  try {
    const response = await axios.get(store.state.backendUrl + '/group/loadMyGroup', {
      params: { owner_uuid: userInfo.value.uuid }
    })
    if (response.data.code === 200) {
      createdGroups.value = response.data.data || []
      createdGroups.value.forEach(item => {
        if (item.avatar && !item.avatar.startsWith('http')) {
          item.avatar = store.state.backendUrl + item.avatar
        }
      })
    }
  } catch (error) {
    console.error('加载创建的群失败:', error)
  }

  try {
    const response = await axios.get(store.state.backendUrl + '/contact/loadMyJoinedGroup', {
      params: { user_uuid: userInfo.value.uuid }
    })
    if (response.data.code === 200) {
      joinedGroups.value = response.data.data || []
      joinedGroups.value.forEach(item => {
        if (item.avatar && !item.avatar.startsWith('http')) {
          item.avatar = store.state.backendUrl + item.avatar
        }
      })
    }
  } catch (error) {
    console.error('加载加入的群失败:', error)
  }

  showUserContacts()
}

const showUserContacts = () => {
  currentType.value = 'user'
  currentList.value = userContacts.value
}

const showCreatedGroups = () => {
  currentType.value = 'created'
  currentList.value = createdGroups.value
}

const showJoinedGroups = () => {
  currentType.value = 'joined'
  currentList.value = joinedGroups.value
}

const openChat = (item) => {
  let sessionId
  if (currentType.value === 'user') {
    sessionId = userInfo.value.uuid + '_' + item.uuid
  } else {
    sessionId = userInfo.value.uuid + '_' + item.uuid
  }
  router.push('/chat/' + sessionId)
}

const showCreateGroupModal = () => {
  createGroupVisible.value = true
}

const showAddContactModal = () => {
  addContactVisible.value = true
}

const showAddGroupModal = () => {
  addGroupVisible.value = true
}

const showNewContactModal = () => {
  newContactVisible.value = true
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

.contact-list-container {
  padding: 10px;
}

.contact-list-header {
  display: flex;
  justify-content: space-between;
  padding: 10px;
}

.contact-menu {
  background-color: transparent;
  border: none;
}

.contact-menu .el-menu-item {
  color: #fff;
}

.contact-menu .el-menu-item:hover {
  background-color: #3e3e3e;
}

.contact-item {
  display: flex;
  align-items: center;
  padding: 10px;
  cursor: pointer;
  border-radius: 4px;
  margin-bottom: 5px;
}

.contact-item:hover {
  background-color: #3e3e3e;
}

.contact-avatar {
  width: 40px;
  height: 40px;
  border-radius: 4px;
  margin-right: 10px;
}

.contact-name {
  font-size: 14px;
  color: #fff;
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