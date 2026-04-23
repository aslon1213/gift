<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/endpoints'
import { auth } from '../stores/auth'
import Icon from '../components/Icon.vue'

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
    <div class="eyebrow">
      <span class="dot"></span>
      <span>invite your people</span>
    </div>
    <h1>New <em>crew.</em></h1>
    <form @submit.prevent="submit">
      <label class="field">
        <span>Email</span>
        <input v-model="email" type="email" required autocomplete="email" />
      </label>
      <label class="field">
        <span>Name</span>
        <input v-model="username" required autocomplete="username" />
      </label>
      <label class="field">
        <span>Password</span>
        <input v-model="password" type="password" required autocomplete="new-password" />
      </label>
      <p v-if="error" class="error">{{ error }}</p>
      <button class="btn btn-accent btn-lg btn-block" type="submit" :disabled="loading">
        {{ loading ? 'Creating…' : 'Create account' }}
        <Icon v-if="!loading" name="check" :size="16" />
      </button>
    </form>
    <p class="muted">
      Already registered? <router-link to="/login">Sign in</router-link>
    </p>
  </div>
</template>
