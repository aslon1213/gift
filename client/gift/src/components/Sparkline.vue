<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    data: number[]
    width?: number
    height?: number
    stroke?: string
  }>(),
  { width: 320, height: 48, stroke: '#F5F1E8' },
)

const parts = computed(() => {
  const data = props.data.length ? props.data : [0, 0]
  const max = Math.max(...data, 1)
  const min = 0
  const pts = data.map((v, i) => [
    (i / Math.max(data.length - 1, 1)) * props.width,
    props.height - 4 - ((v - min) / (max - min || 1)) * (props.height - 8),
  ])
  const d = pts
    .map((p, i) => (i ? 'L' : 'M') + p[0].toFixed(1) + ' ' + p[1].toFixed(1))
    .join(' ')
  const area = d + ` L${props.width} ${props.height} L0 ${props.height} Z`
  return { d, area, pts }
})

const gradId = `spark-fill-${Math.random().toString(36).slice(2, 7)}`
</script>

<template>
  <svg
    :width="width"
    :height="height"
    :viewBox="`0 0 ${width} ${height}`"
    style="display: block"
  >
    <defs>
      <linearGradient :id="gradId" x1="0" x2="0" y1="0" y2="1">
        <stop offset="0%" :stop-color="stroke" stop-opacity="0.25" />
        <stop offset="100%" :stop-color="stroke" stop-opacity="0" />
      </linearGradient>
    </defs>
    <path :d="parts.area" :fill="`url(#${gradId})`" />
    <path
      :d="parts.d"
      fill="none"
      :stroke="stroke"
      stroke-width="1.2"
      stroke-linejoin="round"
      stroke-linecap="round"
    />
    <circle
      v-if="parts.pts.length"
      :cx="parts.pts[parts.pts.length - 1][0]"
      :cy="parts.pts[parts.pts.length - 1][1]"
      r="3"
      :fill="stroke"
    />
  </svg>
</template>
