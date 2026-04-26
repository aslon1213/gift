<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { borrowingApi, groupApi, incomeApi, lendingApi, spendingApi } from '../api/endpoints'
import type { Credit, Group, Income, Spending } from '../api/types'
import { auth } from '../stores/auth'
import Icon from '../components/Icon.vue'
import Sparkline from '../components/Sparkline.vue'
import NotificationsSheet from '../components/NotificationsSheet.vue'
import type { IconName } from '../components/icons'
import { money, moneyWhole, signed, signedWhole } from '../utils/format'
import { lastNDays, formatDay, groupBy, sumBy } from '../utils/charts'
import { userStore } from '../stores/user'
import { t } from '../i18n'

const router = useRouter()

const groups = ref<Group[]>([])
const spendings = ref<Spending[]>([])
const incomes = ref<Income[]>([])
const borrowed = ref<Credit[]>([])
const lent = ref<Credit[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const showNotifications = ref(false)

const monthLabel = computed(() =>
  new Date().toLocaleDateString('en-US', { month: 'long', year: 'numeric' }).toUpperCase(),
)

// User's preferred currency (USD/EUR/UZS) drives all aggregate amounts on this page.
const currency = computed(() => userStore.currency.value)

const inMonth = (d: Date) => {
  const now = new Date()
  return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth()
}

const monthIn = computed(() =>
  sumBy(
    incomes.value.filter((i) => inMonth(new Date(i.date))),
    (i) => i.amount,
  ),
)
const monthOut = computed(() =>
  sumBy(
    spendings.value.filter((s) => inMonth(new Date(s.date))),
    (s) => s.amount,
  ),
)
const monthNet = computed(() => monthIn.value - monthOut.value)
const netWorth = computed(() =>
  sumBy(incomes.value, (i) => i.amount) - sumBy(spendings.value, (s) => s.amount),
)
const netWorth30dDelta = computed(() => {
  const cutoff = Date.now() - 30 * 24 * 3600 * 1000
  const inNet = sumBy(
    incomes.value.filter((i) => new Date(i.date).getTime() >= cutoff),
    (i) => i.amount,
  )
  const outNet = sumBy(
    spendings.value.filter((s) => new Date(s.date).getTime() >= cutoff),
    (s) => s.amount,
  )
  return inNet - outNet
})

const spark = computed(() => {
  const days = lastNDays(30)
  const byDay = groupBy(spendings.value, (s) => formatDay(new Date(s.date)))
  return days.map((d) => sumBy(byDay.get(d) ?? [], (s) => s.amount))
})

interface CatRow {
  cat: IconName
  label: string
  amt: number
  pct: number
}

const CAT_ICON_MAP: Record<string, IconName> = {
  rent: 'bed',
  stay: 'bed',
  hotel: 'bed',
  groceries: 'cart',
  food: 'fork',
  dining: 'fork',
  'eating out': 'fork',
  travel: 'plane',
  flights: 'plane',
  transport: 'car',
  car: 'car',
  ride: 'car',
  activity: 'ticket',
  coffee: 'coffee',
  cafes: 'coffee',
  home: 'home2',
}

function iconFor(category?: string): IconName {
  const k = (category ?? '').toLowerCase().trim()
  return CAT_ICON_MAP[k] ?? 'wallet'
}

const topCats = computed<CatRow[]>(() => {
  const bucket = new Map<string, number>()
  for (const s of spendings.value) {
    const key = (s.category || 'Other').toLowerCase()
    bucket.set(key, (bucket.get(key) ?? 0) + s.amount)
  }
  const total = Array.from(bucket.values()).reduce((a, b) => a + b, 0) || 1
  const rows = Array.from(bucket.entries())
    .map(([k, amt]) => ({
      cat: iconFor(k),
      label: k.replace(/\b\w/g, (c) => c.toUpperCase()),
      amt,
      pct: amt / total,
    }))
    .sort((a, b) => b.amt - a.amt)
    .slice(0, 5)
  return rows
})

interface GroupRollup {
  id: string
  name: string
  members: number
  you: number
  tint: string
}

const TINTS = ['#D64933', '#2F5F4F', '#B8915A', '#4A5577', '#8B4A55']

const sharedGroups = computed<GroupRollup[]>(() => {
  const myId = auth.userIdFromToken()
  return groups.value.slice(0, 3).map((g, i) => {
    const gid = String(g._id ?? g.id)
    const sp = spendings.value.filter((s) => String(s.group_id) === gid)
    const total = sumBy(sp, (s) => s.amount)
    const mCount = g.member_ids?.length ?? 1
    const per = mCount > 0 ? total / mCount : 0
    const paid = myId
      ? sumBy(
          sp.filter((s) => String(s.user_id) === myId),
          (s) => s.amount,
        )
      : 0
    return {
      id: gid,
      name: g.name,
      members: mCount,
      you: paid - per,
      tint: TINTS[i % TINTS.length],
    }
  })
})

interface ActivityRow {
  id: string
  cat: IconName
  title: string
  when: string
  amt: number
  group: string | null
}

function whenLabel(dateStr: string): string {
  const d = new Date(dateStr)
  const now = new Date()
  const mid = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate())
  const diff = (mid(now).getTime() - mid(d).getTime()) / (24 * 3600 * 1000)
  if (diff === 0) return 'TODAY · ' + d.toTimeString().slice(0, 5)
  if (diff === 1) return 'YESTERDAY'
  if (diff < 7) return d.toLocaleDateString('en-US', { weekday: 'short' }).toUpperCase()
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }).toUpperCase()
}

