<template>
  <Modal :isVisible="isVisible" title="添加群聊" width="400px" @close="handleClose">
    <template #body>
      <el-form :model="searchData" label-width="80px">
        <el-form-item label="群ID">
          <el-input v-model="searchData.group_id" />
        </el-form-item>
      </el-form>
      <div v-if="searchResult" class="search-result">
        <div class="result-item">
          <img :src="searchResult.avatar" class="result-avatar" />
          <div class="result-info">
            <span class="result-name">{{ searchResult.name }}</span>
            <span class="result-desc">{{ searchResult.info }}</span>
          </div>
          <el-button type="primary" size="small" @click="joinGroup">加入</el-button>
        </div>
      </div>
    </template>
    <template #footer>
      <el-button type="primary" @click="searchGroup">搜索</el-button>
      <el-button @click="handleClose">取消</el-button>
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
  }
})

const emit = defineEmits(['close', 'success', 'update:isVisible'])

const store = useStore()

const isVisible = ref(props.isVisible)
const searchData = ref({ group_id: '' })
const searchResult = ref(null)

watch(() => props.isVisible, (val) => {
  isVisible.value = val
  if (!val) {
    searchData.value = { group_id: '' }
    searchResult.value = null
  }
})

const handleClose = () => {
  emit('close')
  emit('update:isVisible', false)
}

const searchGroup = async () => {
  if (!searchData.value.group_id) {
    ElMessage.error('请输入群ID')
    return
  }
  try {
    const response = await axios.get(store.state.backendUrl + '/group/getGroupInfo', {
      params: { uuid: searchData.value.group_id }
    })
    if (response.data.code === 200) {
      searchResult.value = response.data.data
      if (searchResult.value.avatar && !searchResult.value.avatar.startsWith('http')) {
        searchResult.value.avatar = store.state.backendUrl + searchResult.value.avatar
      }
    } else {
      ElMessage.error(response.data.message)
    }
  } catch (error) {
    ElMessage.error('搜索失败: ' + error.message)
  }
}

const joinGroup = async () => {
  try {
    // 先检查入群方式
    const checkResponse = await axios.get(store.state.backendUrl + '/group/checkGroupAddMode', {
      params: { group_uuid: searchResult.value.uuid }
    })
    if (checkResponse.data.code === 200) {
      const addMode = checkResponse.data.data.add_mode
      if (addMode === 0) {
        // 自由加入
        const response = await axios.post(store.state.backendUrl + '/group/enterGroupDirectly', {
          group_uuid: searchResult.value.uuid,
          user_uuid: store.state.userInfo.uuid
        })
        if (response.data.code === 200) {
          ElMessage.success('加入成功')
          emit('success')
          handleClose()
        } else {
          ElMessage.error(response.data.message)
        }
      } else {
        ElMessage.warning('该群需要申请才能加入')
        handleClose()
      }
    }
  } catch (error) {
    ElMessage.error('加入失败: ' + error.message)
  }
}
</script>

<style scoped>
.search-result {
  margin-top: 20px;
}

.result-item {
  display: flex;
  align-items: center;
  padding: 10px;
  border: 1px solid #eee;
  border-radius: 4px;
}

.result-avatar {
  width: 50px;
  height: 50px;
  border-radius: 4px;
  margin-right: 10px;
}

.result-info {
  flex: 1;
}

.result-name {
  font-size: 16px;
  font-weight: bold;
}

.result-desc {
  font-size: 12px;
  color: #666;
  margin-top: 5px;
  display: block;
}
</style>