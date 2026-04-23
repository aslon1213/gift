<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { authApi } from '../api/endpoints'
import { auth } from '../stores/auth'
import { server } from '../stores/server'
import Icon from '../components/Icon.vue'

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
    <div class="eyebrow">
      <span class="dot"></span>
      <span>self-hosted · v0.4.2</span>
    </div>
    <h1>Welcome <em>back.</em></h1>
    <form @submit.prevent="submit">
      <label class="field">
        <span>Email</span>
        <input v-model="email" type="email" required autocomplete="email" />
      </label>
      <label class="field">
        <span>Password</span>
        <input v-model="password" type="password" required autocomplete="current-password" />
      </label>
      <p v-if="error" class="error">{{ error }}</p>
      <button class="btn btn-primary btn-lg btn-block" type="submit" :disabled="loading">
        {{ loading ? 'Signing in…' : 'Sign in' }}
        <Icon v-if="!loading" name="arrowRight" :size="16" />
      </button>
    </form>
    <p class="muted">
      No account? <router-link to="/register">Create one</router-link>
    </p>
    <p class="muted small server-line">
      Server: <span class="server-url">{{ server.baseUrl.value }}</span> ·
      <router-link to="/setup">change</router-link>
    </p>
  </div>
</template>

<style scoped>
.server-line {
  margin-top: 8px;
  font-size: 11px;
  display: flex;
  gap: 6px;
  justify-content: center;
  align-items: center;
  flex-wrap: wrap;
  font-family: var(--mono);
  letter-spacing: 0.06em;
}
.server-url {
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 220px;
  white-space: nowrap;
  color: var(--ink);
}
</style>
