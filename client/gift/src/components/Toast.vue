<script setup lang="ts">
import { toast } from '../stores/toast'
import Icon from './Icon.vue'
</script>

<template>
  <Transition name="toast">
    <div v-if="toast.message.value" class="toast">
      <span class="pip"><Icon name="check" :size="14" /></span>
      <span>{{ toast.message.value }}</span>
    </div>
  </Transition>
</template>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.25s, transform 0.25s;
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(10px);
}
</style>
