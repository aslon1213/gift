<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { groupApi, spendingApi, userApi } from '../api/endpoints'
import type { Group, Spending, User } from '../api/types'
import { auth } from '../stores/auth'
import DonutChart from '../components/DonutChart.vue'
import BarChart from '../components/BarChart.vue'
import { formatDay, groupBy, lastNDays, sumBy } from '../utils/charts'

const props = defineProps<{ id: string }>()

const group = ref<Group | null>(null)
const spendings = ref<Spending[]>([])
const members = ref<Record<string, User>>({})
const loading = ref(true)
const error = ref<string | null>(null)
const busy = ref(false)
const router = useRouter()

const isOwner = computed(
  () => !!group.value && group.value.owner_id === auth.userIdFromToken(),
)

// --- members modal --------------------------------------------------------

const showMembers = ref(false)
const modalError = ref<string | null>(null)
const searchQuery = ref('')
const searchResults = ref<User[]>([])
const searching = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | null = null

function openMembers() {
  showMembers.value = true
  searchQuery.value = ''
  searchResults.value = []
  modalError.value = null
}

function closeMembers() {
  showMembers.value = false
  searchQuery.value = ''
  searchResults.value = []
}

function onSearchInput() {
  if (searchTimer) clearTimeout(searchTimer)
  const q = searchQuery.value.trim()
  if (!q) {
    searchResults.value = []
    return
  }
  searchTimer = setTimeout(() => runSearch(q), 250)
}

async function runSearch(q: string) {
  searching.value = true
  try {
    const res = await userApi.search(q)
    searchResults.value = res ?? []
  } catch (e) {
    modalError.value = e instanceof Error ? e.message : 'Search failed'
  } finally {
    searching.value = false
  }
}

function userId(u: User): string {
  return String(u._id ?? u.id ?? '')
}

async function addMember(u: User) {
  if (!group.value) return
  const uid = userId(u)
  if (!uid || group.value.member_ids?.includes(uid)) return
  busy.value = true
  modalError.value = null
  try {
    const g = await groupApi.invite(props.id, uid)
    group.value = g
    members.value = { ...members.value, [uid]: u }
    searchQuery.value = ''
    searchResults.value = []
  } catch (e) {
    modalError.value = e instanceof Error ? e.message : 'Failed to add'
  } finally {
    busy.value = false
  }
}

async function removeMember(mid: string) {
  if (!confirm('Remove this member?')) return
  busy.value = true
  modalError.value = null
  try {
    const g = await groupApi.removeMember(props.id, mid)
    group.value = g
  } catch (e) {
    modalError.value = e instanceof Error ? e.message : 'Remove failed'
  } finally {
    busy.value = false
  }
}

async function loadMembers(ids: string[]) {
  const unique = Array.from(new Set(ids))
  const missing = unique.filter((id) => !members.value[id])
  const results = await Promise.allSettled(missing.map((id) => userApi.getById(id)))
  const next = { ...members.value }
  results.forEach((r, i) => {
    if (r.status === 'fulfilled' && r.value) next[missing[i]] = r.value
  })
  members.value = next
}

function memberName(uid: string) {
  return members.value[uid]?.name || 'Unknown'
}

