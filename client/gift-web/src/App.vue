<script setup lang="ts">
import { useRouter } from 'vue-router'
import { auth } from './stores/auth'
import { authApi } from './api/endpoints'

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
      <div class="brand">Gift</div>
      <nav class="desktop-nav">
        <router-link to="/dashboard">Dashboard</router-link>
        <router-link to="/groups">Groups</router-link>
        <router-link to="/spendings">Spendings</router-link>
        <router-link to="/incomes">Incomes</router-link>
      </nav>
      <button class="linklike" @click="logout">Logout</button>
    </header>

    <main :class="{ auth: !auth.isAuthenticated.value }">
      <router-view />
    </main>

    <nav class="bottom-nav" v-if="auth.isAuthenticated.value">
      <router-link to="/dashboard">
        <span class="icon">▦</span>
        <span>Home</span>
      </router-link>
      <router-link to="/groups">
        <span class="icon">◐</span>
        <span>Groups</span>
      </router-link>
      <router-link to="/spendings">
        <span class="icon">−</span>
        <span>Spend</span>
      </router-link>
      <router-link to="/incomes">
        <span class="icon">+</span>
        <span>Income</span>
      </router-link>
    </nav>
  </div>
</template>