const recent = computed<ActivityRow[]>(() => {
  const copy = spendings.value.slice()
  copy.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime())
  return copy.slice(0, 5).map((s) => {
    const g = groups.value.find(
      (x) => String(x._id ?? x.id) === String(s.group_id),
    )
    return {
      id: String(s._id ?? s.id),
      cat: iconFor(s.category),
      title: s.description || s.category || 'Spending',
      when: whenLabel(s.date),
      amt: s.amount,
      group: g?.name ?? null,
    }
  })
})

function goSpending() {
  router.push('/spendings')
}

function goIncome() {
  router.push('/incomes')
}

function goGroup(id: string) {
  router.push(`/groups/${id}`)
}

function goGroups() {
  router.push('/groups')
}

function goLedger(side: 'borrowed' | 'lent') {
  router.push({ path: '/ledger', query: { side } })
}

// Off-book aggregates — what the screenshot's strip needs. Outstanding is the
// sum that's still owed (amount - resolved, clipped at 0); `actual` is the
// total principal across all records on that side; `open` counts pending
// FinanceRequests so we can flash the red strip when amendments need review.
interface SideTotals {
  outstanding: number
  actual: number
  count: number
  open: number
  currency: string
}

function totalsFor(list: Credit[]): SideTotals {
  let outstanding = 0
  let actual = 0
  let open = 0
  for (const c of list) {
    const amt = c.amount || 0
    const res = c.resolved_amount || 0
    actual += amt
    outstanding += Math.max(0, amt - res)
    for (const r of c.finance_requests ?? []) {
      if (r.status === 'pending') open += 1
    }
  }
  return {
    outstanding,
    actual,
    count: list.length,
    open,
    currency: list[0]?.currency || currency.value,
  }
}

const borrowedTotals = computed(() => totalsFor(borrowed.value))
const lentTotals = computed(() => totalsFor(lent.value))
const hasOffBook = computed(() => borrowed.value.length > 0 || lent.value.length > 0)

// Pending requests where I'm the *reviewer* (other side opened the request).
// Drives the bell badge — requests I opened myself live on the credit detail
// screen as "awaiting them" and don't belong in the inbox count.
const pendingForMe = computed(() => {
  const me = auth.userIdFromToken() ?? ''
  let n = 0
  for (const c of [...borrowed.value, ...lent.value]) {
    for (const r of c.finance_requests ?? []) {
      if (r.status === 'pending' && r.requested_by !== me) n += 1
    }
  }
  return n
})

