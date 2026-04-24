<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/endpoints'
import { auth } from '../stores/auth'
import { userStore } from '../stores/user'
import type { CurrencyCode } from '../api/types'
import { CURRENCY_CODES, CURRENCY_META } from '../utils/format'
import { t } from '../i18n'
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
      <span>{{ t('register.invite_your_people') }}</span>
    </div>
    <h1>{{ t('register.new_crew') }}</h1>
    <form @submit.prevent="submit">
      <label class="field">
        <span>{{ t('common.email') }}</span>
        <input v-model="email" type="email" required autocomplete="email" />
      </label>
      <label class="field">
        <span>{{ t('common.name') }}</span>
        <input v-model="username" required autocomplete="username" />
      </label>
      <label class="field">
        <span>{{ t('common.password') }}</span>
        <input v-model="password" type="password" required autocomplete="new-password" />
      </label>
      <label class="field">
        <span>{{ t('common.currency') }}</span>
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
        {{ loading ? t('register.creating') : t('register.create_account') }}
        <Icon v-if="!loading" name="check" :size="16" />
      </button>
    </form>
    <p class="muted">
      {{ t('register.already_registered') }} <router-link to="/login">{{ t('login.sign_in') }}</router-link>
    </p>
  </div>
</template>
