<script setup lang="ts">
import { auth } from './stores/auth'
import { t } from './i18n'
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
          <span>{{ t('nav.home') }}</span>
        </router-link>
        <router-link to="/groups">
          <Icon name="people" :size="20" />
          <span>{{ t('nav.groups') }}</span>
        </router-link>
        <router-link to="/budgets">
          <Icon name="gauge" :size="20" />
          <span>{{ t('nav.budget') }}</span>
        </router-link>
        <router-link to="/goals">
          <Icon name="target" :size="20" />
          <span>{{ t('nav.goals') }}</span>
        </router-link>
        <router-link to="/settings">
          <Icon name="settings" :size="20" />
          <span>{{ t('nav.server') }}</span>
        </router-link>
      </div>
    </nav>

    <Toast />
  </div>
</template>