onMounted(async () => {
  try {
    const uid = auth.userIdFromToken()
    const [g, s, i, b, l] = await Promise.all([
      groupApi.list(),
      uid ? spendingApi.query({ user_id: uid }) : spendingApi.query(),
      incomeApi.list().catch(() => [] as Income[]),
      borrowingApi.list().catch(() => [] as Credit[]),
      lendingApi.list().catch(() => [] as Credit[]),
    ])
    groups.value = g ?? []
    spendings.value = s ?? []
    incomes.value = i ?? []
    borrowed.value = b ?? []
    lent.value = l ?? []
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
})

// When the notifications sheet resolves a request, the returned Credit replaces
// our local copy so the strip's open-count and the bell badge tick down without
// a full reload.
function onRequestResolved(updated: Credit, side: 'borrowing' | 'lending') {
  const list = side === 'borrowing' ? borrowed : lent
  const id = String(updated._id ?? updated.id)
  list.value = list.value.map((c) => (String(c._id ?? c.id) === id ? updated : c))
}
</script>

<template>
  <section class="home">
    <p v-if="loading" class="muted">{{ t('common.loading') }}</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <template v-else>
      <!-- Hero -->
      <div class="hero-block">
        <div class="hero-top">
          <p class="eyebrow" style="margin: 0">{{ t('home.good_afternoon') }}</p>
          <button
            class="bell-btn"
            :aria-label="t('notifications.title')"
            @click="showNotifications = true"
          >
            <Icon name="bell" :size="18" />
            <span v-if="pendingForMe > 0" class="bell-badge">{{ pendingForMe }}</span>
          </button>
        </div>
        <h1 class="hero">
          {{ t('home.net_for_month', { month: monthLabel.split(' ')[0].slice(0, 3) }) }}<br />
          <em class="net-pos">{{ t('home.is_amount', { amount: signedWhole(monthNet, currency) }) }}</em>
        </h1>
      </div>

      <!-- Net worth card -->
      <div class="card-ink net-card">
        <div class="row spread">
          <span class="nw-label">{{ t('home.net_worth_30d') }}</span>
          <span class="nw-change">▲ {{ signedWhole(netWorth30dDelta, currency) }}</span>
        </div>
        <div class="money money-big inverse">
          {{ moneyWhole(netWorth, currency) }}
        </div>
        <div class="spark">
          <Sparkline :data="spark" :width="314" :height="52" stroke="#F5F1E8" />
        </div>
        <div class="nw-stats">
          <div>
            <div class="nw-stat-label">{{ t('home.in') }}</div>
            <div class="nw-stat-value nw-in">{{ moneyWhole(monthIn, currency) }}</div>
          </div>
          <div>
            <div class="nw-stat-label">{{ t('home.out') }}</div>
            <div class="nw-stat-value nw-out">{{ moneyWhole(monthOut, currency) }}</div>
          </div>
          <div>
            <div class="nw-stat-label">{{ t('home.net') }}</div>
            <div class="nw-stat-value nw-net">{{ signedWhole(monthNet, currency) }}</div>
          </div>
        </div>
      </div>

      <!-- Quick actions -->
      <div class="quick-row">
        <button class="btn btn-accent btn-lg" @click="goSpending">
          <Icon name="plus" :size="18" /> {{ t('home.spending') }}
        </button>
        <button class="btn btn-secondary btn-lg" @click="goIncome">
          <Icon name="arrowDown" :size="18" /> {{ t('home.income') }}
        </button>
      </div>

      <!-- Off-book ledger strip -->
      <div v-if="hasOffBook" class="section-block off-book">
        <div class="row spread section-head">
          <span class="eyebrow">{{ t('home.off_book_eyebrow') }}</span>
          <button class="linklike" @click="goLedger('borrowed')">{{ t('home.open_ledger') }}</button>
        </div>
        <h3 class="serif">
          {{ t('home.borrowed_amp_lent') }} <em>{{ t('home.lent_em') }}</em>
        </h3>
        <div class="ob-grid">
          <button
            class="ob-card"
            @click="goLedger('borrowed')"
          >
            <div class="ob-head">
              <span class="dot" style="background: var(--hot)"></span>
              <span class="ob-label">{{ t('home.you_owe_label') }}</span>
            </div>
            <div class="ob-money" style="color: var(--hot)">
              {{ moneyWhole(borrowedTotals.outstanding, borrowedTotals.currency) }}
            </div>
            <div class="ob-meta">
              {{ t('home.of_count_rec', {
                actual: moneyWhole(borrowedTotals.actual, borrowedTotals.currency),
                count: borrowedTotals.count,
              }) }}
            </div>
            <div class="ob-bar">
              <div
                class="ob-bar-fill"
                :style="{
                  width: (borrowedTotals.actual ? (borrowedTotals.actual - borrowedTotals.outstanding) / borrowedTotals.actual * 100 : 0) + '%',
                  background: 'var(--hot)',
                }"
              ></div>
            </div>
            <div v-if="borrowedTotals.open > 0" class="ob-open">
              <span class="dot" style="background: var(--hot)"></span>
              {{ t(borrowedTotals.open === 1 ? 'home.open_amendment_one' : 'home.open_amendments_other', { n: borrowedTotals.open }) }}
            </div>
          </button>
          <button
            class="ob-card"
            @click="goLedger('lent')"
          >
            <div class="ob-head">
              <span class="dot" style="background: var(--moss)"></span>
              <span class="ob-label">{{ t('home.owed_to_you_label') }}</span>
            </div>
            <div class="ob-money" style="color: var(--moss)">
              {{ moneyWhole(lentTotals.outstanding, lentTotals.currency) }}
            </div>
            <div class="ob-meta">
              {{ t('home.of_count_rec', {
                actual: moneyWhole(lentTotals.actual, lentTotals.currency),
                count: lentTotals.count,
              }) }}
            </div>
            <div class="ob-bar">
              <div
                class="ob-bar-fill"
                :style="{
                  width: (lentTotals.actual ? (lentTotals.actual - lentTotals.outstanding) / lentTotals.actual * 100 : 0) + '%',
                  background: 'var(--moss)',
                }"
              ></div>
            </div>
            <div v-if="lentTotals.open > 0" class="ob-open">
              <span class="dot" style="background: var(--hot)"></span>
              {{ t(lentTotals.open === 1 ? 'home.open_amendment_one' : 'home.open_amendments_other', { n: lentTotals.open }) }}
            </div>
          </button>
        </div>
        <div class="ob-footer">{{ t('home.off_book_footer') }}</div>
      </div>

      <!-- Top categories -->
      <div v-if="topCats.length" class="section-block">
        <div class="row spread section-head">
          <span class="eyebrow">{{ t('home.where_it_went') }} · {{ monthLabel.split(' ')[0].slice(0, 3).toUpperCase() }}</span>
          <span class="eyebrow">{{ t('home.view_all') }}</span>
        </div>
        <h3 class="serif">{{ t('home.top_categories') }}</h3>
        <div>
          <div
            v-for="(c, i) in topCats"
            :key="c.label"
            class="cat-row"
            :class="{ last: i === topCats.length - 1 }"
          >
            <div class="cat-glyph">
              <Icon :name="c.cat" :size="16" />
            </div>
            <div>
              <div class="row spread cat-label-row">
                <span class="cat-label">{{ c.label }}</span>
                <span class="cat-trend">— flat</span>
              </div>
              <div class="cat-rail">
                <div class="cat-fill" :style="{ width: c.pct * 100 + '%' }"></div>
              </div>
            </div>
            <div class="cat-amt">{{ moneyWhole(c.amt, currency) }}</div>
          </div>
        </div>
      </div>

      <!-- Shared with others -->
      <div v-if="sharedGroups.length" class="section-block">
        <div class="row spread section-head">
          <span class="eyebrow">{{ t('home.shared_groups', { n: sharedGroups.length }) }}</span>
          <button class="linklike" @click="goGroups">{{ t('home.view_all') }}</button>
        </div>
        <h3 class="serif">{{ t('home.split_with_others') }}</h3>
        <div class="shared-list">
          <button
            v-for="g in sharedGroups"
            :key="g.id"
            class="shared-row"
            @click="goGroup(g.id)"
          >
            <div class="shared-tint" :style="{ background: g.tint }"></div>
            <div>
              <div class="shared-name">{{ g.name }}</div>
              <div class="shared-sub">
                {{ g.members }} {{ t('home.members') }} ·
                {{ g.you >= 0 ? t('home.you_are_owed') : t('home.you_owe') }}
              </div>
            </div>
            <div class="shared-bal" :style="{ color: g.you >= 0 ? 'var(--moss)' : 'var(--hot)' }">
              {{ signed(g.you, currency) }}
            </div>
          </button>
        </div>
      </div>

      <!-- Recent activity -->
      <div v-if="recent.length" class="section-block">
        <span class="eyebrow">{{ t('home.activity') }}</span>
        <h3 class="serif">{{ t('home.recent_movements') }}</h3>
        <div>
          <div
            v-for="(e, i) in recent"
            :key="e.id"
            class="row-entry"
            :class="{ last: i === recent.length - 1 }"
          >
            <div class="glyph"><Icon :name="e.cat" :size="18" /></div>
            <div style="min-width: 0">
              <div class="title">{{ e.title }}</div>
              <div class="sub">
                {{ e.when }}<span v-if="e.group" class="group-tag"> · {{ e.group.toUpperCase() }}</span>
              </div>
            </div>
            <div class="figure">{{ money(e.amt, currency) }}</div>
          </div>
        </div>
      </div>

    </template>

    <NotificationsSheet
      :open="showNotifications"
      :borrowed="borrowed"
      :lent="lent"
      @close="showNotifications = false"
      @request-resolved="onRequestResolved"
    />
  </section>
