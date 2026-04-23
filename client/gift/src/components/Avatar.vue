<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    name: string
    color?: string
    size?: number
    ring?: boolean
  }>(),
  { size: 32, ring: false, color: '#4A5577' },
)

function initial(n: string): string {
  if (!n) return '?'
  return n.trim()[0].toUpperCase()
}

import { computed } from 'vue'

const style = computed(() => ({
  width: `${props.size}px`,
  height: `${props.size}px`,
  background: props.color,
  fontSize: `${props.size * 0.48}px`,
  color: props.color,
  boxShadow: props.ring
    ? `0 0 0 2px var(--paper), 0 0 0 3px ${props.color}`
    : undefined,
}))
</script>

<template>
  <div class="avatar" :style="style">
    <span style="color: var(--paper)">{{ initial(name) }}</span>
  </div>
</template>
