<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { groupApi, spendingApi } from '../api/endpoints'
import type { Group, Spending } from '../api/types'
import { auth } from '../stores/auth'

const groups = ref<Group[]>([])
const spendings = ref<Spending[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

const filterGroup = ref<string>('')
const filterCategory = ref<string>('')

const showForm = ref(false)
const form = ref({
  group_id: '',
  amount: 0,
  currency: 'UZS',
  category: '',
  description: '',
  date: new Date().toISOString().slice(0, 10),
})
const submitting = ref(false)

const total = computed(() =>
  spendings.value.reduce((a, s) => a + (s.amount ?? 0), 0),
)

async function loadGroups() {
  groups.value = (await groupApi.list()) ?? []
  if (!form.value.group_id && groups.value[0]) {
    form.value.group_id = String(groups.value[0]._id ?? groups.value[0].id ?? '')
  }
}

async function load() {
  loading.value = true
  error.value = null
  try {
    const uid = auth.userIdFromToken() ?? undefined
    const q = {
      user_id: uid,
      group_id: filterGroup.value || undefined,
      category: filterCategory.value || undefined,
    }
    spendings.value = (await spendingApi.query(q)) ?? []
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
    await spendingApi.create({
      group_id: form.value.group_id,
      amount: Number(form.value.amount),
      currency: form.value.currency,
      category: form.value.category,
      description: form.value.description,
      date: new Date(form.value.date).toISOString(),
    })
    showForm.value = false
    form.value.amount = 0
    form.value.description = ''
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Create failed'
  } finally {
    submitting.value = false
  }
}

async function remove(id: string) {
  if (!confirm('Delete this spending?')) return
  try {
    await spendingApi.remove(id)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

function groupName(id: string) {
  const g = groups.value.find((x) => String(x._id ?? x.id) === id)
  return g?.name ?? 'Unknown'
}

onMounted(async () => {
  await loadGroups()
  await load()
})

watch([filterGroup, filterCategory], load)
</script>

<template>
  <section>
    <header class="row spread">
      <h1>Spendings</h1>
      <button class="ghost" @click="showForm = !showForm" :disabled="!groups.length">
        {{ showForm ? 'Cancel' : '+ New' }}
      </button>
    </header>

    <p v-if="!groups.length" class="muted">
      You need a group first. <router-link to="/groups">Create one</router-link>.
    </p>

    <form v-if="showForm" class="stack-form" @submit.prevent="create">
      <label class="field">
        <span>Group</span>
        <select v-model="form.group_id" required>
          <option v-for="g in groups" :key="g._id ?? g.id" :value="String(g._id ?? g.id)">
            {{ g.name }}
          </option>
        </select>
      </label>
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
        <span>Category</span>
        <input v-model="form.category" placeholder="Food, Transport…" />
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
        {{ submitting ? 'Saving…' : 'Save spending' }}
      </button>
    </form>

    <div class="filters">
      <label class="field">
        <span>Group</span>
        <select v-model="filterGroup">
          <option value="">All</option>
          <option v-for="g in groups" :key="g._id ?? g.id" :value="String(g._id ?? g.id)">
            {{ g.name }}
          </option>
        </select>
      </label>
      <label class="field">
        <span>Category</span>
        <input v-model="filterCategory" placeholder="Filter" />
      </label>
    </div>
    <div class="muted small" style="text-align: right; margin-bottom: 8px">
      Total: <strong>{{ total.toLocaleString() }}</strong>
    </div>

    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <ul v-else-if="spendings.length" class="list">
      <li v-for="s in spendings" :key="s._id ?? s.id">
        <div style="min-width: 0">
          <strong>{{ s.category || 'Uncategorized' }}</strong>
          <span class="muted"> · {{ groupName(s.group_id) }}</span>
          <div class="muted small">
            {{ s.description || 'no note' }} · {{ new Date(s.date).toLocaleDateString() }}
          </div>
        </div>
        <div class="row">
          <span>{{ s.amount.toLocaleString() }} {{ s.currency }}</span>
          <button class="danger" @click="remove(String(s._id ?? s.id))">Delete</button>
        </div>
      </li>
    </ul>
    <div v-else class="empty">No spendings match.</div>
  </section>
</template>