</template>

<style scoped>
.home {
  padding-top: 0;
}

.hero-block {
  padding: 0 0 2px;
}

.hero-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.bell-btn {
  position: relative;
  background: none;
  border: none;
  padding: 4px;
  cursor: pointer;
  color: var(--ink-soft);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.bell-btn:hover {
  color: var(--ink);
}
.bell-badge {
  position: absolute;
  top: 0;
  right: 0;
  min-width: 14px;
  height: 14px;
  padding: 0 3px;
  border-radius: 7px;
  background: var(--hot);
  color: var(--paper);
  font-family: var(--mono);
  font-size: 9px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 0 0 2px var(--paper);
}

.hero .net-pos {
  color: var(--moss);
  font-style: italic;
}

/* Off-book strip */
.off-book .section-head {
  margin-bottom: 4px;
}
.ob-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-top: 10px;
}
.ob-card {
  text-align: left;
  background: var(--paper);
  border: 1px solid var(--line);
  border-radius: var(--r-lg);
  padding: 14px 14px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 6px;
  transition: border-color 0.15s;
}
.ob-card:hover {
  border-color: var(--ink);
}
.ob-head {
  display: flex;
  align-items: center;
  gap: 6px;
}
.ob-head .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}
.ob-label {
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.1em;
  color: var(--ink-mute);
}
.ob-money {
  font-family: var(--serif);
  font-size: 28px;
  line-height: 1;
  margin-top: 4px;
}
.ob-meta {
  font-family: var(--mono);
  font-size: 9px;
  color: var(--ink-mute);
  letter-spacing: 0.06em;
  margin-top: 2px;
}
.ob-bar {
  height: 3px;
  background: var(--line);
  border-radius: 2px;
  overflow: hidden;
  margin-top: 6px;
}
.ob-bar-fill {
  height: 100%;
  opacity: 0.5;
}
.ob-open {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 6px;
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.08em;
  color: var(--hot);
}
.ob-open .dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
}
.ob-footer {
  margin-top: 10px;
  font-family: var(--mono);
  font-size: 10px;
  color: var(--ink-ghost);
  letter-spacing: 0.06em;
}

