<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { groupApi, incomeApi, spendingApi, userApi } from '../api/endpoints'
import type { Group, Income, Spending, User } from '../api/types'
import { auth } from '../stores/auth'
import DonutChart from '../components/DonutChart.vue'
import BarChart from '../components/BarChart.vue'
import { formatDay, groupBy, lastNDays, sumBy } from '../utils/charts'

const groups = ref<Group[]>([])
const spendings = ref<Spending[]>([])
const incomes = ref<Income[]>([])
const me = ref<User | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const currency = computed(() => spendings.value[0]?.currency ?? incomes.value[0]?.currency ?? '')

const totalSpent = computed(() => sumBy(spendings.value, (s) => s.amount))
const totalIncome = computed(() => sumBy(incomes.value, (i) => i.amount))
const net = computed(() => totalIncome.value - totalSpent.value)

const thisMonthSpent = computed(() => {
  const now = new Date()
  return sumBy(
    spendings.value.filter((s) => {
      const d = new Date(s.date)
      return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth()
    }),
    (s) => s.amount,
  )
})

const thisMonthIncome = computed(() => {
  const now = new Date()
  return sumBy(
    incomes.value.filter((i) => {
      const d = new Date(i.date)
      return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth()
    }),
    (i) => i.amount,
  )
})

const categoryData = computed(() => {
  const byCat = groupBy(spendings.value, (s) => s.category || 'Uncategorized')
  return Array.from(byCat.entries()).map(([label, items]) => ({
    label,
    value: sumBy(items, (i) => i.amount),
  }))
})

const flowData = computed(() => {
  const days = lastNDays(30)
  const incByDay = groupBy(incomes.value, (i) => formatDay(new Date(i.date)))
  const spByDay = groupBy(spendings.value, (s) => formatDay(new Date(s.date)))
  return days.map((d) => ({
    label: d.slice(5),
    value: sumBy(incByDay.get(d) ?? [], (x) => x.amount),
    secondary: sumBy(spByDay.get(d) ?? [], (x) => x.amount),
  }))
})

const greeting = computed(() => {
  const name = me.value?.name
  if (!name) return 'Dashboard'
  const hour = new Date().getHours()
  const part = hour < 12 ? 'Good morning' : hour < 18 ? 'Good afternoon' : 'Good evening'
  return `${part}, ${name.split(' ')[0]}`
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
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <section>
    <h1>{{ greeting }}</h1>
    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <template v-else>
      <div class="stats">
        <div class="card">
          <div class="stat-label">Balance</div>
          <div class="stat-value">{{ (me?.balance ?? 0).toLocaleString() }} {{ currency }}</div>
        </div>
        <div class="card">
          <div class="stat-label">Net</div>
          <div class="stat-value" :class="net >= 0 ? 'income' : 'spent'">
            {{ net.toLocaleString() }}
          </div>
        </div>
        <div class="card">
          <div class="stat-label">Income this month</div>
          <div class="stat-value income">{{ thisMonthIncome.toLocaleString() }}</div>
        </div>
        <div class="card">
          <div class="stat-label">Spent this month</div>
          <div class="stat-value spent">{{ thisMonthSpent.toLocaleString() }}</div>
        </div>
      </div>

      <div class="chart-grid">
        <DonutChart title="By category" :data="categoryData" :currency="currency" />
        <BarChart
          title="Income vs spending"
          :data="flowData"
          primary-color="#4ade80"
          secondary-color="#f87171"
        />
      </div>

      <div class="legend">
        <span><span class="dot" style="background:#4ade80"></span>Income</span>
        <span><span class="dot" style="background:#f87171"></span>Spending</span>
      </div>

      <h2>Recent spendings</h2>
      <ul v-if="spendings.length" class="list">
        <li v-for="s in spendings.slice(0, 5)" :key="s._id ?? s.id">
          <div style="min-width: 0">
            <strong>{{ s.category || 'Uncategorized' }}</strong>
            <div class="muted small">{{ s.description || 'no note' }}</div>
          </div>
          <div>{{ s.amount.toLocaleString() }} {{ s.currency }}</div>
        </li>
      </ul>
      <p v-else class="muted">
        No spendings yet. <router-link to="/spendings">Add one</router-link>.
      </p>

      <h2>Recent incomes</h2>
      <ul v-if="incomes.length" class="list">
        <li v-for="i in incomes.slice(0, 5)" :key="i.id">
          <div style="min-width: 0">
            <strong>{{ i.source || 'Unspecified' }}</strong>
            <div class="muted small">{{ i.description || 'no note' }}</div>
          </div>
          <div>{{ i.amount.toLocaleString() }} {{ i.currency }}</div>
        </li>
      </ul>
      <p v-else class="muted">
        No incomes yet. <router-link to="/incomes">Add one</router-link>.
      </p>

      <h2>Groups ({{ groups.length }})</h2>
      <p v-if="!groups.length" class="muted">
        You aren't in any groups yet. <router-link to="/groups">Create one</router-link>.
      </p>
      <ul v-else class="list">
        <li v-for="g in groups" :key="g._id ?? g.id">
          <router-link :to="`/groups/${g._id ?? g.id}`">
            <strong>{{ g.name }}</strong>
          </router-link>
          <span class="muted">{{ g.member_ids?.length ?? 0 }} members</span>
        </li>
      </ul>
    </template>
  </section>
</template>
