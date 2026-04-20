<template>
  <Modal :isVisible="isVisible" title="创建群聊" width="400px" @close="handleClose">
    <template #body>
      <el-form :model="groupData" label-width="80px">
        <el-form-item label="群名称">
          <el-input v-model="groupData.name" />
        </el-form-item>
        <el-form-item label="群介绍">
          <el-input v-model="groupData.info" type="textarea" />
        </el-form-item>
        <el-form-item label="群头像">
          <el-upload
            class="avatar-uploader"
            :action="uploadUrl"
            :show-file-list="false"
            :on-success="handleAvatarSuccess"
          >
            <img v-if="groupData.avatar" :src="groupData.avatar" class="avatar" />
            <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
          </el-upload>
        </el-form-item>
        <el-form-item label="入群方式">
          <el-radio-group v-model="groupData.add_mode">
            <el-radio :value="0">自由加入</el-radio>
            <el-radio :value="1">需要申请</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
    </template>
    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" @click="handleCreate">创建</el-button>
    </template>
  </Modal>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useStore } from 'vuex'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
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
const groupData = ref({
  name: '',
  info: '',
  avatar: '',
  add_mode: 0
})

const uploadUrl = computed(() => store.state.backendUrl + '/message/uploadAvatar')

watch(() => props.isVisible, (val) => {
  isVisible.value = val
})

const handleClose = () => {
  emit('close')
  emit('update:isVisible', false)
  groupData.value = { name: '', info: '', avatar: '', add_mode: 0 }
}

const handleAvatarSuccess = (response) => {
  if (response.code === 200) {
    groupData.value.avatar = store.state.backendUrl + response.data.url
  }
}

const handleCreate = async () => {
  if (!groupData.value.name) {
    ElMessage.error('请输入群名称')
    return
  }
  try {
    const response = await axios.post(store.state.backendUrl + '/group/createGroup', {
      name: groupData.value.name,
      info: groupData.value.info,
      avatar: groupData.value.avatar,
      add_mode: groupData.value.add_mode,
      owner_uuid: store.state.userInfo.uuid
    })
    if (response.data.code === 200) {
      ElMessage.success('创建成功')
      emit('success')
      handleClose()
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('创建失败: ' + error.message)
  }
}
</script>

<style scoped>
.avatar-uploader {
  border: 1px dashed #d9d9d9;
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
}

.avatar-uploader:hover {
  border-color: #409eff;
}

.avatar-uploader-icon {
  font-size: 28px;
  color: #8c939d;
  width: 80px;
  height: 80px;
  text-align: center;
  line-height: 80px;
}

.avatar {
  width: 80px;
  height: 80px;
  display: block;
}
</style>