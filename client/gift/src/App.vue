<script setup lang="ts">
import { auth } from './stores/auth'
import { t } from './i18n'
import Icon from './components/Icon.vue'
import Toast from './components/Toast.vue'
</script>

<template>
  <div class="shell">
    <header class="topbar desktop-only" v-if="auth.isAuthenticated.value">
      <router-link to="/home" class="brand">
        <span class="dot"></span>
        <b>gift</b>
        <span class="brand-sub">· self-hosted</span>
      </router-link>
      <nav class="desktop-nav">
        <router-link to="/home">
          <Icon name="home" :size="14" />
          <span>{{ t('nav.home') }}</span>
        </router-link>
        <router-link to="/groups">
          <Icon name="people" :size="14" />
          <span>{{ t('nav.groups') }}</span>
        </router-link>
        <router-link to="/spendings">
          <Icon name="wallet" :size="14" />
          <span>{{ t('nav.spendings') }}</span>
        </router-link>
        <router-link to="/incomes">
          <Icon name="arrowDown" :size="14" />
          <span>{{ t('nav.incomes') }}</span>
        </router-link>
        <router-link to="/budgets">
          <Icon name="gauge" :size="14" />
          <span>{{ t('nav.budget') }}</span>
        </router-link>
        <router-link to="/goals">
          <Icon name="target" :size="14" />
          <span>{{ t('nav.goals') }}</span>
        </router-link>
        <router-link to="/ledger">
          <Icon name="wallet" :size="14" />
          <span>{{ t('nav.ledger') }}</span>
        </router-link>
        <router-link to="/settings">
          <Icon name="settings" :size="14" />
          <span>{{ t('nav.server') }}</span>
        </router-link>
      </nav>
    </header>

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

<style scoped>
.topbar.desktop-only {
  display: none;
}
.brand {
  text-decoration: none;
  color: var(--ink-mute);
  display: flex;
  align-items: center;
  gap: 10px;
}
.brand b {
  font-family: var(--serif);
  font-style: italic;
  font-size: 18px;
  letter-spacing: -0.01em;
  color: var(--ink);
  text-transform: none;
}
.brand-sub {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.1em;
  color: var(--ink-ghost);
  text-transform: uppercase;
}
.desktop-nav a {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
@media (min-width: 900px) {
  .topbar.desktop-only {
    display: flex;
  }
}
</style>
