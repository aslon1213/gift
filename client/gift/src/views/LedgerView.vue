<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import Icon from '../components/Icon.vue'
import Avatar from '../components/Avatar.vue'
import { borrowingApi, lendingApi, userApi } from '../api/endpoints'
import type { Credit, User } from '../api/types'
import { toast } from '../stores/toast'
import { userStore } from '../stores/user'
import { colorForId, currencySymbol, moneyWhole } from '../utils/format'
import { t } from '../i18n'

type Side = 'borrowed' | 'lent'
type FilterId = 'all' | 'active' | 'open' | 'settled'

const router = useRouter()
const side = ref<Side>('borrowed')
const filter = ref<FilterId>('all')

const borrowed = ref<Credit[]>([])
const lent = ref<Credit[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value = null
  try {
    const [b, l] = await Promise.all([borrowingApi.list(), lendingApi.list()])
    borrowed.value = b ?? []
    lent.value = l ?? []
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

const records = computed<Credit[]>(() => (side.value === 'borrowed' ? borrowed.value : lent.value))

// For a borrowing the counterparty is `from`; for a lending it's `to`.
function counterparty(c: Credit) {
  return side.value === 'borrowed' ? c.from : c.to
}

function counterpartyName(c: Credit): string {
  const cp = counterparty(c)
  if (!cp) return t('ledger.unnamed_counterparty')
  if (cp.is_oid) return cp.oid.slice(-6)
  return cp.str || t('ledger.unnamed_counterparty')
}

function counterpartyKey(c: Credit): string {
  const cp = counterparty(c)
  if (!cp) return ''
  return cp.is_oid ? cp.oid : cp.str
}

function isInApp(c: Credit): boolean {
  return counterparty(c)?.is_oid ?? false
}

function remaining(c: Credit): number {
  return Math.max(0, (c.amount || 0) - (c.resolved_amount || 0))
}

function pct(c: Credit): number {
  const a = c.amount || 0
  return a > 0 ? (c.resolved_amount || 0) / a : 0
}

function isSettled(c: Credit): boolean {
  return remaining(c) <= 0.005 && (c.amount || 0) > 0
}

function openCount(c: Credit): number {
  return (c.finance_requests ?? []).filter((r) => r.status === 'pending').length
}

interface Totals {
  count: number
  actual: number
  resolved: number
  outstanding: number
  open: number
}

function totalsFor(list: Credit[]): Totals {
  return list.reduce<Totals>(
    (a, c) => ({
      count: a.count + 1,
      actual: a.actual + (c.amount || 0),
      resolved: a.resolved + (c.resolved_amount || 0),
      outstanding: a.outstanding + remaining(c),
      open: a.open + openCount(c),
    }),
    { count: 0, actual: 0, resolved: 0, outstanding: 0, open: 0 },
  )
}

const borrowedTotals = computed(() => totalsFor(borrowed.value))
const lentTotals = computed(() => totalsFor(lent.value))
const sideTotals = computed(() => (side.value === 'borrowed' ? borrowedTotals.value : lentTotals.value))

const filtered = computed<Credit[]>(() => {
  return records.value.filter((c) => {
    if (filter.value === 'active') return !isSettled(c)
    if (filter.value === 'settled') return isSettled(c)
    if (filter.value === 'open') return openCount(c) > 0
    return true
  })
})

const sideEm = computed(() => (side.value === 'borrowed' ? t('ledger.borrowed_em') : t('ledger.lent_em')))
const sideColorVar = computed(() => (side.value === 'borrowed' ? 'var(--hot)' : 'var(--moss)'))

// Pick a sane currency to format card totals — first record's currency, else USD.
const sideCurrency = computed(() => records.value[0]?.currency || 'USD')

function open(c: Credit) {
  const id = String(c._id ?? c.id ?? '')
  if (!id) return
  router.push(`/${side.value === 'borrowed' ? 'borrowings' : 'lendings'}/${id}`)
}

// --- create modal --------------------------------------------------------
type CreateMode = 'borrowing' | 'lending'

const showCreate = ref(false)
const createMode = ref<CreateMode>('borrowing')
const creating = ref(false)
const createError = ref<string | null>(null)

const cpQuery = ref('')
const cpResults = ref<User[]>([])
const cpSearching = ref(false)
const cpPicked = ref<User | null>(null)
let cpTimer: ReturnType<typeof setTimeout> | null = null

const newAmount = ref<number | null>(null)
const newCurrency = ref<string>(userStore.currency.value || 'USD')
const newPaid = ref<number | null>(null)
const newDescription = ref('')
const newDate = ref<string>(new Date().toISOString().slice(0, 10))

function userId(u: User): string {
  return String(u._id ?? u.id ?? '')
}

function openCreate(mode: CreateMode) {
  createMode.value = mode
  showCreate.value = true
  createError.value = null
  cpQuery.value = ''
  cpResults.value = []
  cpPicked.value = null
  newAmount.value = null
  newPaid.value = null
  newDescription.value = ''
  newDate.value = new Date().toISOString().slice(0, 10)
  newCurrency.value = userStore.currency.value || 'USD'
}

function closeCreate() {
  showCreate.value = false
}

function pickUser(u: User) {
  cpPicked.value = u
  cpQuery.value = u.name || u.email
  cpResults.value = []
}

function clearPick() {
  cpPicked.value = null
  cpQuery.value = ''
  cpResults.value = []
}

function onCpInput() {
  // Picking a user freezes the input; further typing means the user wants to
  // re-search, so drop the lock.
  if (cpPicked.value) cpPicked.value = null
  if (cpTimer) clearTimeout(cpTimer)
  const q = cpQuery.value.trim()
  if (!q) {
    cpResults.value = []
    return
  }
  cpTimer = setTimeout(() => runCpSearch(q), 250)
}

async function runCpSearch(q: string) {
  cpSearching.value = true
  try {
    cpResults.value = (await userApi.search(q)) ?? []
  } catch {
    cpResults.value = []
  } finally {
    cpSearching.value = false
  }
}

const canSubmit = computed(() => {
  if (creating.value) return false
  if (!newAmount.value || newAmount.value <= 0) return false
  if (cpPicked.value) return true
  return cpQuery.value.trim().length > 0
})

async function submitCreate() {
  if (!canSubmit.value || !newAmount.value) return
  creating.value = true
  createError.value = null
  try {
    const common = {
      amount: newAmount.value,
      resolved_amount: newPaid.value ?? 0,
      currency: newCurrency.value,
      description: newDescription.value || undefined,
      date: new Date(newDate.value).toISOString(),
    }
    let created: Credit
    if (createMode.value === 'borrowing') {
      created = await borrowingApi.create(
        cpPicked.value
          ? { from_user_id: userId(cpPicked.value), ...common }
          : { from_name: cpQuery.value.trim(), ...common },
      )
      borrowed.value = [created, ...borrowed.value]
    } else {
      created = await lendingApi.create(
        cpPicked.value
          ? { to_user_id: userId(cpPicked.value), ...common }
          : { to_name: cpQuery.value.trim(), ...common },
      )
      lent.value = [created, ...lent.value]
    }
    side.value = createMode.value === 'borrowing' ? 'borrowed' : 'lent'
    toast.flash(t('ledger.created_toast'))
    closeCreate()
  } catch (e) {
    createError.value = e instanceof Error ? e.message : 'Create failed'
  } finally {
    creating.value = false
  }
}

watch(
  () => router.currentRoute.value.query.side,
  (s) => {
    if (s === 'lent' || s === 'borrowed') side.value = s
  },
  { immediate: true },
)

onMounted(load)
</script>

<template>
  <section class="ledger">
    <div class="eyebrow top-eye">{{ t('ledger.eyebrow') }}</div>
    <h1 class="hero">
      {{ t('ledger.you_owe') }}<br />
      <em :style="{ color: sideColorVar }">{{ sideEm }}</em>
    </h1>
    <p class="lead">{{ t('ledger.lead') }}</p>

    <p v-if="loading" class="muted">{{ t('common.loading') }}</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <template v-else>
      <!-- Side toggle -->
      <div class="side-toggle">
        <button
          v-for="s in (['borrowed', 'lent'] as const)"
          :key="s"
          class="side-btn"
          :class="{ on: side === s }"
          @click="side = s"
        >
          <div class="side-row">
            <span class="dot" :style="{ background: s === 'borrowed' ? 'var(--hot)' : 'var(--moss)' }"></span>
            <span class="side-label">{{ s === 'borrowed' ? t('ledger.borrowed') : t('ledger.lent') }}</span>
          </div>
          <div class="side-money" :style="{ color: side === s ? (s === 'borrowed' ? 'var(--hot)' : 'var(--moss)') : 'var(--ink-mute)' }">
            {{ moneyWhole((s === 'borrowed' ? borrowedTotals : lentTotals).outstanding, sideCurrency) }}
          </div>
          <div class="side-meta">
            {{ t('ledger.records_open', {
              count: (s === 'borrowed' ? borrowedTotals : lentTotals).count,
              open: (s === 'borrowed' ? borrowedTotals : lentTotals).open,
            }) }}
          </div>
        </button>
      </div>

      <!-- Stat row -->
      <div class="stat-row">
        <div class="stat">
          <div class="stat-label">{{ t('ledger.actual') }}</div>
          <div class="stat-value">{{ moneyWhole(sideTotals.actual, sideCurrency) }}</div>
        </div>
        <div class="stat">
          <div class="stat-label">{{ t('ledger.resolved') }}</div>
          <div class="stat-value moss">{{ moneyWhole(sideTotals.resolved, sideCurrency) }}</div>
        </div>
        <div class="stat">
          <div class="stat-label">{{ t('ledger.outstanding') }}</div>
          <div class="stat-value" :style="{ color: sideColorVar }">{{ moneyWhole(sideTotals.outstanding, sideCurrency) }}</div>
        </div>
      </div>

      <!-- Filter chips -->
      <div class="pill-row" style="margin-top: 18px">
        <button
          v-for="f in (['all', 'active', 'open', 'settled'] as const)"
          :key="f"
          type="button"
          class="pill"
          :class="{ on: filter === f }"
          @click="filter = f"
        >
          {{ t('ledger.filter.' + f) }}
        </button>
      </div>

      <!-- List -->
      <div v-if="!filtered.length" class="empty" style="margin-top: 18px">
        {{ t('ledger.empty') }}
      </div>
      <div v-else class="list">
        <button
          v-for="c in filtered"
          :key="String(c._id ?? c.id)"
          class="rec"
          @click="open(c)"
        >
          <div class="rec-head">
            <Avatar
              :name="counterpartyName(c)"
              :color="isInApp(c) ? colorForId(counterpartyKey(c)) : 'var(--ink-ghost)'"
              :size="38"
            />
            <div class="rec-mid">
              <div class="rec-title">
                <span class="cp-name">{{ counterpartyName(c) }}</span>
                <span class="cp-tag" :class="{ inapp: isInApp(c) }">
                  {{ isInApp(c) ? t('ledger.in_app') : t('ledger.off_app') }}
                </span>
              </div>
              <div class="rec-sub">
                {{ c.description || '—' }}
              </div>
            </div>
            <div class="rec-amount" :style="{ color: isSettled(c) ? 'var(--ink-mute)' : sideColorVar }">
              <template v-if="isSettled(c)">—</template>
              <template v-else>{{ moneyWhole(remaining(c), c.currency) }}</template>
              <div class="rec-amount-sub">
                {{ isSettled(c) ? t('ledger.settled') : t('ledger.outstanding_label') }}
              </div>
            </div>
          </div>

          <div class="progress-wrap">
            <div class="progress-meta">
              <span>{{ t('ledger.resolved_of', { resolved: moneyWhole(c.resolved_amount || 0, c.currency), actual: moneyWhole(c.amount || 0, c.currency) }) }}</span>
              <span>{{ Math.round(pct(c) * 100) }}%</span>
            </div>
            <div class="progress-track">
              <div
                class="progress-fill"
                :style="{ width: `${pct(c) * 100}%`, background: isSettled(c) ? 'var(--moss)' : sideColorVar }"
              ></div>
            </div>
          </div>

          <div v-if="openCount(c) > 0" class="open-strip">
            <span class="dot" style="background: var(--hot)"></span>
            <span>
              {{ t(openCount(c) === 1 ? 'ledger.open_amendments' : 'ledger.open_amendments_plural', { n: openCount(c) }) }}
            </span>
            <span class="review">{{ t('ledger.review') }}</span>
          </div>
        </button>
      </div>

      <!-- New record CTAs -->
      <div class="cta-row">
        <button class="btn btn-secondary btn-lg btn-block" @click="openCreate('borrowing')">
          <Icon name="arrowDown" :size="16" /> {{ t('ledger.log_borrowing') }}
        </button>
        <button class="btn btn-secondary btn-lg btn-block" @click="openCreate('lending')">
          <Icon name="arrowUp" :size="16" /> {{ t('ledger.log_lending') }}
        </button>
      </div>
    </template>

    <!-- Create modal -->
    <Teleport to="body">
      <div v-if="showCreate" class="modal-backdrop" @click.self="closeCreate">
        <div class="modal">
          <div class="modal-header">
            <button class="linklike" @click="closeCreate">
              <Icon name="close" :size="16" /> {{ t('common.cancel').toUpperCase() }}
            </button>
            <div class="eyebrow">
              {{ createMode === 'borrowing' ? t('ledger.create_borrowing_eyebrow') : t('ledger.create_lending_eyebrow') }}
            </div>
          </div>
          <div class="modal-body">
            <h1 class="display" style="margin: 0">
              {{ createMode === 'borrowing' ? t('ledger.create_borrowing_title') : t('ledger.create_lending_title') }}
            </h1>

            <p v-if="createError" class="error" style="margin-top: 10px">{{ createError }}</p>

            <!-- Counterparty -->
            <label class="field" style="margin-top: 20px">
              <span>{{ createMode === 'borrowing' ? t('ledger.from_who') : t('ledger.to_who') }}</span>
              <input
                v-model="cpQuery"
                :placeholder="t('ledger.counterparty_placeholder')"
                @input="onCpInput"
              />
            </label>

            <div v-if="cpPicked" class="picked-row">
              <div>{{ t('ledger.counterparty_picked', { name: cpPicked.name || cpPicked.email }) }}</div>
              <button class="linklike" type="button" @click="clearPick">
                {{ t('ledger.counterparty_clear') }}
              </button>
            </div>

            <div v-else-if="cpSearching" class="muted small" style="margin-top: 8px">
              {{ t('ledger.searching') }}
            </div>

            <div v-else-if="cpResults.length" class="search-list">
              <button
                v-for="u in cpResults"
                :key="userId(u)"
                type="button"
                class="search-row"
                @click="pickUser(u)"
              >
                <div class="search-name">{{ u.name }}</div>
                <div class="search-email">{{ u.email }}</div>
              </button>
            </div>

            <div
              v-else-if="cpQuery.trim()"
              class="muted small"
              style="margin-top: 8px"
            >
              {{ t('ledger.counterparty_freeform', { name: cpQuery.trim() }) }}
            </div>

            <!-- Amount + currency -->
            <div class="stack-form split" style="margin-top: 18px">
              <label class="field">
                <span>{{ t('ledger.amount_label') }} ({{ newCurrency }})</span>
                <input
                  v-model.number="newAmount"
                  type="number"
                  min="0"
                  step="0.01"
                  :placeholder="currencySymbol(newCurrency) + '100'"
                />
              </label>
              <label class="field">
                <span>{{ t('ledger.already_paid_label') }}</span>
                <input
                  v-model.number="newPaid"
                  type="number"
                  min="0"
                  step="0.01"
                  placeholder="0"
                />
              </label>
            </div>

            <label class="field" style="margin-top: 14px">
              <span>{{ t('ledger.description_label') }}</span>
              <input v-model="newDescription" />
            </label>

            <label class="field" style="margin-top: 14px">
              <span>{{ t('ledger.date_label') }}</span>
              <input v-model="newDate" type="date" />
            </label>

            <div style="margin-top: 28px">
              <button
                class="btn btn-primary btn-lg btn-block"
                :disabled="!canSubmit"
                @click="submitCreate"
              >
                <Icon name="check" :size="18" />
                {{ creating ? t('ledger.creating') : (createMode === 'borrowing' ? t('ledger.create_borrowing') : t('ledger.create_lending')) }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<style scoped>
.ledger {
  padding: 4px 0;
}
.top-eye {
  margin-bottom: 6px;
}
.lead {
  font-size: 13px;
  color: var(--ink-soft);
  margin: 10px 0 0;
  max-width: 320px;
  line-height: 1.45;
}

.side-toggle {
  margin-top: 22px;
  background: var(--paper-deep);
  border: 1px solid var(--line);
  border-radius: var(--r);
  padding: 4px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
}
.side-btn {
  background: transparent;
  border: none;
  border-radius: var(--r-sm);
  padding: 12px 14px;
  cursor: pointer;
  text-align: left;
  color: var(--ink-mute);
  transition: background 0.15s;
}
.side-btn.on {
  background: var(--paper);
  color: var(--ink);
  box-shadow: 0 1px 0 rgba(20, 23, 31, 0.04), 0 1px 3px rgba(20, 23, 31, 0.05);
}
.side-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.side-row .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}
.side-label {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.1em;
}
.side-money {
  font-family: var(--serif);
  font-size: 22px;
  margin-top: 4px;
  line-height: 1;
}
.side-meta {
  font-family: var(--mono);
  font-size: 9px;
  color: var(--ink-mute);
  letter-spacing: 0.06em;
  margin-top: 4px;
}

.stat-row {
  margin-top: 14px;
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  border: 1px solid var(--line);
  border-radius: var(--r);
  background: var(--paper);
}
.stat {
  padding: 12px 14px;
  border-left: 1px solid var(--line);
}
.stat:first-child {
  border-left: none;
}
.stat-label {
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.1em;
  color: var(--ink-mute);
}
.stat-value {
  font-family: var(--serif);
  font-size: 18px;
  margin-top: 3px;
  line-height: 1;
}
.stat-value.moss {
  color: var(--moss);
}

.list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 14px;
}
.rec {
  width: 100%;
  text-align: left;
  background: var(--paper);
  border: 1px solid var(--line);
  border-radius: var(--r-lg);
  padding: 14px 16px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.rec:hover {
  border-color: var(--line-hard);
}
.rec-head {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 12px;
  align-items: center;
}
.rec-mid {
  min-width: 0;
}
.rec-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 3px;
}
.cp-name {
  font-family: var(--serif);
  font-size: 18px;
  color: var(--ink);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cp-tag {
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.08em;
  color: var(--ink-mute);
  background: var(--line);
  padding: 2px 6px;
  border-radius: 4px;
}
.cp-tag.inapp {
  color: var(--moss);
  background: rgba(47, 95, 79, 0.08);
}
.rec-sub {
  font-family: var(--mono);
  font-size: 10px;
  color: var(--ink-mute);
  letter-spacing: 0.06em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.rec-amount {
  text-align: right;
  font-family: var(--serif);
  font-size: 22px;
  line-height: 1;
}
.rec-amount-sub {
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.08em;
  color: var(--ink-mute);
  margin-top: 4px;
}

.progress-wrap {
  margin-top: 2px;
}
.progress-meta {
  display: flex;
  justify-content: space-between;
  font-family: var(--mono);
  font-size: 10px;
  color: var(--ink-mute);
  letter-spacing: 0.06em;
  margin-bottom: 6px;
}
.progress-track {
  height: 4px;
  background: var(--line);
  border-radius: 2px;
  overflow: hidden;
}
.progress-fill {
  height: 100%;
}

.open-strip {
  margin-top: 2px;
  padding-top: 10px;
  border-top: 1px dashed var(--line);
  display: flex;
  align-items: center;
  gap: 6px;
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.08em;
  color: var(--hot);
}
.open-strip .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  box-shadow: 0 0 0 3px rgba(214, 73, 51, 0.1);
}
.open-strip .review {
  color: var(--ink);
  margin-left: auto;
}

.cta-row {
  margin-top: 22px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.picked-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 8px;
  padding: 10px 12px;
  background: var(--paper-deep);
  border: 1px solid var(--line);
  border-radius: var(--r-sm);
  font-family: var(--mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  color: var(--ink);
}

.search-list {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 220px;
  overflow-y: auto;
}
.search-row {
  text-align: left;
  background: var(--paper);
  border: 1px solid var(--line);
  border-radius: var(--r-sm);
  padding: 10px 12px;
  cursor: pointer;
  transition: border-color 0.15s;
}
.search-row:hover {
  border-color: var(--ink);
}
.search-name {
  font-size: 14px;
  color: var(--ink);
}
.search-email {
  font-family: var(--mono);
  font-size: 10px;
  color: var(--ink-mute);
  margin-top: 2px;
  letter-spacing: 0.04em;
}
</style>
