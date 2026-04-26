<script setup lang="ts" generic="T">
import { computed } from 'vue'
import Icon from './Icon.vue'
import { useVoiceInput } from '../composables/useVoiceInput'
import { llm } from '../stores/llm'
import { t } from '../i18n'

const props = defineProps<{
  parser: (audio: Blob) => Promise<T>
  label?: string
}>()

const emit = defineEmits<{
  result: [value: T]
  error: [message: string]
}>()

const voice = useVoiceInput()

const busy = computed(() => voice.status.value === 'thinking')
const recording = computed(() => voice.status.value === 'recording')
const configured = computed(() => llm.isConfigured.value)

const hint = computed(() => {
  switch (voice.status.value) {
    case 'recording':
      return t('voice.recording_tap_to_stop')
    case 'thinking':
      return t('voice.parsing')
    case 'error':
      return voice.error.value || t('voice.error')
    default:
      return props.label ?? t('voice.tap_to_speak')
  }
})

async function toggle() {
  if (!configured.value) {
    emit('error', t('voice.not_configured'))
    return
  }
  if (voice.status.value === 'idle' || voice.status.value === 'error') {
    await voice.start()
    return
  }
  if (recording.value) {
    const result = await voice.run(props.parser)
    if (result != null) {
      emit('result', result)
      voice.reset()
    } else if (voice.error.value) {
      emit('error', voice.error.value)
    }
  }
}
</script>

<template>
  <button
    type="button"
    class="voice-btn"
    :class="{ rec: recording, busy }"
    :disabled="busy"
    @click="toggle"
  >
    <span class="voice-dot" v-if="recording"></span>
    <Icon v-else :name="busy ? 'sparkle' : 'mic'" :size="14" />
    <span class="voice-label">{{ hint }}</span>
  </button>
</template>

<style scoped>
.voice-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--paper);
  color: var(--ink);
  border: 1px solid var(--line);
  border-radius: 999px;
  font-family: var(--mono);
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}
.voice-btn:hover:not(:disabled) {
  border-color: var(--ink);
}
.voice-btn:disabled {
  cursor: wait;
  opacity: 0.75;
}
.voice-btn.rec {
  background: var(--hot, #d64933);
  color: #fff;
  border-color: var(--hot, #d64933);
  animation: voicePulse 1.2s ease-in-out infinite;
}
.voice-btn.busy {
  background: var(--ink);
  color: var(--paper);
  border-color: var(--ink);
}
.voice-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.35);
}
.voice-label {
  line-height: 1;
}
@keyframes voicePulse {
  0%,
  100% {
    box-shadow: 0 0 0 0 rgba(214, 73, 51, 0.6);
  }
  50% {
    box-shadow: 0 0 0 8px rgba(214, 73, 51, 0);
  }
}
</style>
