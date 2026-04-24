<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import Icon from '../components/Icon.vue'
import { auth } from '../stores/auth'
import { server } from '../stores/server'
import { userStore } from '../stores/user'
import { toast } from '../stores/toast'
import { authApi, settingsApi } from '../api/endpoints'
import type { CurrencyCode, SettingsInfo } from '../api/types'
import { CURRENCY_CODES, CURRENCY_META } from '../utils/format'
import { i18n, LOCALE_META, SUPPORTED_LOCALES, t, type Locale } from '../i18n'

const router = useRouter()

const info = ref<SettingsInfo | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const showCurrency = ref(false)
const showLanguage = ref(false)
const showExport = ref(false)

async function load() {
  loading.value = true
  error.value = null
  try {
    info.value = await settingsApi.get()
    // Sync profile into the user store so other pages pick up the preferred currency.
    if (info.value) {
      void userStore.load()
    }
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

const currentCurrency = computed<CurrencyCode>(() => userStore.currency.value)
const currentLocale = computed<Locale>(() => i18n.locale.value)

function formatBytes(n: number): string {
  if (!n) return '—'
  if (n >= 1024 * 1024 * 1024) return (n / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
  if (n >= 1024 * 1024) return (n / (1024 * 1024)).toFixed(1) + ' MB'
  if (n >= 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}

function pickCurrency(code: CurrencyCode) {
  userStore.setCurrency(code)
  showCurrency.value = false
  toast.flash(`${t('common.currency')}: ${code}`)
}

function pickLanguage(l: Locale) {
  i18n.setLocale(l)
  showLanguage.value = false
  toast.flash(`${t('common.language')}: ${LOCALE_META[l].label}`)
}

async function doExport(format: 'json' | 'csv') {
  showExport.value = false
  try {
    await settingsApi.export(format)
    toast.flash(format === 'json' ? t('settings.export_json') : t('settings.export_csv'))
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Export failed'
  }
}

async function signOut() {
  try {
    await authApi.logout()
  } catch {
    // ignore — clear client state regardless
  }
  auth.clear()
  userStore.clear()
  router.push('/login')
}
</script>

<template>
  <section class="settings">
    <h1 class="hero">{{ t('settings.your_server') }}</h1>

    <p v-if="loading" class="muted">{{ t('common.loading') }}</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <template v-else-if="info">
      <!-- Server identity card -->
      <div class="identity card-ink">
        <div class="row" style="gap: 8px; margin-bottom: 10px">
          <div class="pulse" :class="{ offline: !info.server.online }"></div>
          <div class="id-label">
            {{ info.server.online ? t('settings.online') : t('settings.offline') }}
            · {{ t('settings.autostart') }}
          </div>
        </div>
        <div class="id-host">{{ host }}</div>
        <div class="id-stats">
          <div>
            <div class="id-stat-label">{{ t('settings.uptime') }}</div>
            <div class="id-stat-value">{{ info.server.uptime || '—' }}</div>
          </div>
          <div>
            <div class="id-stat-label">{{ t('settings.version') }}</div>
            <div class="id-stat-value">{{ info.server.version }}</div>
          </div>
        </div>
      </div>

      <!-- Stat grid -->
      <div class="stats">
        <div class="stat">
          <div class="stat-label">{{ t('settings.db_size') }}</div>
          <div class="stat-value pos">{{ formatBytes(info.stats.db_size_bytes) }}</div>
          <div class="stat-sub">mongodb · dbStats</div>
        </div>
        <div class="stat">
          <div class="stat-label">{{ t('settings.users') }}</div>
          <div class="stat-value">{{ info.stats.users }}</div>
          <div class="stat-sub">{{ info.stats.groups }} {{ t('nav.groups').toLowerCase() }}</div>
        </div>
        <div class="stat">
          <div class="stat-label">{{ t('settings.mem') }}</div>
          <div class="stat-value pos">{{ info.stats.mem_mb }} MB</div>
          <div class="stat-sub">{{ info.stats.goroutines }} {{ t('settings.goroutines').toLowerCase() }}</div>
        </div>
        <div class="stat">
          <div class="stat-label">{{ t('settings.budgets_goals') }}</div>
          <div class="stat-value">
            {{ info.stats.budgets }} · {{ info.stats.goals }}
          </div>
          <div class="stat-sub">you</div>
        </div>
      </div>

      <div class="section-label eyebrow">{{ t('settings.account') }}</div>
      <button class="setting-row">
        <Icon name="people" :size="18" class="r-icon" />
        <span class="r-label">{{ t('settings.profile') }}</span>
        <span class="r-value">{{ info.profile.email }}</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>
      <button class="setting-row" @click="showCurrency = true">
        <Icon name="wallet" :size="18" class="r-icon" />
        <span class="r-label">{{ t('settings.currency') }}</span>
        <span class="r-value">
          {{ CURRENCY_META[currentCurrency].symbol }} {{ currentCurrency }}
        </span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>
      <button class="setting-row" @click="showLanguage = true">
        <Icon name="gift" :size="18" class="r-icon" />
        <span class="r-label">{{ t('settings.language') }}</span>
        <span class="r-value">{{ LOCALE_META[currentLocale].label }}</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>
      <button class="setting-row">
        <Icon name="bell" :size="18" class="r-icon" />
        <span class="r-label">{{ t('settings.notifications') }}</span>
        <span class="r-value">—</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>
      <button class="setting-row" @click="signOut">
        <Icon name="close" :size="18" class="r-icon" />
        <span class="r-label">{{ t('settings.sign_out') }}</span>
        <span class="r-value">{{ info.profile.email }}</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>

      <div class="section-label eyebrow">{{ t('settings.server') }}</div>
      <button class="setting-row">
        <Icon name="server" :size="18" class="r-icon" />
        <span class="r-label">{{ t('settings.hosting') }}</span>
        <span class="r-value">{{ host }}</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>
      <button class="setting-row">
        <Icon name="arrowDown" :size="18" class="r-icon" />
        <span class="r-label">{{ t('settings.backups') }}</span>
        <span class="r-value">{{ formatBytes(info.stats.db_size_bytes) }}</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>
      <button class="setting-row">
        <Icon name="arrowUp" :size="18" class="r-icon" />
        <span class="r-label">{{ t('settings.version') }}</span>
        <span class="r-value">{{ info.server.version }}</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>
      <button class="setting-row">
        <Icon name="settings" :size="18" class="r-icon" />
        <span class="r-label">{{ t('settings.goroutines') }}</span>
        <span class="r-value">{{ info.stats.goroutines }}</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>

      <div class="section-label eyebrow">{{ t('settings.data') }}</div>
      <button class="setting-row" @click="showExport = true">
        <Icon name="arrowDown" :size="18" class="r-icon" />
        <span class="r-label">{{ t('settings.export_all') }}</span>
        <span class="r-value">.json · .zip</span>
        <Icon name="chevR" :size="14" class="r-chev" />
      </button>

      <div class="foot center">
        <div>
          gift · self-hosted · MIT ·
          <span v-if="info.server.started_at">
            {{ t('settings.up_since', { when: new Date(info.server.started_at).toLocaleString() }) }}
          </span>
        </div>
        <div>{{ t('settings.tagline') }}</div>
      </div>
    </template>

    <!-- Currency picker -->
    <Teleport to="body">
      <div v-if="showCurrency" class="modal-backdrop" @click.self="showCurrency = false">
        <div class="modal">
          <div class="modal-header">
            <button class="linklike" @click="showCurrency = false">
              <Icon name="close" :size="16" /> {{ t('common.cancel') }}
            </button>
            <div class="eyebrow">{{ t('settings.currency') }}</div>
          </div>
          <div class="modal-body">
            <div class="picker-list">
              <button
                v-for="code in CURRENCY_CODES"
                :key="code"
                class="picker-row"
                :class="{ on: currentCurrency === code }"
                @click="pickCurrency(code)"
              >
                <span class="picker-symbol">{{ CURRENCY_META[code].symbol }}</span>
                <span class="picker-label">{{ code }}</span>
                <Icon v-if="currentCurrency === code" name="check" :size="18" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Language picker -->
    <Teleport to="body">
      <div v-if="showLanguage" class="modal-backdrop" @click.self="showLanguage = false">
        <div class="modal">
          <div class="modal-header">
            <button class="linklike" @click="showLanguage = false">
              <Icon name="close" :size="16" /> {{ t('common.cancel') }}
            </button>
            <div class="eyebrow">{{ t('settings.language') }}</div>
          </div>
          <div class="modal-body">
            <div class="picker-list">
              <button
                v-for="l in SUPPORTED_LOCALES"
                :key="l"
                class="picker-row"
                :class="{ on: currentLocale === l }"
                @click="pickLanguage(l)"
              >
                <span class="picker-symbol">{{ LOCALE_META[l].short }}</span>
                <span class="picker-label">{{ LOCALE_META[l].label }}</span>
                <Icon v-if="currentLocale === l" name="check" :size="18" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Export format picker -->
    <Teleport to="body">
      <div v-if="showExport" class="modal-backdrop" @click.self="showExport = false">
        <div class="modal">
          <div class="modal-header">
            <button class="linklike" @click="showExport = false">
              <Icon name="close" :size="16" /> {{ t('common.cancel') }}
            </button>
            <div class="eyebrow">{{ t('settings.export_all') }}</div>
          </div>
          <div class="modal-body">
            <div class="picker-list">
              <button class="picker-row" @click="doExport('json')">
                <span class="picker-symbol">{ }</span>
                <span class="picker-label">{{ t('settings.export_json') }}</span>
              </button>
              <button class="picker-row" @click="doExport('csv')">
                <span class="picker-symbol">.csv</span>
                <span class="picker-label">{{ t('settings.export_csv') }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<style scoped>
.settings {
  padding: 4px 0;
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
  font-size: clamp(18px, 5vw, 22px);
  line-height: 1.15;
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.id-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid rgba(245, 241, 232, 0.1);
}

.id-stats > div {
  min-width: 0;
}

.id-stat-label {
  font-family: var(--mono);
  font-size: 9px;
  color: rgba(245, 241, 232, 0.55);
  letter-spacing: 0.1em;
}

.id-stat-value {
  font-family: var(--serif);
  font-size: clamp(15px, 4.5vw, 18px);
  margin-top: 4px;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: clip;
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

.r-icon {
  color: var(--ink-soft);
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

.picker-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.picker-row {
  display: grid;
  grid-template-columns: 48px 1fr 22px;
  gap: 14px;
  align-items: center;
  padding: 14px 18px;
  background: var(--paper);
  border: 1px solid var(--line);
  border-radius: var(--r);
  cursor: pointer;
  text-align: left;
  color: var(--ink);
  transition: border-color 0.15s, background 0.15s;
}

.picker-row:hover {
  border-color: var(--ink);
}

.picker-row.on {
  background: var(--ink);
  color: var(--paper);
  border-color: var(--ink);
}

.picker-symbol {
  font-family: var(--serif);
  font-size: 24px;
  font-weight: 400;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.picker-label {
  font-family: var(--sans);
  font-size: 15px;
  font-weight: 500;
}
</style>