/* Net worth card */
.net-card {
  margin-top: 22px;
  padding: clamp(16px, 4.5vw, 20px) clamp(16px, 5vw, 22px) clamp(14px, 4vw, 18px);
  overflow: hidden;
}

.net-card .nw-label {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.12em;
  color: rgba(245, 241, 232, 0.6);
}

.net-card .nw-change {
  font-family: var(--mono);
  font-size: 10px;
  color: #4ade80;
  white-space: nowrap;
}

.net-card .money-big {
  /* Compact: 28px on iPhone SE → 40px on tablet+ */
  font-size: clamp(28px, 8.5vw, 40px);
  line-height: 1;
  letter-spacing: -0.02em;
  margin-top: 6px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: clip;
  display: flex;
  align-items: baseline;
  gap: 0;
  min-width: 0;
}

.net-card .money-big .int {
  min-width: 0;
}

.net-card .money-big .cur {
  font-size: clamp(12px, 3.5vw, 16px);
  vertical-align: top;
  margin-right: 2px;
  color: rgba(245, 241, 232, 0.55);
  line-height: 1;
  align-self: flex-start;
}

.net-card .money-big .decimals {
  font-size: clamp(14px, 4.5vw, 20px);
  color: rgba(245, 241, 232, 0.6);
}

.net-card .spark {
  margin: 10px -4px 0;
}

