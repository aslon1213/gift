<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { groupApi, incomeApi, spendingApi } from '../api/endpoints'
import type { Group, Income, Spending } from '../api/types'
import { auth } from '../stores/auth'
import Icon from '../components/Icon.vue'
import Sparkline from '../components/Sparkline.vue'
import type { IconName } from '../components/icons'
import { money, moneyWhole, signed, signedWhole } from '../utils/format'
import { lastNDays, formatDay, groupBy, sumBy } from '../utils/charts'
import { userStore } from '../stores/user'
import { t } from '../i18n'

const router = useRouter()

const groups = ref<Group[]>([])
const spendings = ref<Spending[]>([])
const incomes = ref<Income[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

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

onMounted(async () => {
  try {
    const uid = auth.userIdFromToken()
    const [g, s, i] = await Promise.all([
      groupApi.list(),
      uid ? spendingApi.query({ user_id: uid }) : spendingApi.query(),
      incomeApi.list().catch(() => [] as Income[]),
    ])
    groups.value = g ?? []
    spendings.value = s ?? []
    incomes.value = i ?? []
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <section class="home">
    <p v-if="loading" class="muted">{{ t('common.loading') }}</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <template v-else>
      <!-- Hero -->
      <div class="hero-block">
        <p class="eyebrow">{{ t('home.good_afternoon') }}</p>
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
  </section>
</template>

<style scoped>
.home {
  padding-top: 0;
}

.hero-block {
  padding: 0 0 2px;
}

.hero .net-pos {
  color: var(--moss);
  font-style: italic;
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
