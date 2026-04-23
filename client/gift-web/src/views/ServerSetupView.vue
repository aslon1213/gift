<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { server } from '../stores/server'

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
    <h1>Connect to server</h1>
    <p class="muted">
      Enter the URL of your Gift server. You can change it later from the sign-in screen.
    </p>
    <form @submit.prevent="save">
      <label>
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
        <button type="button" class="linklike small" @click="useLocal">
          localhost:3000
        </button>
        <button type="button" class="linklike small" @click="useSameOrigin">
          This origin
        </button>
      </div>

      <p v-if="status === 'fail'" class="error">✗ {{ message }}</p>
      <p v-else-if="status === 'ok'" class="success">✓ {{ message }}</p>

      <div class="actions-row">
        <button type="button" class="ghost" :disabled="testing" @click="test">
          {{ testing ? 'Testing…' : 'Test' }}
        </button>
        <button type="submit" :disabled="testing">
          Connect
        </button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.setup-card {
  max-width: 440px;
}
.setup-card p.muted {
  font-size: 13px;
  margin: -4px 0 16px;
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
.success {
  color: var(--success);
  font-size: 13px;
}
.error {
  font-size: 13px;
}
</style>
