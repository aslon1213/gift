<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/endpoints'
import { auth } from '../stores/auth'
import { userStore } from '../stores/user'
import type { CurrencyCode } from '../api/types'
import { CURRENCY_CODES, CURRENCY_META } from '../utils/format'
import Icon from '../components/Icon.vue'

const email = ref('')
const username = ref('')
const password = ref('')
const currency = ref<CurrencyCode>('UZS')
const error = ref<string | null>(null)
const loading = ref(false)
const router = useRouter()

async function submit() {
  error.value = null
  loading.value = true
  try {
    await authApi.register(email.value, username.value, password.value, currency.value)
    const login = await authApi.login(email.value, password.value)
    auth.setTokens(login.access_token, login.refresh_token)
    auth.setUser({
      id: auth.userIdFromToken() ?? '',
      email: email.value,
      username: username.value,
    })
    await userStore.load()
    router.push('/home')
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
      <label class="field">
        <span>Currency</span>
        <div class="pill-row">
          <button
            v-for="code in CURRENCY_CODES"
            :key="code"
            type="button"
            class="pill"
            :class="{ on: currency === code }"
            @click="currency = code"
          >
            {{ CURRENCY_META[code].symbol }} {{ code }}
          </button>
        </div>
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