function memberEmail(uid: string) {
  return members.value[uid]?.email || ''
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (!parts.length) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

// --- rename ---------------------------------------------------------------

const renaming = ref(false)
const renameValue = ref('')

function startRename() {
  if (!group.value) return
  renameValue.value = group.value.name
  renaming.value = true
}

async function saveRename() {
  if (!renameValue.value.trim()) return
  busy.value = true
  error.value = null
  try {
    const g = await groupApi.update(props.id, { name: renameValue.value.trim() })
    group.value = g
    renaming.value = false
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Rename failed'
  } finally {
    busy.value = false
  }
}

// --- add spending ---------------------------------------------------------

const showSpendingForm = ref(false)
const spendingForm = ref({
  amount: 0,
  currency: 'UZS',
  category: '',
  description: '',
  date: new Date().toISOString().slice(0, 10),
})
const creatingSpending = ref(false)

async function createSpending() {
  creatingSpending.value = true
  error.value = null
  try {
    const s = await spendingApi.create({
      group_id: props.id,
      amount: Number(spendingForm.value.amount),
      currency: spendingForm.value.currency,
      category: spendingForm.value.category,
      description: spendingForm.value.description,
      date: new Date(spendingForm.value.date).toISOString(),
    })
    if (s) spendings.value = [s, ...spendings.value]
    showSpendingForm.value = false
    spendingForm.value.amount = 0
    spendingForm.value.description = ''
    spendingForm.value.category = ''
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Create failed'
  } finally {
    creatingSpending.value = false
  }
}

async function removeSpending(id: string) {
  if (!confirm('Delete this spending?')) return
  try {
    await spendingApi.remove(id)
    spendings.value = spendings.value.filter((s) => String(s._id ?? s.id) !== id)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

// --- derived / charts -----------------------------------------------------

const currency = computed(() => spendings.value[0]?.currency ?? '')

const total = computed(() => sumBy(spendings.value, (s) => s.amount))

const categoryData = computed(() => {
  const byCat = groupBy(spendings.value, (s) => s.category || 'Uncategorized')
  return Array.from(byCat.entries()).map(([label, items]) => ({
    label,
    value: sumBy(items, (i) => i.amount),
  }))
})

const memberData = computed(() => {
  const byUser = groupBy(spendings.value, (s) => s.user_id)
  return Array.from(byUser.entries()).map(([uid, items]) => ({
    label: memberName(uid),
    value: sumBy(items, (i) => i.amount),
  }))
})

const dailyData = computed(() => {
  const days = lastNDays(30)
  const byDay = groupBy(spendings.value, (s) => formatDay(new Date(s.date)))
  return days.map((d) => ({
    label: d.slice(5),
    value: sumBy(byDay.get(d) ?? [], (i) => i.amount),
  }))
})

// --- load ------------------------------------------------------------------

async function load() {
  loading.value = true
  error.value = null
  try {
    const g = await groupApi.get(props.id)
    group.value = g
    const [sp] = await Promise.all([
      spendingApi.query({ group_id: props.id }),
      g ? loadMembers([g.owner_id, ...(g.member_ids ?? [])]) : Promise.resolve(),
    ])
    spendings.value = sp ?? []
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function deleteGroup() {
  if (!confirm('Delete this group? This cannot be undone.')) return
  try {
    await groupApi.remove(props.id)
    router.push('/groups')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

onMounted(load)
watch(() => props.id, load)

const myId = computed(() => auth.userIdFromToken() ?? '')
</script>

<template>
  <section>
    <router-link to="/groups" class="back">← Groups</router-link>
    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <template v-else-if="group">
      <div class="group-header">
        <div style="min-width: 0">
          <div v-if="renaming" class="inline-form" style="margin: 0">
            <input v-model="renameValue" :disabled="busy" @keyup.enter="saveRename" @keyup.esc="renaming = false" autofocus />
            <button type="button" @click="saveRename" :disabled="busy">Save</button>
          </div>
          <div v-else class="row" style="gap: 8px; flex-wrap: wrap">
            <h1 style="margin: 0">{{ group.name }}</h1>
            <button v-if="isOwner" class="linklike small" @click="startRename">Rename</button>
          </div>
          <div class="meta">
            Owner {{ memberName(group.owner_id) }} · {{ group.member_ids?.length ?? 0 }} members · {{ spendings.length }} spendings
          </div>
        </div>
        <div class="actions">
          <button class="ghost" @click="openMembers">Members</button>
          <button @click="showSpendingForm = !showSpendingForm">
            {{ showSpendingForm ? 'Cancel' : '+ Add' }}
          </button>
        </div>
      </div>

      <form v-if="showSpendingForm" class="stack-form" @submit.prevent="createSpending">
        <div class="split">
          <label class="field">
            <span>Amount</span>
            <input v-model.number="spendingForm.amount" type="number" min="0" step="0.01" required />
          </label>
          <label class="field">
            <span>Currency</span>
            <input v-model="spendingForm.currency" />
          </label>
        </div>
        <label class="field">
          <span>Category</span>
          <input v-model="spendingForm.category" placeholder="Food, Transport…" />
        </label>
        <label class="field">
          <span>Description</span>
          <input v-model="spendingForm.description" />
        </label>
        <label class="field">
          <span>Date</span>
          <input v-model="spendingForm.date" type="date" required />
        </label>
        <button type="submit" :disabled="creatingSpending">
          {{ creatingSpending ? 'Saving…' : 'Save spending' }}
        </button>
      </form>

      <div class="stats">
        <div class="card">
          <div class="stat-label">Total</div>
          <div class="stat-value">{{ total.toLocaleString() }} {{ currency }}</div>
        </div>
        <div class="card">
          <div class="stat-label">30 days</div>
          <div class="stat-value">
            {{ dailyData.reduce((a, d) => a + d.value, 0).toLocaleString() }}
          </div>
        </div>
        <div class="card">
          <div class="stat-label">Top category</div>
          <div class="stat-value" style="font-size: 16px">
            {{ categoryData.slice().sort((a, b) => b.value - a.value)[0]?.label ?? '—' }}
          </div>
        </div>
      </div>

      <div class="chart-grid">
        <DonutChart title="By category" :data="categoryData" :currency="currency" />
        <DonutChart title="By member" :data="memberData" :currency="currency" />
      </div>
      <BarChart title="Last 30 days" :data="dailyData" />

      <h2>Spendings</h2>
      <ul v-if="spendings.length" class="list">
        <li v-for="s in spendings" :key="s._id ?? s.id">
          <div style="min-width: 0">
            <strong>{{ s.category || 'Uncategorized' }}</strong>
            <span class="muted"> · {{ memberName(s.user_id) }}</span>
            <div class="muted small">
              {{ s.description || 'no note' }} · {{ new Date(s.date).toLocaleDateString() }}
            </div>
          </div>
          <div class="row">
            <span>{{ s.amount.toLocaleString() }} {{ s.currency }}</span>
            <button
              v-if="s.user_id === myId"
              class="danger"
              @click="removeSpending(String(s._id ?? s.id))"
            >
              Delete
            </button>
          </div>
        </li>
      </ul>
      <p v-else class="muted">No spendings yet.</p>

      <div v-if="isOwner" style="margin-top: 24px">
        <button class="danger" @click="deleteGroup">Delete group</button>
      </div>
    </template>

    <!-- Members modal -->
    <div v-if="showMembers && group" class="modal-backdrop" @click.self="closeMembers">
      <div class="modal">
        <div class="modal-header">
          <div class="modal-title">Members · {{ group.name }}</div>
          <button class="icon-btn ghost" aria-label="Close" @click="closeMembers">×</button>
        </div>
        <div class="modal-body">
          <p v-if="modalError" class="error">{{ modalError }}</p>

          <div v-if="isOwner">
            <input
              v-model="searchQuery"
              placeholder="Search people by name…"
              @input="onSearchInput"
            />
            <div v-if="searching" class="muted small" style="margin-top: 8px">
              Searching…
            </div>
            <div v-else-if="searchQuery && !searchResults.length" class="muted small" style="margin-top: 8px">
              No matches
            </div>
            <div v-else-if="searchResults.length" style="margin-top: 10px">
              <div
                v-for="u in searchResults"
                :key="userId(u)"
                class="user-row"
              >
                <div class="avatar">{{ initials(u.name) }}</div>
                <div class="info">
                  <div class="name">{{ u.name }}</div>
                  <div class="email">{{ u.email }}</div>
                </div>
                <button
                  v-if="group.member_ids?.includes(userId(u))"
                  class="linklike"
                  disabled
                >
                  Added
                </button>
                <button
                  v-else
                  class="ghost"
                  :disabled="busy"
                  @click="addMember(u)"
                >
                  Add
                </button>
              </div>
            </div>
          </div>

          <h2 style="margin-top: 18px">Current ({{ group.member_ids?.length ?? 0 }})</h2>
          <div
            v-for="m in group.member_ids"
            :key="m"
            class="user-row"
          >
            <div class="avatar">{{ initials(memberName(m)) }}</div>
            <div class="info">
              <div class="name">
                {{ memberName(m) }}
                <span v-if="m === group.owner_id" class="badge">owner</span>
                <span v-if="m === myId" class="badge">you</span>
              </div>
              <div class="email">{{ memberEmail(m) }}</div>
            </div>
            <button
              v-if="m !== group.owner_id && isOwner"
              class="danger"
              :disabled="busy"
              @click="removeMember(m)"
            >
              Remove
            </button>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
