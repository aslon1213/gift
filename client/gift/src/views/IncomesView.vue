<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { incomeApi } from '../api/endpoints'
import type { Income } from '../api/types'
import { toast } from '../stores/toast'
import Icon from '../components/Icon.vue'
import { money } from '../utils/format'
import { sumBy } from '../utils/charts'

const incomes = ref<Income[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const submitting = ref(false)

const showForm = ref(false)
const form = ref({
  amount: 0,
  currency: 'USD',
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
const currency = computed(() => incomes.value[0]?.currency ?? '$')

const topSource = computed(() => {
  const bySrc = new Map<string, number>()
  for (const i of filtered.value) {
    const k = i.source || 'Unspecified'
    bySrc.set(k, (bySrc.get(k) ?? 0) + i.amount)
  }
  let best = ''
  let bestVal = 0
  for (const [k, v] of bySrc.entries()) {
    if (v > bestVal) {
      bestVal = v
      best = k
    }
  }
  return best || '—'
})

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
    toast.flash('Income logged')
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

function when(dateStr: string): string {
  const d = new Date(dateStr)
  const now = new Date()
  const mid = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate())
  const diff = (mid(now).getTime() - mid(d).getTime()) / (24 * 3600 * 1000)
  if (diff === 0) return 'TODAY'
  if (diff === 1) return 'YESTERDAY'
  if (diff < 7) return d.toLocaleDateString(undefined, { weekday: 'short' }).toUpperCase()
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }).toUpperCase()
}

function kindFor(src?: string): string {
  const k = (src ?? '').toLowerCase()
  if (/(salary|payroll)/.test(k)) return 'SALARY'
  if (/(freelance|gig)/.test(k)) return 'FREELANCE'
  if (/(reimburs)/.test(k)) return 'REIMBURSE'
  if (/(top|kitty|fund)/.test(k)) return 'TOP-UP'
  if (/(dividend|invest)/.test(k)) return 'DIVIDEND'
  return 'INCOME'
}

onMounted(load)
</script>

<template>
  <section>
    <div class="eyebrow">LEDGER · INCOMES</div>
    <div class="row spread" style="align-items: flex-start; margin-top: 4px">
      <h1 class="hero">Money <em class="moss-em">in.</em></h1>
      <button class="btn btn-primary" @click="showForm = !showForm">
        {{ showForm ? 'Cancel' : '+ New' }}
      </button>
    </div>

    <!-- Total card -->
    <div class="card-moss total-card">
      <div class="label">TOTAL RECEIVED</div>
      <div class="money-big">
        <span class="cur">$</span>
        {{ Math.floor(total).toLocaleString()
        }}<span class="decimals">.{{
          (total % 1).toFixed(2).slice(2)
        }}</span>
      </div>
      <div class="subline">
        {{ filtered.length }} entr{{ filtered.length === 1 ? 'y' : 'ies' }} ·
        <span v-if="topSource && topSource !== '—'">top source · {{ topSource }}</span>
      </div>
    </div>

    <!-- Add form -->
    <form v-if="showForm" class="stack-form" @submit.prevent="create">
      <div class="split">
        <label class="field">
          <span>AMOUNT</span>
          <input
            v-model.number="form.amount"
            type="number"
            min="0"
            step="0.01"
            required
          />
        </label>
        <label class="field">
          <span>CURRENCY</span>
          <input v-model="form.currency" />
        </label>
      </div>
      <label class="field">
        <span>SOURCE</span>
        <input v-model="form.source" placeholder="Salary, Freelance…" />
      </label>
      <label class="field">
        <span>DESCRIPTION</span>
        <input v-model="form.description" />
      </label>
      <label class="field">
        <span>DATE</span>
        <input v-model="form.date" type="date" required />
      </label>
      <button class="btn btn-primary btn-lg" type="submit" :disabled="submitting">
        <Icon name="check" :size="16" />
        {{ submitting ? 'Saving…' : 'Save income' }}
      </button>
    </form>

    <!-- Filter -->
    <div class="filters">
      <label class="field">
        <span>SOURCE</span>
        <input v-model="filterSource" placeholder="Filter by source" />
      </label>
      <div class="total">
        TOTAL · <b>{{ money(total, currency) }}</b>
      </div>
    </div>

    <!-- List -->
    <div class="row spread" style="margin-top: 8px">
      <span class="eyebrow">LEDGER</span>
      <span class="eyebrow">NEWEST FIRST</span>
    </div>

    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <template v-else>
      <div v-if="filtered.length">
        <div
          v-for="(i, idx) in filtered"
          :key="i.id"
          class="income-row"
          :class="{ last: idx === filtered.length - 1 }"
        >
          <div style="min-width: 0">
            <div class="title">
              {{ i.source || 'Unspecified' }}<span v-if="i.description"> · {{ i.description }}</span>
            </div>
            <div class="sub">{{ when(i.date) }} · {{ kindFor(i.source) }}</div>
          </div>
          <div class="figure">
            <span class="cur">+{{ i.currency || '$' }}</span>
            {{ i.amount.toFixed(2) }}
            <button
              class="linklike"
              style="margin-left: 6px; color: var(--ink-ghost)"
              @click="remove(String(i.id))"
              aria-label="Delete"
            >
              <Icon name="close" :size="14" />
            </button>
          </div>
        </div>
      </div>
      <div v-else class="empty">NO INCOMES YET</div>
    </template>
  </section>
</template>

<style scoped>
.moss-em {
  color: var(--moss) !important;
}

.total-card {
  margin-top: 20px;
}
.total-card .label {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.12em;
  color: rgba(245, 241, 232, 0.65);
  text-transform: uppercase;
}
.total-card .money-big {
  font-family: var(--serif);
  font-size: 54px;
  line-height: 1;
  margin-top: 4px;
  letter-spacing: -0.02em;
}
.total-card .money-big .cur {
  font-size: 22px;
  color: rgba(245, 241, 232, 0.6);
  vertical-align: top;
}
.total-card .money-big .decimals {
  font-size: 26px;
  color: rgba(245, 241, 232, 0.6);
}
.total-card .subline {
  font-family: var(--mono);
  font-size: 11px;
  margin-top: 6px;
  color: rgba(245, 241, 232, 0.7);
}

.income-row {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 14px;
  align-items: center;
  padding: 14px 0;
  border-bottom: 1px solid var(--line);
}
.income-row.last {
  border-bottom: none;
}
.income-row .title {
  font-size: 15px;
  font-weight: 500;
  color: var(--ink);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.income-row .sub {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  color: var(--ink-mute);
  margin-top: 3px;
  text-transform: uppercase;
}
.income-row .figure {
  font-family: var(--serif);
  font-size: 22px;
  color: var(--moss);
  line-height: 1;
  text-align: right;
}
.income-row .figure .cur {
  font-size: 12px;
  color: var(--ink-mute);
  margin-right: 1px;
}
</style>
