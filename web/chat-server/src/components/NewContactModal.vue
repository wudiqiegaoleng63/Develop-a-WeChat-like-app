<template>
  <Modal :isVisible="isVisible" title="新的好友申请" width="400px" @close="handleClose">
    <template #body>
      <el-scrollbar height="300px">
        <div v-for="item in applyList" :key="item.uuid" class="apply-item">
          <img :src="item.avatar" class="apply-avatar" />
          <div class="apply-info">
            <span class="apply-name">{{ item.nickname }}</span>
            <span class="apply-desc">申请添加好友</span>
          </div>
          <div class="apply-actions">
            <el-button type="primary" size="small" @click="passApply(item)">通过</el-button>
            <el-button type="danger" size="small" @click="rejectApply(item)">拒绝</el-button>
          </div>
        </div>
      </el-scrollbar>
    </template>
    <template #footer>
      <el-button @click="handleClose">关闭</el-button>
    </template>
  </Modal>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { useStore } from 'vuex'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import Modal from './Modal.vue'

const props = defineProps({
  isVisible: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['close', 'success', 'update:isVisible'])

const store = useStore()

const isVisible = ref(props.isVisible)
const applyList = ref([])

watch(() => props.isVisible, (val) => {
  isVisible.value = val
  if (val) {
    loadApplyList()
  }
})

const handleClose = () => {
  emit('close')
  emit('update:isVisible', false)
}

const loadApplyList = async () => {
  try {
    const response = await axios.get(store.state.backendUrl + '/contact/getNewContactList', {
      params: { user_uuid: store.state.userInfo.uuid }
    })
    if (response.data.code === 200) {
      applyList.value = response.data.data || []
      applyList.value.forEach(item => {
        if (item.avatar && !item.avatar.startsWith('http')) {
          item.avatar = store.state.backendUrl + item.avatar
        }
      })
    }
  } catch (error) {
    console.error('加载申请列表失败:', error)
  }
}

const passApply = async (item) => {
  try {
    const response = await axios.post(store.state.backendUrl + '/contact/passContactApply', {
      user_uuid: store.state.userInfo.uuid,
      contact_uuid: item.uuid
    })
    if (response.data.code === 200) {
      ElMessage.success('已通过申请')
      loadApplyList()
      emit('success')
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('操作失败: ' + error.message)
  }
}

const rejectApply = async (item) => {
  try {
    const response = await axios.post(store.state.backendUrl + '/contact/refuseContactApply', {
      user_uuid: store.state.userInfo.uuid,
      contact_uuid: item.uuid
    })
    if (response.data.code === 200) {
      ElMessage.success('已拒绝申请')
      loadApplyList()
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('操作失败: ' + error.message)
  }
}
</script>

<style scoped>
.apply-item {
  display: flex;
  align-items: center;
  padding: 10px;
  border-bottom: 1px solid #eee;
}

.apply-avatar {
  width: 40px;
  height: 40px;
  border-radius: 4px;
  margin-right: 10px;
}

.apply-info {
  flex: 1;
}

.apply-name {
  font-size: 14px;
  font-weight: bold;
}

.apply-desc {
  font-size: 12px;
  color: #666;
  display: block;
}

.apply-actions {
  display: flex;
  gap: 5px;
}
</style>