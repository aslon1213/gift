<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import Icon from '../components/Icon.vue'
import type { IconName } from '../components/icons'
import { auth } from '../stores/auth'
import { server } from '../stores/server'
import { authApi, settingsApi } from '../api/endpoints'
import type { SettingsInfo } from '../api/types'

const router = useRouter()

const info = ref<SettingsInfo | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value = null
  try {
    info.value = await settingsApi.get()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

onMounted(load)

const host = computed(
  () => info.value?.server.host || server.baseUrl.value || 'home.lan:4747',
)

function formatBytes(n: number): string {
  if (!n) return '—'
  if (n >= 1024 * 1024 * 1024) return (n / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
  if (n >= 1024 * 1024) return (n / (1024 * 1024)).toFixed(1) + ' MB'
  if (n >= 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}

interface Row {
  label: string
  value: string
  icon: IconName
  danger?: boolean
  onClick?: () => void
}

const accountRows = computed<Row[]>(() => [
  {
    label: 'Profile',
    value: info.value?.profile.email || auth.user.value?.email || '—',
    icon: 'people',
  },
  { label: 'Notifications', value: '—', icon: 'bell' },
  { label: 'Appearance', value: 'Paper', icon: 'gauge' },
])

const serverRows = computed<Row[]>(() => [
  { label: 'Hosting', value: host.value, icon: 'server' },
  {
    label: 'Backups',
    value: info.value ? formatBytes(info.value.stats.db_size_bytes) + ' snapshot' : '—',
    icon: 'arrowDown',
  },
  {
    label: 'Version',
    value: info.value?.server.version ?? '—',
    icon: 'arrowUp',
  },
  {
    label: 'Goroutines',
    value: info.value ? String(info.value.stats.goroutines) : '—',
    icon: 'settings',
  },
])

async function signOut() {
  try {
    await authApi.logout()
  } catch {
    // ignore — clear client state regardless
  }
  auth.clear()
  router.push('/login')
}

function confirmReset() {
  if (
    confirm(
      'Reset the database? This is irreversible — all groups, spendings, budgets, goals will be lost.',
    )
  ) {
    alert('Not yet wired — endpoint forthcoming.')
  }
}
</script>

<template>
  <section class="settings">
    <header class="row spread top-actions">
      <span class="eyebrow">SELF-HOST · ADMIN</span>
    </header>

    <h1 class="hero">
      Your <em>server.</em>
    </h1>

    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <template v-else-if="info">
      <!-- Server identity card -->
      <div class="identity card-ink">
        <div class="row" style="gap: 8px; margin-bottom: 10px">
          <div class="pulse" :class="{ offline: !info.server.online }"></div>
          <div class="id-label">
            {{ info.server.online ? 'ONLINE' : 'OFFLINE' }} · AUTOSTART
          </div>
        </div>
        <div class="id-host">{{ host }}</div>
        <div class="id-stats">
          <div>
            <div class="id-stat-label">UPTIME</div>
            <div class="id-stat-value">{{ info.server.uptime || '—' }}</div>
          </div>
          <div>
            <div class="id-stat-label">VERSION</div>
            <div class="id-stat-value">{{ info.server.version }}</div>
          </div>
        </div>
      </div>

      <!-- Stat grid -->
      <div class="stats">
        <div class="stat">
          <div class="stat-label">DB SIZE</div>
          <div class="stat-value pos">{{ formatBytes(info.stats.db_size_bytes) }}</div>
          <div class="stat-sub">mongodb · dbStats</div>
        </div>
        <div class="stat">
          <div class="stat-label">USERS</div>
          <div class="stat-value">{{ info.stats.users }}</div>
          <div class="stat-sub">{{ info.stats.groups }} groups</div>
        </div>
        <div class="stat">
          <div class="stat-label">MEM</div>
          <div class="stat-value pos">{{ info.stats.mem_mb }} MB</div>
          <div class="stat-sub">{{ info.stats.goroutines }} goroutines</div>
        </div>
        <div class="stat">
          <div class="stat-label">BUDGETS · GOALS</div>
          <div class="stat-value">
            {{ info.stats.budgets }} · {{ info.stats.goals }}
          </div>
          <div class="stat-sub">you</div>
        </div>
      </div>

      <div class="section-label eyebrow">ACCOUNT</div>
      <button v-for="r in accountRows" :key="r.label" class="setting-row">
        <Icon :name="r.icon" :size="18" class="r-icon" />
        <span class="r-label">{{ r.label }}</span>
        <span class="r-value">{{ r.value }}</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>
      <button class="setting-row" @click="signOut">
        <Icon name="close" :size="18" class="r-icon" />
        <span class="r-label">Sign out</span>
        <span class="r-value">{{ info.profile.email }}</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>

      <div class="section-label eyebrow">SERVER</div>
      <button v-for="r in serverRows" :key="r.label" class="setting-row">
        <Icon :name="r.icon" :size="18" class="r-icon" />
        <span class="r-label">{{ r.label }}</span>
        <span class="r-value">{{ r.value }}</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>

      <div class="section-label eyebrow danger">DANGER ZONE</div>
      <button class="setting-row">
        <Icon name="arrowDown" :size="18" class="r-icon" />
        <span class="r-label">Export all data</span>
        <span class="r-value">.json · .csv</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>
      <button class="setting-row danger" @click="confirmReset">
        <Icon name="close" :size="18" class="r-icon" />
        <span class="r-label">Reset database</span>
        <span class="r-value">irreversible</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>

      <div class="foot center">
        <div>
          gift · self-hosted · MIT ·
          <span v-if="info.server.started_at">
            up since {{ new Date(info.server.started_at).toLocaleString() }}
          </span>
        </div>
        <div>no subscription · no telemetry · no surprises</div>
      </div>
    </template>
  </section>
</template>

<style scoped>
.settings {
  padding: 4px 0;
}

.top-actions {
  margin-bottom: 8px;
}

.hero {
  font-size: 44px;
}

.identity {
  margin-top: 22px;
  padding: 20px 22px;
  border-radius: var(--r-lg);
}

.pulse {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #4ade80;
  box-shadow: 0 0 0 3px rgba(74, 222, 128, 0.2);
}

.pulse.offline {
  background: var(--hot);
  box-shadow: 0 0 0 3px var(--hot-soft);
}

.id-label {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  color: rgba(245, 241, 232, 0.6);
}

.id-host {
  font-family: var(--serif);
  font-size: 26px;
  line-height: 1.1;
}

.id-stats {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid rgba(245, 241, 232, 0.1);
}

.id-stat-label {
  font-family: var(--mono);
  font-size: 9px;
  color: rgba(245, 241, 232, 0.55);
  letter-spacing: 0.1em;
}

.id-stat-value {
  font-family: var(--serif);
  font-size: 20px;
  margin-top: 4px;
}

.stats {
  margin-top: 18px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.section-label {
  margin: 28px 0 4px;
}

.section-label.danger {
  color: var(--hot);
}

.setting-row {
  display: grid;
  grid-template-columns: 28px 1fr auto auto;
  gap: 14px;
  align-items: center;
  padding: 14px 0;
  border: none;
  background: none;
  width: 100%;
  text-align: left;
  cursor: pointer;
  border-bottom: 1px solid var(--line);
  color: var(--ink);
}

.setting-row.danger {
  color: var(--hot);
}

.r-icon {
  color: var(--ink-soft);
}

.setting-row.danger .r-icon {
  color: var(--hot);
}

.r-label {
  font-family: var(--sans);
  font-size: 15px;
  font-weight: 500;
}

.r-value {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--ink-mute);
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.r-chev {
  color: var(--ink-ghost);
}

.foot.center {
  margin-top: 32px;
  text-align: center;
  padding-bottom: 16px;
}

.foot.center > div {
  font-family: var(--mono);
  font-size: 10px;
  color: var(--ink-ghost);
  letter-spacing: 0.08em;
  margin-top: 4px;
}
</style>
