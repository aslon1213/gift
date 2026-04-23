<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { incomeApi } from '../api/endpoints'
import type { Income } from '../api/types'
import DonutChart from '../components/DonutChart.vue'
import BarChart from '../components/BarChart.vue'
import { formatDay, groupBy, lastNDays, sumBy } from '../utils/charts'

const incomes = ref<Income[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const submitting = ref(false)

const showForm = ref(false)
const form = ref({
  amount: 0,
  currency: 'UZS',
  source: '',
  description: '',
  date: new Date().toISOString().slice(0, 10),
})

const filterSource = ref('')

const filtered = computed(() =>
  filterSource.value
    ? incomes.value.filter((i) =>
        (i.source || '').toLowerCase().includes(filterSource.value.toLowerCase()),
      )
    : incomes.value,
)

const total = computed(() => sumBy(filtered.value, (i) => i.amount))

const sourceData = computed(() => {
  const bySrc = groupBy(filtered.value, (i) => i.source || 'Unspecified')
  return Array.from(bySrc.entries()).map(([label, items]) => ({
    label,
    value: sumBy(items, (x) => x.amount),
  }))
})

const dailyData = computed(() => {
  const days = lastNDays(30)
  const byDay = groupBy(filtered.value, (i) => formatDay(new Date(i.date)))
  return days.map((d) => ({
    label: d.slice(5),
    value: sumBy(byDay.get(d) ?? [], (x) => x.amount),
  }))
})

const currency = computed(() => incomes.value[0]?.currency ?? '')

async function load() {
  loading.value = true
  error.value = null
  try {
    incomes.value = (await incomeApi.list()) ?? []
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function create() {
  submitting.value = true
  error.value = null
  try {
    await incomeApi.create({
      amount: Number(form.value.amount),
      currency: form.value.currency,
      source: form.value.source,
      description: form.value.description,
      date: new Date(form.value.date).toISOString(),
    })
    showForm.value = false
    form.value.amount = 0
    form.value.description = ''
    form.value.source = ''
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Create failed'
  } finally {
    submitting.value = false
  }
}

async function remove(id: string) {
  if (!confirm('Delete this income?')) return
  try {
    await incomeApi.remove(id)
    incomes.value = incomes.value.filter((i) => String(i.id) !== id)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

onMounted(load)
</script>

<template>
  <section>
    <header class="row spread">
      <h1>Incomes</h1>
      <button class="ghost" @click="showForm = !showForm">
        {{ showForm ? 'Cancel' : '+ New' }}
      </button>
    </header>

    <form v-if="showForm" class="stack-form" @submit.prevent="create">
      <div class="split">
        <label class="field">
          <span>Amount</span>
          <input v-model.number="form.amount" type="number" min="0" step="0.01" required />
        </label>
        <label class="field">
          <span>Currency</span>
          <input v-model="form.currency" />
        </label>
      </div>
      <label class="field">
        <span>Source</span>
        <input v-model="form.source" placeholder="Salary, Freelance…" />
      </label>
      <label class="field">
        <span>Description</span>
        <input v-model="form.description" />
      </label>
      <label class="field">
        <span>Date</span>
        <input v-model="form.date" type="date" required />
      </label>
      <button type="submit" :disabled="submitting">
        {{ submitting ? 'Saving…' : 'Save income' }}
      </button>
    </form>

    <div class="stats">
      <div class="card">
        <div class="stat-label">Total</div>
        <div class="stat-value">{{ total.toLocaleString() }} {{ currency }}</div>
      </div>
      <div class="card">
        <div class="stat-label">Records</div>
        <div class="stat-value">{{ filtered.length }}</div>
      </div>
      <div class="card">
        <div class="stat-label">Top source</div>
        <div class="stat-value" style="font-size: 16px">
          {{ sourceData.slice().sort((a, b) => b.value - a.value)[0]?.label ?? '—' }}
        </div>
      </div>
    </div>

    <div class="chart-grid">
      <DonutChart title="By source" :data="sourceData" :currency="currency" />
      <BarChart title="Last 30 days" :data="dailyData" primary-color="#4ade80" />
    </div>

    <div class="filters">
      <label class="field">
        <span>Source</span>
        <input v-model="filterSource" placeholder="Filter by source" />
      </label>
    </div>

    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <ul v-else-if="filtered.length" class="list">
      <li v-for="i in filtered" :key="i.id">
        <div style="min-width: 0">
          <strong>{{ i.source || 'Unspecified' }}</strong>
          <div class="muted small">
            {{ i.description || 'no note' }} · {{ new Date(i.date).toLocaleDateString() }}
          </div>
        </div>
        <div class="row">
          <span>{{ i.amount.toLocaleString() }} {{ i.currency }}</span>
          <button class="danger" @click="remove(String(i.id))">Delete</button>
        </div>
      </li>
    </ul>
    <div v-else class="empty">No incomes yet.</div>
  </section>
</template>
