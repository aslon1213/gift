import { computed, ref } from 'vue'

const SERVER_KEY = 'gift.server_url'

function normalize(url: string): string {
  const trimmed = url.trim().replace(/\/+$/, '')
  if (!trimmed) return ''
  if (!/^https?:\/\//i.test(trimmed)) return `https://${trimmed}`
  return trimmed
}

const baseUrl = ref<string | null>(localStorage.getItem(SERVER_KEY))

function set(url: string) {
  const n = normalize(url)
  baseUrl.value = n
  localStorage.setItem(SERVER_KEY, n)
}

function clear() {
  baseUrl.value = null
  localStorage.removeItem(SERVER_KEY)
}

async function probe(url: string): Promise<{ ok: true } | { ok: false; error: string }> {
  const n = normalize(url)
  if (!n) return { ok: false, error: 'Enter a URL' }
  try {
    const res = await fetch(`${n}/health`, { method: 'GET' })
    if (!res.ok) return { ok: false, error: `Server responded with ${res.status}` }
    return { ok: true }
  } catch (e) {
    const msg = e instanceof Error ? e.message : 'Connection failed'
    return { ok: false, error: msg }
  }
}

export const server = {
  baseUrl,
  isConfigured: computed(() => !!baseUrl.value),
  set,
  clear,
  normalize,
  probe,
}
