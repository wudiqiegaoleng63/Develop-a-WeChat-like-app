<template>
  <Modal :isVisible="isVisible" title="联系人详情" width="400px" @close="handleClose">
    <template #body>
      <div class="contact-info-content">
        <div class="info-row">
          <span class="info-label">用户ID：</span>
          <span class="info-value">{{ contactInfo.uuid }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">昵称：</span>
          <span class="info-value">{{ contactInfo.nickname }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">电话：</span>
          <span class="info-value">{{ contactInfo.telephone }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">签名：</span>
          <span class="info-value">{{ contactInfo.signature }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">头像：</span>
          <img :src="contactInfo.avatar" class="contact-avatar" />
        </div>
      </div>
    </template>
    <template #footer>
      <el-button type="danger" @click="deleteContact">删除联系人</el-button>
      <el-button type="warning" @click="blackContact">拉黑</el-button>
      <el-button @click="handleClose">关闭</el-button>
    </template>
  </Modal>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useStore } from 'vuex'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import Modal from './Modal.vue'

const props = defineProps({
  isVisible: {
    type: Boolean,
    default: false
  },
  contactInfo: {
    type: Object,
    default: () => {}
  }
})

const emit = defineEmits(['close', 'delete', 'black'])

const store = useStore()

const isVisible = ref(props.isVisible)

watch(() => props.isVisible, (val) => {
  isVisible.value = val
})

const handleClose = () => {
  emit('close')
}

const deleteContact = async () => {
  try {
    const response = await axios.post(store.state.backendUrl + '/contact/deleteContact', {
      user_uuid: store.state.userInfo.uuid,
      contact_uuid: props.contactInfo.uuid
    })
    if (response.data.code === 200) {
      ElMessage.success('已删除联系人')
      emit('delete')
      handleClose()
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('删除失败: ' + error.message)
  }
}

const blackContact = async () => {
  try {
    const response = await axios.post(store.state.backendUrl + '/contact/blackContact', {
      user_uuid: store.state.userInfo.uuid,
      contact_uuid: props.contactInfo.uuid
    })
    if (response.data.code === 200) {
      ElMessage.success('已拉黑联系人')
      emit('black')
      handleClose()
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('拉黑失败: ' + error.message)
  }
}
</script>

<style scoped>
.contact-info-content {
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

.contact-avatar {
  width: 60px;
  height: 60px;
  border-radius: 4px;
}
</style>