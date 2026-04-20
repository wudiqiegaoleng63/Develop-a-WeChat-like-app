<template>
  <div class="manager-wrap">
    <div class="manager-window">
      <el-container>
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
                  <el-dropdown-item @click="goToManager">进入管理</el-dropdown-item>
                  <el-dropdown-item @click="logout">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
          <el-menu default-active="1" class="manager-menu">
            <el-menu-item index="1" @click="showUserManager">
              <el-icon><User /></el-icon>
              <span>用户管理</span>
            </el-menu-item>
            <el-menu-item index="2" @click="showGroupManager">
              <el-icon><Grid /></el-icon>
              <span>群组管理</span>
            </el-menu-item>
          </el-menu>
        </el-aside>
        <el-main class="main-container">
          <div v-if="currentTab === 'user'" class="user-manager">
            <h3>用户管理</h3>
            <el-table :data="userList" stripe>
              <el-table-column prop="uuid" label="用户ID" width="150" />
              <el-table-column prop="nickname" label="昵称" width="120" />
              <el-table-column prop="telephone" label="电话" width="120" />
              <el-table-column prop="status" label="状态" width="80">
                <template #default="{ row }">
                  <span :style="{ color: row.status === 0 ? 'green' : 'red' }">
                    {{ row.status === 0 ? '正常' : '禁用' }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column prop="is_admin" label="管理员" width="80">
                <template #default="{ row }">
                  <span :style="{ color: row.is_admin === 1 ? 'green' : 'gray' }">
                    {{ row.is_admin === 1 ? '是' : '否' }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="250">
                <template #default="{ row }">
                  <el-button size="small" @click="enableUser(row)">启用</el-button>
                  <el-button size="small" type="warning" @click="disableUser(row)">禁用</el-button>
                  <el-button size="small" type="danger" @click="deleteUser(row)">删除</el-button>
                  <el-button size="small" type="success" @click="setAdmin(row)">设管</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
          <div v-if="currentTab === 'group'" class="group-manager">
            <h3>群组管理</h3>
            <el-table :data="groupList" stripe>
              <el-table-column prop="uuid" label="群ID" width="150" />
              <el-table-column prop="name" label="群名" width="150" />
              <el-table-column prop="owner_uuid" label="群主" width="150" />
              <el-table-column prop="status" label="状态" width="80">
                <template #default="{ row }">
                  <span :style="{ color: row.status === 0 ? 'green' : 'red' }">
                    {{ row.status === 0 ? '正常' : '禁用' }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="200">
                <template #default="{ row }">
                  <el-button size="small" type="warning" @click="setGroupStatus(row)">禁用</el-button>
                  <el-button size="small" type="danger" @click="deleteGroup(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
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
import { ChatDotRound, User, HomeFilled, Setting, Grid } from '@element-plus/icons-vue'

const store = useStore()
const router = useRouter()

const userInfo = computed(() => store.state.userInfo)
const currentTab = ref('user')
const userList = ref([])
const groupList = ref([])

onMounted(() => {
  loadUserList()
  loadGroupList()
})

const loadUserList = async () => {
  try {
    const response = await axios.get(store.state.backendUrl + '/user/getUserInfoList')
    if (response.data.code === 200) {
      userList.value = response.data.data || []
    }
  } catch (error) {
    console.error('加载用户列表失败:', error)
  }
}

const loadGroupList = async () => {
  try {
    const response = await axios.get(store.state.backendUrl + '/group/getGroupInfoList')
    if (response.data.code === 200) {
      groupList.value = response.data.data || []
    }
  } catch (error) {
    console.error('加载群组列表失败:', error)
  }
}

const enableUser = async (row) => {
  try {
    const response = await axios.post(store.state.backendUrl + '/user/ableUsers', {
      uuids: [row.uuid]
    })
    if (response.data.code === 200) {
      ElMessage.success('已启用用户')
      loadUserList()
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('操作失败: ' + error.message)
  }
}

const disableUser = async (row) => {
  try {
    const response = await axios.post(store.state.backendUrl + '/user/disableUsers', {
      uuids: [row.uuid]
    })
    if (response.data.code === 200) {
      ElMessage.success('已禁用用户')
      loadUserList()
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('操作失败: ' + error.message)
  }
}

const deleteUser = async (row) => {
  try {
    const response = await axios.post(store.state.backendUrl + '/user/deleteUsers', {
      uuids: [row.uuid]
    })
    if (response.data.code === 200) {
      ElMessage.success('已删除用户')
      loadUserList()
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('操作失败: ' + error.message)
  }
}

const setAdmin = async (row) => {
  try {
    const response = await axios.post(store.state.backendUrl + '/user/setAdmin', {
      uuid: row.uuid,
      is_admin: row.is_admin === 1 ? 0 : 1
    })
    if (response.data.code === 200) {
      ElMessage.success('已设置管理员状态')
      loadUserList()
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('操作失败: ' + error.message)
  }
}

const setGroupStatus = async (row) => {
  try {
    const response = await axios.post(store.state.backendUrl + '/group/setGroupsStatus', {
      uuids: [row.uuid],
      status: row.status === 0 ? 1 : 0
    })
    if (response.data.code === 200) {
      ElMessage.success('已更新群组状态')
      loadGroupList()
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('操作失败: ' + error.message)
  }
}

const deleteGroup = async (row) => {
  try {
    const response = await axios.post(store.state.backendUrl + '/group/deleteGroups', {
      uuids: [row.uuid]
    })
    if (response.data.code === 200) {
      ElMessage.success('已删除群组')
      loadGroupList()
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('操作失败: ' + error.message)
  }
}

const showUserManager = () => {
  currentTab.value = 'user'
}

const showGroupManager = () => {
  currentTab.value = 'group'
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
.manager-wrap {
  width: 100%;
  height: 100vh;
  background-color: #f5f5f5;
}

.manager-window {
  width: 100%;
  height: 100%;
}

.aside-container {
  width: 200px;
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

.manager-menu {
  background-color: transparent;
  border: none;
}

.manager-menu .el-menu-item {
  color: #fff;
}

.manager-menu .el-menu-item:hover {
  background-color: #3e3e3e;
}

.main-container {
  padding: 20px;
}

.user-manager,
.group-manager {
  background-color: #fff;
  padding: 20px;
  border-radius: 8px;
}
</style>