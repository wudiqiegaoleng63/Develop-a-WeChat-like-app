<template>
  <Modal :isVisible="isVisible" :title="title" width="500px" @close="handleClose">
    <template #body>
      <div class="group-info-content">
        <div class="info-row">
          <span class="info-label">群ID：</span>
          <span class="info-value">{{ groupInfo.uuid }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">群名称：</span>
          <span class="info-value">{{ groupInfo.name }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">群介绍：</span>
          <span class="info-value">{{ groupInfo.info }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">创建时间：</span>
          <span class="info-value">{{ groupInfo.created_at }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">群头像：</span>
          <img :src="groupInfo.avatar" class="group-avatar" />
        </div>
        <div class="member-section">
          <h4>群成员</h4>
          <el-scrollbar height="200px">
            <div v-for="member in memberList" :key="member.uuid" class="member-item">
              <img :src="member.avatar" class="member-avatar" />
              <span class="member-name">{{ member.nickname }}</span>
              <el-button v-if="isOwner && member.uuid !== userInfo.uuid" type="danger" size="small" @click="removeMember(member)">移除</el-button>
            </div>
          </el-scrollbar>
        </div>
      </div>
    </template>
    <template #footer>
      <el-button v-if="isOwner" type="primary" @click="showUpdateModal">编辑</el-button>
      <el-button v-if="isOwner" type="danger" @click="dismissGroup">解散群</el-button>
      <el-button v-if="!isOwner" type="warning" @click="leaveGroup">退出群</el-button>
      <el-button @click="handleClose">关闭</el-button>
    </template>
  </Modal>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useStore } from 'vuex'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import Modal from './Modal.vue'

const props = defineProps({
  isVisible: {
    type: Boolean,
    default: false
  },
  groupInfo: {
    type: Object,
    default: () => {}
  },
  memberList: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['close', 'update', 'leave', 'dismiss', 'removeMember'])

const store = useStore()

const isVisible = ref(props.isVisible)
const userInfo = computed(() => store.state.userInfo)
const isOwner = computed(() => props.groupInfo.owner_uuid === userInfo.value.uuid)
const title = computed(() => props.groupInfo.name || '群信息')

watch(() => props.isVisible, (val) => {
  isVisible.value = val
})

const handleClose = () => {
  emit('close')
}

const showUpdateModal = () => {
  emit('update')
}

const leaveGroup = async () => {
  try {
    const response = await axios.post(store.state.backendUrl + '/group/leaveGroup', {
      user_uuid: userInfo.value.uuid,
      group_uuid: props.groupInfo.uuid
    })
    if (response.data.code === 200) {
      ElMessage.success('已退出群聊')
      emit('leave')
      handleClose()
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('退出失败: ' + error.message)
  }
}

const dismissGroup = async () => {
  try {
    const response = await axios.post(store.state.backendUrl + '/group/dismissGroup', {
      group_uuid: props.groupInfo.uuid
    })
    if (response.data.code === 200) {
      ElMessage.success('已解散群聊')
      emit('dismiss')
      handleClose()
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('解散失败: ' + error.message)
  }
}

const removeMember = async (member) => {
  try {
    const response = await axios.post(store.state.backendUrl + '/group/removeGroupMembers', {
      group_uuid: props.groupInfo.uuid,
      member_uuids: [member.uuid]
    })
    if (response.data.code === 200) {
      ElMessage.success('已移除成员')
      emit('removeMember', member)
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('移除失败: ' + error.message)
  }
}
</script>

<style scoped>
.group-info-content {
  padding: 10px;
}

.info-row {
  display: flex;
  align-items: center;
  margin-bottom: 15px;
}

.info-label {
  width: 80px;
  color: #666;
}

.info-value {
  flex: 1;
}

.group-avatar {
  width: 60px;
  height: 60px;
  border-radius: 4px;
}

.member-section {
  margin-top: 20px;
}

.member-item {
  display: flex;
  align-items: center;
  padding: 8px;
  border-bottom: 1px solid #eee;
}

.member-avatar {
  width: 40px;
  height: 40px;
  border-radius: 4px;
  margin-right: 10px;
}

.member-name {
  flex: 1;
}
</style>