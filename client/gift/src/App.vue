<script setup lang="ts">
import { auth } from './stores/auth'
import Icon from './components/Icon.vue'
import Toast from './components/Toast.vue'
</script>

<template>
  <div class="shell">
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
