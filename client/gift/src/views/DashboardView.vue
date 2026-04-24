<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { groupApi, incomeApi, spendingApi, userApi } from '../api/endpoints'
import type { Group, Income, Spending, User } from '../api/types'
import { auth } from '../stores/auth'
import Icon from '../components/Icon.vue'
import Avatar from '../components/Avatar.vue'
import Sparkline from '../components/Sparkline.vue'
import type { IconName } from '../components/icons'
import { colorForId, money, signed } from '../utils/format'
import { formatDay, groupBy, lastNDays, sumBy } from '../utils/charts'

const groups = ref<Group[]>([])
const spendings = ref<Spending[]>([])
const incomes = ref<Income[]>([])
const me = ref<User | null>(null)
const memberCache = ref<Record<string, User>>({})
const loading = ref(true)
const error = ref<string | null>(null)

const activeGroup = computed<Group | null>(() => groups.value[0] ?? null)

const currency = computed(
  () => spendings.value[0]?.currency ?? incomes.value[0]?.currency ?? '',
)

const groupSpendings = computed(() => {
  const g = activeGroup.value
  if (!g) return spendings.value
  const gid = String(g._id ?? g.id)
  return spendings.value.filter((s) => String(s.group_id) === gid)
})

const totalSpent = computed(() => sumBy(groupSpendings.value, (s) => s.amount))

const todaySpent = computed(() => {
  const now = new Date().toDateString()
  return sumBy(
    groupSpendings.value.filter((s) => new Date(s.date).toDateString() === now),
    (s) => s.amount,
  )
})

const txToday = computed(
  () =>
    groupSpendings.value.filter(
      (s) => new Date(s.date).toDateString() === new Date().toDateString(),
    ).length,
)

const spark = computed(() => {
  const days = lastNDays(10)
  const byDay = groupBy(groupSpendings.value, (s) =>
    formatDay(new Date(s.date)),
  )
  return days.map((d) => sumBy(byDay.get(d) ?? [], (s) => s.amount))
})

const recent = computed(() => {
  const copy = groupSpendings.value.slice()
  copy.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime())
  return copy.slice(0, 6)
})

const members = computed(() => {
  if (!activeGroup.value) return []
  const ids = Array.from(
    new Set([
      activeGroup.value.owner_id,
      ...(activeGroup.value.member_ids ?? []),
    ]),
  )
  return ids.map((id) => memberCache.value[id]).filter(Boolean) as User[]
})

interface Balance {
  user: User
  net: number
}

const balances = computed<Balance[]>(() => {
  if (!activeGroup.value || !members.value.length) return []
  const total = totalSpent.value
  const per = members.value.length ? total / members.value.length : 0
  return members.value.map((u) => {
    const uid = String(u._id ?? u.id)
    const paid = sumBy(
      groupSpendings.value.filter((s) => String(s.user_id) === uid),
      (s) => s.amount,
    )
    return { user: u, net: paid - per }
  })
})

const balanceMax = computed(() =>
  Math.max(1, ...balances.value.map((b) => Math.abs(b.net))),
)

function iconFor(category?: string): IconName {
  const map: Record<string, IconName> = {
    food: 'fork',
    dining: 'fork',
    travel: 'plane',
    flights: 'plane',
    stay: 'bed',
    hotel: 'bed',
    transport: 'car',
    ride: 'car',
    activity: 'ticket',
    groceries: 'cart',
    coffee: 'coffee',
    home: 'home2',
  }
  const k = (category ?? '').toLowerCase().trim()
  return map[k] ?? 'wallet'
}

function when(dateStr: string): string {
  const d = new Date(dateStr)
  const now = new Date()
  const mid = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate())
  const diff = (mid(now).getTime() - mid(d).getTime()) / (24 * 3600 * 1000)
  if (diff === 0)
    return 'TODAY · ' + d.toTimeString().slice(0, 5)
  if (diff === 1) return 'YESTERDAY'
  if (diff < 7) return d.toLocaleDateString(undefined, { weekday: 'short' }).toUpperCase()
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }).toUpperCase()
}

function memberName(uid: string): string {
  if (uid === auth.userIdFromToken()) return 'You'
  return memberCache.value[uid]?.name || 'Someone'
}

