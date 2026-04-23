<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { authApi } from '../api/endpoints'
import { auth } from '../stores/auth'
import { server } from '../stores/server'

const email = ref('')
const password = ref('')
const error = ref<string | null>(null)
const loading = ref(false)

const router = useRouter()
const route = useRoute()

async function submit() {
  error.value = null
  loading.value = true
  try {
    const data = await authApi.login(email.value, password.value)
    auth.setTokens(data.access_token, data.refresh_token)
    const to = (route.query.redirect as string) || '/dashboard'
    router.push(to)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-card">
    <h1>Welcome back</h1>
    <form @submit.prevent="submit">
      <label>
        <span>Email</span>
        <input v-model="email" type="email" required autocomplete="email" />
      </label>
      <label>
        <span>Password</span>
        <input v-model="password" type="password" required autocomplete="current-password" />
      </label>
      <p v-if="error" class="error">{{ error }}</p>
      <button type="submit" :disabled="loading">
        {{ loading ? 'Signing in…' : 'Sign in' }}
      </button>
    </form>
    <p class="muted">
      No account? <router-link to="/register">Create one</router-link>
    </p>
    <p class="muted small server-line">
      Server: <span class="server-url">{{ server.baseUrl.value }}</span>
      · <router-link to="/setup">Change</router-link>
    </p>
  </div>
</template>

<style scoped>
.server-line {
  margin-top: 8px;
  font-size: 12px;
  display: flex;
  gap: 4px;
  justify-content: center;
  align-items: center;
  flex-wrap: wrap;
}
.server-url {
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 220px;
  white-space: nowrap;
}
</style>
