<template>

  <div class="modal-overlay" v-if="isVisible">
    <div class="modal-content">
      <slot name="header">
      </slot>
      <slot name="body">
      </slot>
      <slot name="footer">
      </slot>
    </div>
  </div>
</template>

<script>
export default {
  name: "Modal",
  props: {
    isVisible: false,
  },
};
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 2000;
  border-radius: 30px;
}

.modal-content {
  background: #fff;
  height: 400px;
  border-radius: 20px;
  width: 500px;
  box-shadow: 0 2px 15px rgb(195, 8, 8);
  display: flex;
  flex-direction: column;
  align-items: center;
  
}

</style>
