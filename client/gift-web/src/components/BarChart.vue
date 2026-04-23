<script setup lang="ts">
import { computed } from 'vue'

interface Bar {
  label: string
  value: number
  // Optional split: when set, bar renders value (positive) and secondary (negative) stacked
  secondary?: number
}

const props = defineProps<{
  data: Bar[]
  height?: number
  title?: string
  primaryColor?: string
  secondaryColor?: string
  showEveryNthLabel?: number
}>()

const height = computed(() => props.height ?? 180)
const primary = computed(() => props.primaryColor ?? '#a78bfa')
const secondary = computed(() => props.secondaryColor ?? '#f87171')

const max = computed(() => {
  let m = 0
  for (const b of props.data) {
    const v = Math.max(b.value, b.secondary ?? 0)
    if (v > m) m = v
  }
  return m || 1
})

const step = computed(() => props.showEveryNthLabel ?? Math.max(1, Math.floor(props.data.length / 6)))

const barW = 24
const gap = 6
const padL = 32
const padR = 8
const padT = 12
const padB = 24

const innerH = computed(() => height.value - padT - padB)
const totalW = computed(() => padL + padR + props.data.length * (barW + gap))

function y(v: number) {
  return padT + innerH.value - (v / max.value) * innerH.value
}

function fmt(n: number) {
  if (n >= 1000) return `${(n / 1000).toFixed(n >= 10000 ? 0 : 1)}k`
  return n.toLocaleString(undefined, { maximumFractionDigits: 0 })
}
</script>

<template>
  <div class="bars">
    <h3 v-if="title">{{ title }}</h3>
    <div class="scroll">
      <svg
        :width="totalW"
        :height="height"
        :viewBox="`0 0 ${totalW} ${height}`"
        role="img"
        aria-label="Bar chart"
      >
        <!-- y axis baseline -->
        <line
          :x1="padL"
          :x2="totalW - padR"
          :y1="padT + innerH"
          :y2="padT + innerH"
          stroke="var(--border)"
          stroke-width="1"
        />
        <!-- y axis label -->
        <text
          :x="padL - 6"
          :y="padT + 10"
          text-anchor="end"
          class="axis"
        >{{ fmt(max) }}</text>
        <text
          :x="padL - 6"
          :y="padT + innerH"
          text-anchor="end"
          class="axis"
        >0</text>

        <g
          v-for="(b, i) in data"
          :key="i"
          :transform="`translate(${padL + i * (barW + gap)}, 0)`"
        >
          <rect
            v-if="b.secondary"
            :x="0"
            :y="y(b.secondary)"
            :width="barW / 2 - 1"
            :height="padT + innerH - y(b.secondary)"
            :fill="secondary"
            rx="2"
          >
            <title>{{ b.label }}: {{ fmt(b.secondary) }}</title>
          </rect>
          <rect
            :x="b.secondary ? barW / 2 + 1 : 0"
            :y="y(b.value)"
            :width="b.secondary ? barW / 2 - 1 : barW"
            :height="padT + innerH - y(b.value)"
            :fill="primary"
            rx="2"
          >
            <title>{{ b.label }}: {{ fmt(b.value) }}</title>
          </rect>
          <text
            v-if="i % step === 0 || i === data.length - 1"
            :x="barW / 2"
            :y="padT + innerH + 14"
            text-anchor="middle"
            class="axis"
          >{{ b.label }}</text>
        </g>
      </svg>
    </div>
  </div>
</template>

<style scoped>
.bars {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 16px;
}
.bars h3 {
  margin: 0 0 12px;
  font-size: 14px;
  color: var(--muted);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.scroll {
  overflow-x: auto;
}
.axis {
  font-size: 10px;
  fill: var(--muted);
}
</style>
