import { computed, ref } from 'vue'
import { authApi } from '../api/endpoints'
import type { CurrencyCode, User } from '../api/types'
import { auth } from './auth'

const CACHE_KEY = 'gift.user_profile'

function readCache(): User | null {
  const raw = localStorage.getItem(CACHE_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as User
  } catch {
    return null
  }
}

const profile = ref<User | null>(readCache())
const loading = ref(false)
const error = ref<string | null>(null)

function persist(u: User | null) {
  if (u) localStorage.setItem(CACHE_KEY, JSON.stringify(u))
  else localStorage.removeItem(CACHE_KEY)
}

async function load(): Promise<User | null> {
  if (!auth.isAuthenticated.value) {
    profile.value = null
    persist(null)
    return null
  }
  loading.value = true
  error.value = null
  try {
    const u = await authApi.me()
    profile.value = u
    persist(u)
    return u
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load profile'
    return profile.value
  } finally {
    loading.value = false
  }
}

function clear() {
  profile.value = null
  persist(null)
}

function setCurrency(code: CurrencyCode) {
  if (profile.value) {
    profile.value = { ...profile.value, currency: code }
    persist(profile.value)
  }
}

const currency = computed<CurrencyCode>(() => profile.value?.currency ?? 'UZS')

export const userStore = {
  profile,
  currency,
  loading,
  error,
  load,
  clear,
  setCurrency,
}
