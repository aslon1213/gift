<script setup lang="ts">
import type { IconName } from './icons'

defineProps<{
  name: IconName
  size?: number
}>()
</script>

<template>
  <svg
    :width="size ?? 20"
    :height="size ?? 20"
    viewBox="0 0 20 20"
    fill="none"
    stroke="currentColor"
    stroke-width="1.4"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path v-if="name === 'plus'" d="M10 4v12M4 10h12" />
    <path v-else-if="name === 'minus'" d="M4 10h12" />
    <path v-else-if="name === 'arrowRight'" d="M4 10h12M11 5l5 5-5 5" />
    <path v-else-if="name === 'arrowLeft'" d="M16 10H4M9 5l-5 5 5 5" />
    <path v-else-if="name === 'arrowUp'" d="M10 16V4M5 9l5-5 5 5" />
    <path v-else-if="name === 'arrowDown'" d="M10 4v12M5 11l5 5 5-5" />
    <path v-else-if="name === 'close'" d="M5 5l10 10M15 5L5 15" />
    <path v-else-if="name === 'check'" d="M4 10l4 4 8-9" stroke-width="1.6" />
    <path v-else-if="name === 'chevR'" d="M8 5l5 5-5 5" />
    <path v-else-if="name === 'chevD'" d="M5 8l5 5 5-5" />
    <g v-else-if="name === 'dots'" fill="currentColor" stroke="none">
      <circle cx="4" cy="10" r="1.5" />
      <circle cx="10" cy="10" r="1.5" />
      <circle cx="16" cy="10" r="1.5" />
    </g>
    <g v-else-if="name === 'search'">
      <circle cx="9" cy="9" r="5" />
      <path d="M13 13l4 4" />
    </g>
    <path
      v-else-if="name === 'bell'"
      d="M10 3a5 5 0 00-5 5v3l-1.5 2.5h13L15 11V8a5 5 0 00-5-5zM8 17a2 2 0 004 0"
    />
    <g v-else-if="name === 'people'">
      <circle cx="7" cy="8" r="3" />
      <path d="M1.5 17c.5-2.5 2.8-4.5 5.5-4.5s5 2 5.5 4.5" />
      <circle cx="14" cy="7" r="2.5" />
      <path d="M13 12c2 0 4.5 1.5 5.5 4" />
    </g>
    <path v-else-if="name === 'home'" d="M3 9l7-5 7 5v8a1 1 0 01-1 1h-3v-6H7v6H4a1 1 0 01-1-1V9z" />
    <g v-else-if="name === 'target'">
      <circle cx="10" cy="10" r="7" />
      <circle cx="10" cy="10" r="3.5" />
      <circle cx="10" cy="10" r="1" fill="currentColor" />
    </g>
    <g v-else-if="name === 'gauge'">
      <path d="M3.5 14a7 7 0 0113 0" />
      <path d="M10 14l3.5-5" />
    </g>
    <g v-else-if="name === 'settings'">
      <circle cx="10" cy="10" r="2.5" />
      <path d="M10 2v2M10 16v2M2 10h2M16 10h2M4.5 4.5l1.5 1.5M14 14l1.5 1.5M4.5 15.5L6 14M14 6l1.5-1.5" />
    </g>
    <g v-else-if="name === 'server'">
      <rect x="3" y="3" width="14" height="6" rx="1.5" />
      <rect x="3" y="11" width="14" height="6" rx="1.5" />
      <path d="M6 6h.01M6 14h.01" />
    </g>
    <path
      v-else-if="name === 'plane'"
      d="M10 3l1 6 6 3-6 1-1 4-1-4-6-1 6-3 1-6z"
    />
    <path
      v-else-if="name === 'fork'"
      d="M6 3v6a2 2 0 104 0V3M8 11v6M13 3c-1 2-1 4 0 6v8"
    />
    <g v-else-if="name === 'bed'">
      <path d="M3 14V6M3 10h14v4M17 14v2M3 16v-2" />
      <circle cx="7" cy="10" r="1.5" />
    </g>
    <g v-else-if="name === 'cart'">
      <path d="M3 4h2l2 9h9l2-6H6" />
      <circle cx="8" cy="16" r="1" />
      <circle cx="15" cy="16" r="1" />
    </g>
    <g v-else-if="name === 'car'">
      <path d="M3 12l1.5-5h11L17 12M3 12v3h2v-1h10v1h2v-3M3 12h14" />
      <circle cx="6" cy="13.5" r="0.5" fill="currentColor" />
      <circle cx="14" cy="13.5" r="0.5" fill="currentColor" />
    </g>
    <g v-else-if="name === 'ticket'">
      <path d="M3 7a2 2 0 002-2h10a2 2 0 002 2v2a2 2 0 000 4v2a2 2 0 00-2 2H5a2 2 0 00-2-2v-2a2 2 0 000-4V7z" />
      <path d="M9 5v10" stroke-dasharray="1 2" />
    </g>
    <path
      v-else-if="name === 'coffee'"
      d="M3 8h11v4a4 4 0 01-4 4H7a4 4 0 01-4-4V8zM14 9h2a2 2 0 010 4h-2M7 5s1-1 0-2M10 5s1-1 0-2"
    />
    <path v-else-if="name === 'home2'" d="M3 9l7-5 7 5v8h-5v-5H8v5H3V9z" />
    <g v-else-if="name === 'gift'">
      <rect x="3" y="8" width="14" height="9" rx="1" />
      <path d="M3 11h14M10 8v9M10 8c-2 0-3-1-3-2.5S8 3 10 6c2-3 3-2.5 3-.5S12 8 10 8z" />
    </g>
    <g v-else-if="name === 'wallet'">
      <rect x="3" y="6" width="14" height="10" rx="2" />
      <path d="M3 9h14M14 12h1" />
    </g>
  </svg>
</template>
