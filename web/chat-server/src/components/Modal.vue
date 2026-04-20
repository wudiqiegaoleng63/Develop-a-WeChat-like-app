<template>
  <el-dialog
    v-model="isVisible"
    :title="title"
    :width="width"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <template #header>
      <slot name="header"></slot>
    </template>
    <slot name="body"></slot>
    <template #footer>
      <slot name="footer"></slot>
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
    default: ''
  },
  width: {
    type: String,
    default: '50%'
  }
})

const emit = defineEmits(['close', 'update:isVisible'])

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
</script>

<style scoped>
</style>