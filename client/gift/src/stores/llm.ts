import { computed, ref } from 'vue'

export interface LlmConfig {
  base_url: string
  api_key: string
  chat_model: string
}

const STORAGE_KEY = 'gift.llm_config'

const DEFAULT_CONFIG: LlmConfig = {
  base_url: 'https://api.openai.com/v1',
  api_key: '',
  chat_model: 'gpt-4o-audio-preview',
}

function load(): LlmConfig {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...DEFAULT_CONFIG }
    const parsed = JSON.parse(raw) as Partial<LlmConfig>
    return { ...DEFAULT_CONFIG, ...parsed }
  } catch {
    return { ...DEFAULT_CONFIG }
  }
}

function normalizeBaseUrl(url: string): string {
  return url.trim().replace(/\/+$/, '')
}

const config = ref<LlmConfig>(load())

function set(next: Partial<LlmConfig>) {
  const merged: LlmConfig = {
    ...config.value,
    ...next,
    base_url: next.base_url !== undefined ? normalizeBaseUrl(next.base_url) : config.value.base_url,
  }
  config.value = merged
  localStorage.setItem(STORAGE_KEY, JSON.stringify(merged))
}

function clear() {
  config.value = { ...DEFAULT_CONFIG }
  localStorage.removeItem(STORAGE_KEY)
}

const isConfigured = computed(
  () => !!config.value.base_url && !!config.value.api_key && !!config.value.chat_model,
)

export const llm = {
  config,
  isConfigured,
  set,
  clear,
  DEFAULT_CONFIG,
}
