<template>
  <el-dialog
    v-model="isVisible"
    :title="title"
    width="300px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <span>{{ content }}</span>
    <template #footer>
      <el-button @click="handleCancel">取消</el-button>
      <el-button type="primary" @click="handleConfirm">确认</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  isVisible: {
    type: Boolean,
    default: false
  },
  title: {
    type: String,
    default: '提示'
  },
  content: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['close', 'confirm', 'cancel', 'update:isVisible'])

const isVisible = computed({
  get() {
    return props.isVisible
  },
  set(value) {
    emit('update:isVisible', value)
  }
})

const handleClose = () => {
  emit('close')
}

const handleCancel = () => {
  emit('cancel')
  emit('update:isVisible', false)
}

const handleConfirm = () => {
  emit('confirm')
  emit('update:isVisible', false)
}
</script>

<style scoped>
</style>