import { computed, ref } from 'vue'

const ACCESS_KEY = 'gift.access_token'
const REFRESH_KEY = 'gift.refresh_token'
const USER_KEY = 'gift.user'

export interface SessionUser {
  id: string
  email: string
  username: string
}

const accessToken = ref<string | null>(localStorage.getItem(ACCESS_KEY))
const refreshToken = ref<string | null>(localStorage.getItem(REFRESH_KEY))
const user = ref<SessionUser | null>(readUser())

function readUser(): SessionUser | null {
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as SessionUser
  } catch {
    return null
  }
}

function setTokens(access: string, refresh: string) {
  accessToken.value = access
  refreshToken.value = refresh
  localStorage.setItem(ACCESS_KEY, access)
  localStorage.setItem(REFRESH_KEY, refresh)
  const parsed = parseJwt(access)
  if (parsed?.sub && !user.value) {
    user.value = { id: parsed.sub, email: '', username: '' }
    localStorage.setItem(USER_KEY, JSON.stringify(user.value))
  }
}

function setUser(u: SessionUser) {
  user.value = u
  localStorage.setItem(USER_KEY, JSON.stringify(u))
}

function clear() {
  accessToken.value = null
  refreshToken.value = null
  user.value = null
  localStorage.removeItem(ACCESS_KEY)
  localStorage.removeItem(REFRESH_KEY)
  localStorage.removeItem(USER_KEY)
}

function parseJwt(token: string): { sub?: string; exp?: number } | null {
  try {
    const [, payload] = token.split('.')
    const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/'))
    return JSON.parse(json)
  } catch {
    return null
  }
}

function userIdFromToken(): string | null {
  if (user.value?.id) return user.value.id
  if (!accessToken.value) return null
  return parseJwt(accessToken.value)?.sub ?? null
}

export const auth = {
  accessToken,
  refreshToken,
  user,
  isAuthenticated: computed(() => !!accessToken.value),
  setTokens,
  setUser,
  clear,
  userIdFromToken,
}
