<template>
  <Modal :isVisible="isVisible" :title="title" width="400px" @close="handleClose">
    <template #header>
      <span>{{ title }}</span>
    </template>
    <template #body>
      <el-scrollbar height="300px">
        <div v-for="item in contactList" :key="item.contact_id" class="contact-item" @click="selectContact(item)">
          <img v-if="item.avatar" :src="item.avatar" class="contact-avatar" />
          <span class="contact-name">{{ item.nickname || item.group_name }}</span>
        </div>
      </el-scrollbar>
    </template>
    <template #footer>
      <el-button @click="handleClose">关闭</el-button>
    </template>
  </Modal>
</template>

<script setup>
import { ref, watch } from 'vue'
import Modal from './Modal.vue'

const props = defineProps({
  isVisible: {
    type: Boolean,
    default: false
  },
  title: {
    type: String,
    default: '选择联系人'
  },
  contacts: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['close', 'select', 'update:isVisible'])

const isVisible = ref(props.isVisible)
const contactList = ref([])

watch(() => props.isVisible, (val) => {
  isVisible.value = val
})

watch(() => props.contacts, (val) => {
  contactList.value = val
})

const handleClose = () => {
  emit('close')
  emit('update:isVisible', false)
}

const selectContact = (item) => {
  emit('select', item)
  handleClose()
}
</script>

<style scoped>
.contact-item {
  display: flex;
  align-items: center;
  padding: 10px;
  cursor: pointer;
  border-bottom: 1px solid #eee;
}

.contact-item:hover {
  background-color: #f5f5f5;
}

.contact-avatar {
  width: 40px;
  height: 40px;
  border-radius: 4px;
  margin-right: 10px;
}

.contact-name {
  font-size: 14px;
}
</style>