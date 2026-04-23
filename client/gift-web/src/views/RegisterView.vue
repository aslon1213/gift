<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/endpoints'
import { auth } from '../stores/auth'

const email = ref('')
const username = ref('')
const password = ref('')
const error = ref<string | null>(null)
const loading = ref(false)
const router = useRouter()

async function submit() {
  error.value = null
  loading.value = true
  try {
    await authApi.register(email.value, username.value, password.value)
    const login = await authApi.login(email.value, password.value)
    auth.setTokens(login.access_token, login.refresh_token)
    auth.setUser({
      id: auth.userIdFromToken() ?? '',
      email: email.value,
      username: username.value,
    })
    router.push('/dashboard')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Registration failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-card">
    <h1>Create account</h1>
    <form @submit.prevent="submit">
      <label>
        <span>Email</span>
        <input v-model="email" type="email" required autocomplete="email" />
      </label>
      <label>
        <span>Name</span>
        <input v-model="username" required autocomplete="username" />
      </label>
      <label>
        <span>Password</span>
        <input v-model="password" type="password" required autocomplete="new-password" />
      </label>
      <p v-if="error" class="error">{{ error }}</p>
      <button type="submit" :disabled="loading">
        {{ loading ? 'Creating…' : 'Create account' }}
      </button>
    </form>
    <p class="muted">
      Already registered? <router-link to="/login">Sign in</router-link>
    </p>
    <p class="muted small server-line">
      Server: <router-link to="/setup">Change</router-link>
    </p>
  </div>
</template>

<style scoped>
.server-line {
  margin-top: 8px;
  font-size: 12px;
  text-align: center;
}
</style>
