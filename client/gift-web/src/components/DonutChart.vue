<script setup lang="ts">
import { computed } from 'vue'
import { colorFor } from '../utils/charts'

interface Slice {
  label: string
  value: number
}

const props = defineProps<{
  data: Slice[]
  size?: number
  currency?: string
  title?: string
}>()

const size = computed(() => props.size ?? 220)
const radius = computed(() => size.value / 2 - 8)
const inner = computed(() => radius.value * 0.6)

const total = computed(() => props.data.reduce((a, s) => a + s.value, 0))

interface Arc {
  d: string
  color: string
  label: string
  value: number
  pct: number
}

const arcs = computed<Arc[]>(() => {
  const cx = size.value / 2
  const cy = size.value / 2
  const r = radius.value
  const ri = inner.value
  const t = total.value
  if (t <= 0) return []

  const data = [...props.data].sort((a, b) => b.value - a.value)

  // Single slice: draw full donut as a complete circle (two half-arcs)
  if (data.length === 1) {
    const s = data[0]
    const d = [
      `M ${cx - r} ${cy}`,
      `A ${r} ${r} 0 1 1 ${cx + r} ${cy}`,
      `A ${r} ${r} 0 1 1 ${cx - r} ${cy}`,
      `L ${cx - ri} ${cy}`,
      `A ${ri} ${ri} 0 1 0 ${cx + ri} ${cy}`,
      `A ${ri} ${ri} 0 1 0 ${cx - ri} ${cy}`,
      'Z',
    ].join(' ')
    return [{ d, color: colorFor(0), label: s.label, value: s.value, pct: 1 }]
  }

  let start = -Math.PI / 2
  return data.map((s, i) => {
    const pct = s.value / t
    const end = start + pct * Math.PI * 2
    const x1 = cx + r * Math.cos(start)
    const y1 = cy + r * Math.sin(start)
    const x2 = cx + r * Math.cos(end)
    const y2 = cy + r * Math.sin(end)
    const xi1 = cx + ri * Math.cos(end)
    const yi1 = cy + ri * Math.sin(end)
    const xi2 = cx + ri * Math.cos(start)
    const yi2 = cy + ri * Math.sin(start)
    const large = pct > 0.5 ? 1 : 0
    const d = [
      `M ${x1} ${y1}`,
      `A ${r} ${r} 0 ${large} 1 ${x2} ${y2}`,
      `L ${xi1} ${yi1}`,
      `A ${ri} ${ri} 0 ${large} 0 ${xi2} ${yi2}`,
      'Z',
    ].join(' ')
    const arc: Arc = { d, color: colorFor(i), label: s.label, value: s.value, pct }
    start = end
    return arc
  })
})

function fmt(n: number) {
  return n.toLocaleString(undefined, { maximumFractionDigits: 2 })
}
</script>

<template>
  <div class="donut">
    <h3 v-if="title">{{ title }}</h3>
    <div v-if="total === 0" class="empty muted">No data</div>
    <div v-else class="donut-body">
      <svg :width="size" :height="size" :viewBox="`0 0 ${size} ${size}`" role="img" aria-label="Donut chart">
        <path
          v-for="(a, i) in arcs"
          :key="i"
          :d="a.d"
          :fill="a.color"
        >
          <title>{{ a.label }}: {{ fmt(a.value) }} ({{ (a.pct * 100).toFixed(1) }}%)</title>
        </path>
        <text :x="size / 2" :y="size / 2 - 4" text-anchor="middle" class="donut-total">
          {{ fmt(total) }}
        </text>
        <text :x="size / 2" :y="size / 2 + 14" text-anchor="middle" class="donut-sub">
          {{ currency ?? 'total' }}
        </text>
      </svg>
      <ul class="legend">
        <li v-for="(a, i) in arcs" :key="i">
          <span class="swatch" :style="{ background: a.color }"></span>
          <span class="legend-label">{{ a.label }}</span>
          <span class="muted small">{{ (a.pct * 100).toFixed(0) }}%</span>
          <span>{{ fmt(a.value) }}</span>
        </li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.donut {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 16px;
}
.donut h3 {
  margin: 0 0 12px;
  font-size: 14px;
  color: var(--muted);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.donut-body {
  display: flex;
  gap: 20px;
  align-items: center;
  flex-wrap: wrap;
}
.empty {
  padding: 40px 0;
  text-align: center;
}
.donut-total {
  font-size: 18px;
  font-weight: 600;
  fill: var(--text);
}
.donut-sub {
  font-size: 11px;
  fill: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.legend {
  list-style: none;
  padding: 0;
  margin: 0;
  flex: 1;
  min-width: 180px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.legend li {
  display: grid;
  grid-template-columns: 12px 1fr auto auto;
  gap: 8px;
  align-items: center;
  font-size: 14px;
}
.swatch {
  width: 12px;
  height: 12px;
  border-radius: 3px;
}
.legend-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