const greetingLine = computed(() => {
  const g = activeGroup.value
  if (!g) return { first: 'Your', second: 'groups.' }
  const parts = g.name.trim().split(/\s+/)
  if (parts.length === 1) return { first: parts[0], second: '' }
  return { first: parts[0], second: parts.slice(1).join(' ') }
})

onMounted(async () => {
  try {
    const uid = auth.userIdFromToken()
    const [g, s, i, u] = await Promise.all([
      groupApi.list(),
      uid ? spendingApi.query({ user_id: uid }) : spendingApi.query(),
      incomeApi.list().catch(() => [] as Income[]),
      uid ? userApi.getById(uid).catch(() => null) : Promise.resolve(null),
    ])
    groups.value = g ?? []
    spendings.value = s ?? []
    incomes.value = i ?? []
    me.value = u

    // Preload members of the active group
    if (groups.value[0]) {
      const g0 = groups.value[0]
      const ids = Array.from(
        new Set([g0.owner_id, ...(g0.member_ids ?? [])]),
      )
      const results = await Promise.allSettled(
        ids.map((id) => userApi.getById(id)),
      )
      const next: Record<string, User> = { ...memberCache.value }
      results.forEach((r, idx) => {
        if (r.status === 'fulfilled' && r.value) next[ids[idx]] = r.value
      })
      memberCache.value = next
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <section>
    <header class="row spread top-actions">
      <router-link to="/groups" class="back-link">
        <Icon name="people" :size="14" />
        <span>ALL GROUPS</span>
      </router-link>
      <div class="row" style="gap: 2px">
        <button class="icon-btn" aria-label="Search">
          <Icon name="search" :size="18" />
        </button>
        <button class="icon-btn" aria-label="Notifications" style="position: relative">
          <Icon name="bell" :size="18" />
          <span class="pip-dot"></span>
        </button>
      </div>
    </header>

    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <template v-else>
      <div v-if="!activeGroup" class="empty-state">
        <p class="eyebrow">NO GROUP YET</p>
        <h1 class="hero">Start your <em>first</em> split.</h1>
        <router-link to="/groups" class="btn btn-accent btn-lg btn-block">
          <Icon name="plus" :size="18" /> Create a group
        </router-link>
      </div>

      <template v-else>
        <!-- Hero -->
        <div class="hero-block">
          <p class="eyebrow">GROUP · ACTIVE</p>
          <div class="hero-row">
            <h1 class="hero">
              {{ greetingLine.first }}<br />
              <em>{{ greetingLine.second || 'trip.' }}</em>
            </h1>
            <div class="avatar-stack" v-if="members.length">
              <Avatar
                v-for="m in members.slice(0, 4)"
                :key="String(m._id ?? m.id)"
                :name="m.name"
                :color="colorForId(String(m._id ?? m.id))"
                :size="30"
                ring
              />
            </div>
          </div>
          <div class="subtitle">
            {{ activeGroup.member_ids?.length ?? 0 }} members · {{ groupSpendings.length }} spendings
          </div>
        </div>

        <!-- Total card with sparkline -->
        <div class="card-ink total-card">
          <div class="mono-label">TOTAL SPENT · TRIP</div>
          <div class="money-big">
            <span class="cur">{{ currency || '$' }}</span>
            {{ Math.floor(totalSpent).toLocaleString()
            }}<span class="decimals">.{{
              (totalSpent % 1).toFixed(2).slice(2)
            }}</span>
          </div>
          <div class="row spread meta">
            <span><span class="dot" />{{ money(todaySpent, currency || '$') }} today · {{ txToday }} txns</span>
            <span>10 DAYS</span>
          </div>
          <div class="spark">
            <Sparkline :data="spark" :width="314" :height="48" stroke="#F5F1E8" />
          </div>
        </div>

        <!-- Quick actions -->
        <div class="quick-row">
          <router-link to="/spendings" class="btn btn-accent btn-lg">
            <Icon name="plus" :size="18" /> Spending
          </router-link>
          <router-link to="/incomes" class="btn btn-secondary btn-lg">
            <Icon name="arrowDown" :size="18" /> Income
          </router-link>
        </div>

        <!-- Balances -->
        <div v-if="balances.length" class="section-block">
          <div class="row spread section-head">
            <span class="eyebrow">BALANCES</span>
            <span class="eyebrow">SETTLE UP →</span>
          </div>
          <h3 class="serif">Who is <em>up.</em></h3>
          <div class="balances">
            <div
              v-for="b in balances"
              :key="String(b.user._id ?? b.user.id)"
              class="balance-row"
            >
              <Avatar
                :name="b.user.name"
                :color="colorForId(String(b.user._id ?? b.user.id))"
                :size="28"
              />
              <div class="balance-track">
                <div class="mid"></div>
                <div
                  class="fill"
                  :style="{
                    left:
                      b.net >= 0
                        ? '50%'
                        : 50 - (Math.abs(b.net) / balanceMax) * 50 + '%',
                    width:
                      (Math.abs(b.net) / balanceMax) * 50 + '%',
                    background: b.net >= 0 ? 'var(--moss)' : 'var(--hot)',
                  }"
                ></div>
              </div>
              <div
                class="figure"
                :style="{ color: b.net >= 0 ? 'var(--moss)' : 'var(--hot)' }"
              >
                {{ signed(b.net, currency || '$') }}
              </div>
            </div>
          </div>
        </div>

        <!-- Recent -->
        <div class="section-block">
          <div class="row spread section-head">
            <span class="eyebrow">ACTIVITY · LAST</span>
            <router-link to="/spendings" class="eyebrow" style="color: var(--ink-mute); text-decoration: none">VIEW ALL →</router-link>
          </div>
          <h3 class="serif">Recent <em>spendings.</em></h3>

          <div v-if="!recent.length" class="empty">No spendings yet.</div>
          <div v-else>
            <div
              v-for="(s, i) in recent"
              :key="String(s._id ?? s.id)"
              class="row-entry"
              :class="{ last: i === recent.length - 1 }"
            >
              <div class="glyph"><Icon :name="iconFor(s.category)" :size="18" /></div>
              <div style="min-width: 0">
                <div class="title">{{ s.description || s.category || 'Spending' }}</div>
                <div class="sub">
                  {{ when(s.date) }} · {{ memberName(s.user_id) }} paid
                </div>
              </div>
              <div class="figure">
                <span class="cur">{{ s.currency || '$' }}</span>
                {{ s.amount.toFixed(2) }}
              </div>
            </div>
          </div>
        </div>

      </template>
    </template>
  </section>
</template>

<style scoped>
.top-actions {
  margin-bottom: 18px;
}
.back-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  color: var(--ink-soft);
  text-decoration: none;
}
.back-link:hover {
  color: var(--ink);
  text-decoration: none;
}
.pip-dot {
  position: absolute;
  top: 10px;
  right: 10px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--hot);
}

.hero-block {
  padding: 8px 0 4px;
}
.hero-row {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
}
.subtitle {
  font-family: var(--sans);
  font-size: 13px;
  color: var(--ink-soft);
  margin-top: 10px;
}

.total-card {
  margin-top: 24px;
  padding: 22px 22px 18px;
}
.total-card .mono-label {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: rgba(245, 241, 232, 0.6);
}
.total-card .money-big {
  font-family: var(--serif);
  font-weight: 400;
  font-size: 52px;
  line-height: 1;
  letter-spacing: -0.02em;
  margin-top: 8px;
}
.total-card .money-big .cur {
  font-size: 22px;
  vertical-align: top;
  margin-right: 2px;
  color: rgba(245, 241, 232, 0.55);
}
.total-card .money-big .decimals {
  font-size: 28px;
  color: rgba(245, 241, 232, 0.6);
}
.total-card .meta {
  margin-top: 6px;
  font-family: var(--mono);
  font-size: 11px;
  color: rgba(245, 241, 232, 0.6);
}
.total-card .meta .dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--hot);
  margin-right: 6px;
  vertical-align: middle;
}
.total-card .spark {
  margin: 12px -4px 0;
}

.quick-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-top: 18px;
}
.quick-row .btn {
  width: 100%;
  text-decoration: none;
}

.section-block {
  margin-top: 30px;
}
.section-head {
  margin-bottom: 4px;
}

.empty-state {
  padding: 40px 0;
  text-align: left;
}
.empty-state .btn {
  margin-top: 20px;
  width: 100%;
  text-decoration: none;
}

.balances {
  margin-top: 4px;
}
</style>