.nw-stats {
  margin-top: 10px;
  padding-top: 12px;
  border-top: 1px solid rgba(245, 241, 232, 0.1);
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: clamp(8px, 3vw, 14px);
}

.nw-stats > div {
  min-width: 0;
}

.nw-stat-label {
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.1em;
  color: rgba(245, 241, 232, 0.5);
}

.nw-stat-value {
  font-family: var(--serif);
  font-size: clamp(13px, 3.8vw, 16px);
  margin-top: 2px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: clip;
  letter-spacing: -0.01em;
}

.nw-in {
  color: #4ade80;
}

.nw-out {
  color: #fca5a5;
}

.nw-net {
  color: var(--paper);
}

/* Quick actions */
.quick-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-top: 18px;
}

.quick-row .btn {
  width: 100%;
}

/* Sections */
.section-block {
  margin-top: 32px;
}

.section-head {
  margin-bottom: 4px;
}

/* Categories */
.cat-row {
  display: grid;
  grid-template-columns: 32px 1fr auto;
  gap: 12px;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px solid var(--line);
}

.cat-row.last {
  border-bottom: none;
}

.cat-glyph {
  width: 32px;
  height: 32px;
  border-radius: 10px;
  background: var(--paper-deep);
  border: 1px solid var(--line);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--ink);
}

.cat-label-row {
  margin-bottom: 4px;
}

.cat-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--ink);
}

.cat-trend {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--ink-mute);
}

.cat-rail {
  height: 4px;
  background: var(--line);
  border-radius: 2px;
  overflow: hidden;
}

.cat-fill {
  height: 100%;
  background: var(--ink);
}

.cat-amt {
  font-family: var(--serif);
  font-size: 18px;
  color: var(--ink);
  min-width: 70px;
  text-align: right;
}

.cat-amt .cur {
  font-size: 11px;
  color: var(--ink-mute);
  margin-right: 1px;
}

/* Shared rollup */
.split-em {
  color: var(--hot);
  font-style: italic;
}

.shared-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 4px;
}

.shared-row {
  display: grid;
  grid-template-columns: 8px 1fr auto;
  gap: 14px;
  align-items: center;
  padding: 12px 14px;
  background: var(--paper);
  border: 1px solid var(--line);
  border-radius: var(--r);
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s;
}

.shared-row:hover {
  border-color: var(--ink);
}

.shared-tint {
  width: 8px;
  height: 40px;
  border-radius: 2px;
}

.shared-name {
  font-family: var(--serif);
  font-size: 18px;
  line-height: 1.1;
  color: var(--ink);
}

.shared-sub {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  color: var(--ink-mute);
  margin-top: 3px;
}

.shared-bal {
  font-family: var(--mono);
  font-size: 12px;
}

/* Recent activity */
.group-tag {
  color: var(--hot);
}
</style>
