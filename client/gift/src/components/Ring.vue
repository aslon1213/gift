<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    pct: number
    size?: number
    stroke?: number
    color?: string
    bg?: string
  }>(),
  { size: 52, stroke: 4, color: '#14171F', bg: 'rgba(20,23,31,0.08)' },
)

const geom = computed(() => {
  const r = (props.size - props.stroke) / 2
  const c = 2 * Math.PI * r
  const clamped = Math.min(Math.max(props.pct, 0), 1)
  return { r, c, dashOffset: c * (1 - clamped) }
})
</script>

<template>
  <svg :width="size" :height="size" :viewBox="`0 0 ${size} ${size}`">
    <circle
      :cx="size / 2"
      :cy="size / 2"
      :r="geom.r"
      fill="none"
      :stroke="bg"
      :stroke-width="stroke"
    />
    <circle
      :cx="size / 2"
      :cy="size / 2"
      :r="geom.r"
      fill="none"
      :stroke="color"
      :stroke-width="stroke"
      :stroke-dasharray="geom.c"
      :stroke-dashoffset="geom.dashOffset"
      stroke-linecap="round"
      :transform="`rotate(-90 ${size / 2} ${size / 2})`"
    />
  </svg>
</template>
