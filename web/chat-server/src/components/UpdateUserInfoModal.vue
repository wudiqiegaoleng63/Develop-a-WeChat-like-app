<template>
  <Modal :isVisible="isVisible" title="编辑个人信息" width="400px" @close="handleClose">
    <template #body>
      <el-form :model="userInfoData" label-width="80px">
        <el-form-item label="昵称">
          <el-input v-model="userInfoData.nickname" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="userInfoData.email" />
        </el-form-item>
        <el-form-item label="性别">
          <el-radio-group v-model="userInfoData.gender">
            <el-radio :value="0">男</el-radio>
            <el-radio :value="1">女</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="生日">
          <el-date-picker v-model="userInfoData.birthday" type="date" placeholder="选择日期" />
        </el-form-item>
        <el-form-item label="签名">
          <el-input v-model="userInfoData.signature" type="textarea" />
        </el-form-item>
        <el-form-item label="头像">
          <el-upload
            class="avatar-uploader"
            :action="uploadUrl"
            :show-file-list="false"
            :on-success="handleAvatarSuccess"
          >
            <img v-if="userInfoData.avatar" :src="userInfoData.avatar" class="avatar" />
            <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
          </el-upload>
        </el-form-item>
      </el-form>
    </template>
    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" @click="handleUpdate">保存</el-button>
    </template>
  </Modal>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
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
const userInfoData = ref({
  nickname: '',
  email: '',
  gender: 0,
  birthday: '',
  signature: '',
  avatar: ''
})

const uploadUrl = computed(() => store.state.backendUrl + '/message/uploadAvatar')

watch(() => props.isVisible, (val) => {
  isVisible.value = val
  if (val) {
    userInfoData.value = {
      nickname: store.state.userInfo.nickname || '',
      email: store.state.userInfo.email || '',
      gender: store.state.userInfo.gender || 0,
      birthday: store.state.userInfo.birthday || '',
      signature: store.state.userInfo.signature || '',
      avatar: store.state.userInfo.avatar || ''
    }
  }
})

const handleClose = () => {
  emit('close')
  emit('update:isVisible', false)
}

const handleAvatarSuccess = (response) => {
  if (response.code === 200) {
    userInfoData.value.avatar = store.state.backendUrl + response.data.url
  }
}

const handleUpdate = async () => {
  try {
    const response = await axios.post(store.state.backendUrl + '/user/updateUserInfo', {
      uuid: store.state.userInfo.uuid,
      ...userInfoData.value
    })
    if (response.data.code === 200) {
      ElMessage.success('更新成功')
      store.commit('setUserInfo', response.data.data)
      emit('success')
      handleClose()
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('更新失败: ' + error.message)
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