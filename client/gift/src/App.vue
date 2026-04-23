<script setup lang="ts">
import { useRouter } from 'vue-router'
import { auth } from './stores/auth'
import { authApi } from './api/endpoints'
import Icon from './components/Icon.vue'
import Toast from './components/Toast.vue'

const router = useRouter()

async function logout() {
  try {
    await authApi.logout()
  } catch {
    // ignore — clear client state regardless
  }
  auth.clear()
  router.push('/login')
}
</script>

<template>
  <div class="shell">
    <header class="topbar" v-if="auth.isAuthenticated.value">
      <div class="brand">
        <span class="dot"></span>
        <span>self-hosted · <b>gift</b></span>
      </div>
      <nav class="desktop-nav">
        <router-link to="/home">Home</router-link>
        <router-link to="/groups">Groups</router-link>
        <router-link to="/budgets">Budget</router-link>
        <router-link to="/goals">Goals</router-link>
        <router-link to="/settings">Server</router-link>
      </nav>
      <button class="linklike" @click="logout">Logout</button>
    </header>

    <main :class="{ auth: !auth.isAuthenticated.value }">
      <router-view />
    </main>

    <nav class="tab-bar" v-if="auth.isAuthenticated.value">
      <div class="tab-bar-inner">
        <router-link to="/home">
          <Icon name="home" :size="20" />
          <span>Home</span>
        </router-link>
        <router-link to="/groups">
          <Icon name="people" :size="20" />
          <span>Groups</span>
        </router-link>
        <router-link to="/budgets">
          <Icon name="gauge" :size="20" />
          <span>Budget</span>
        </router-link>
        <router-link to="/goals">
          <Icon name="target" :size="20" />
          <span>Goals</span>
        </router-link>
        <router-link to="/settings">
          <Icon name="settings" :size="20" />
          <span>Server</span>
        </router-link>
      </div>
    </nav>

    <Toast />
  </div>
</template>
