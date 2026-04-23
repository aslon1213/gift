<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { server } from '../stores/server'
import Icon from '../components/Icon.vue'

const router = useRouter()
const route = useRoute()

const url = ref(server.baseUrl.value ?? '')
const testing = ref(false)
const status = ref<'idle' | 'ok' | 'fail'>('idle')
const message = ref<string>('')

async function test() {
  status.value = 'idle'
  message.value = ''
  testing.value = true
  try {
    const res = await server.probe(url.value)
    if (res.ok) {
      status.value = 'ok'
      message.value = 'Connected successfully.'
    } else {
      status.value = 'fail'
      message.value = res.error
    }
  } finally {
    testing.value = false
  }
}

async function save() {
  await test()
  if (status.value !== 'ok') return
  server.set(url.value)
  const to = (route.query.redirect as string) || '/login'
  router.push(to)
}

function useLocal() {
  url.value = 'http://localhost:3000'
}

function useSameOrigin() {
  url.value = window.location.origin
}
</script>

<template>
  <div class="auth-card setup-card">
    <div class="eyebrow">
      <span class="dot"></span>
      <span>self-host · step 1</span>
    </div>
    <h1>Connect your <em>server.</em></h1>
    <p class="muted small intro">
      Enter the URL of your Gift server. You can change it later from the sign-in screen.
    </p>
    <form @submit.prevent="save">
      <label class="field">
        <span>Server URL</span>
        <input
          v-model="url"
          type="url"
          inputmode="url"
          autocomplete="off"
          placeholder="https://gift.example.com"
          required
        />
      </label>

      <div class="shortcut-row">
        <button type="button" class="linklike" @click="useLocal">
          localhost:3000
        </button>
        <button type="button" class="linklike" @click="useSameOrigin">
          this origin
        </button>
      </div>

      <p v-if="status === 'fail'" class="error">✗ {{ message }}</p>
      <p v-else-if="status === 'ok'" class="success">✓ {{ message }}</p>

      <div class="actions-row">
        <button type="button" class="btn btn-secondary" :disabled="testing" @click="test">
          {{ testing ? 'Testing…' : 'Test' }}
        </button>
        <button type="submit" class="btn btn-primary btn-lg" :disabled="testing">
          Connect <Icon name="arrowRight" :size="16" />
        </button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.setup-card {
  max-width: 440px;
}
.intro {
  margin: -4px 0 18px;
  font-family: var(--sans);
  letter-spacing: normal;
  text-align: left;
  color: var(--ink-soft);
  font-size: 14px;
}
.shortcut-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: -4px;
}
.actions-row {
  display: grid;
  grid-template-columns: 1fr 2fr;
  gap: 10px;
  margin-top: 8px;
}
</style>
