<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { groupApi, spendingApi, userApi } from '../api/endpoints'
import type { Group, Spending, User } from '../api/types'
import { auth } from '../stores/auth'
import DonutChart from '../components/DonutChart.vue'
import BarChart from '../components/BarChart.vue'
import { formatDay, groupBy, lastNDays, sumBy } from '../utils/charts'

const groups = ref<Group[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

const showForm = ref(false)
const name = ref('')
const creating = ref(false)

const memberCache = ref<Record<string, User>>({})

// --- Members modal --------------------------------------------------------

const modalGroup = ref<Group | null>(null)
const modalBusy = ref(false)
const modalError = ref<string | null>(null)
const searchQuery = ref('')
const searchResults = ref<User[]>([])
const searching = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | null = null

function openMembers(g: Group) {
  modalGroup.value = g
  searchQuery.value = ''
  searchResults.value = []
  modalError.value = null
  loadMembersFor(g)
}

function closeModal() {
  modalGroup.value = null
  searchQuery.value = ''
  searchResults.value = []
  modalError.value = null
}

async function loadMembersFor(g: Group) {
  const ids = Array.from(new Set([g.owner_id, ...(g.member_ids ?? [])]))
  const missing = ids.filter((id) => !memberCache.value[id])
  if (!missing.length) return
  const results = await Promise.allSettled(missing.map((id) => userApi.getById(id)))
  const next = { ...memberCache.value }
  results.forEach((r, i) => {
    if (r.status === 'fulfilled' && r.value) next[missing[i]] = r.value
  })
  memberCache.value = next
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

function memberName(uid: string): string {
  return memberCache.value[uid]?.name || 'Unknown'
}

function memberEmail(uid: string): string {
  return memberCache.value[uid]?.email || ''
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (!parts.length) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

async function addMember(u: User) {
  if (!modalGroup.value) return
  const uid = userId(u)
  if (!uid) return
  if (modalGroup.value.member_ids?.includes(uid)) return
  modalBusy.value = true
  modalError.value = null
  try {
    const updated = await groupApi.invite(
      String(modalGroup.value._id ?? modalGroup.value.id),
      uid,
    )
    modalGroup.value = updated
    memberCache.value = { ...memberCache.value, [uid]: u }
    // sync into list
    const gid = String(updated._id ?? updated.id)
    groups.value = groups.value.map((g) =>
      String(g._id ?? g.id) === gid ? updated : g,
    )
    searchQuery.value = ''
    searchResults.value = []
  } catch (e) {
    modalError.value = e instanceof Error ? e.message : 'Failed to add'
  } finally {
    modalBusy.value = false
  }
}

async function removeMember(uid: string) {
  if (!modalGroup.value) return
  if (!confirm('Remove this member?')) return
  modalBusy.value = true
  modalError.value = null
  try {
    const updated = await groupApi.removeMember(
      String(modalGroup.value._id ?? modalGroup.value.id),
      uid,
    )
    modalGroup.value = updated
    const gid = String(updated._id ?? updated.id)
    groups.value = groups.value.map((g) =>
      String(g._id ?? g.id) === gid ? updated : g,
    )
  } catch (e) {
    modalError.value = e instanceof Error ? e.message : 'Failed to remove'
  } finally {
    modalBusy.value = false
  }
}

// --- Stats ----------------------------------------------------------------

const groupSpendings = ref<Record<string, Spending[]>>({})
const statsLoading = ref(false)

async function loadStats() {
  if (!groups.value.length) {
    groupSpendings.value = {}
    return
  }
  statsLoading.value = true
  try {
    const entries = await Promise.all(
      groups.value.map(async (g) => {
        const id = String(g._id ?? g.id)
        const sp = await spendingApi.query({ group_id: id }).catch(() => [] as Spending[])
        return [id, sp ?? []] as const
      }),
    )
    const map: Record<string, Spending[]> = {}
    for (const [id, sp] of entries) map[id] = sp
    groupSpendings.value = map
  } finally {
    statsLoading.value = false
  }
}

const allSpendings = computed(() => Object.values(groupSpendings.value).flat())

const statsCurrency = computed(() => allSpendings.value[0]?.currency ?? '')

const totalAcrossGroups = computed(() =>
  sumBy(allSpendings.value, (s) => s.amount),
)

const spendingByGroup = computed(() =>
  groups.value.map((g) => {
    const id = String(g._id ?? g.id)
    const sp = groupSpendings.value[id] ?? []
    return { label: g.name, value: sumBy(sp, (s) => s.amount) }
  }),
)

const membersByGroup = computed(() =>
  groups.value.map((g) => ({
    label: g.name,
    value: g.member_ids?.length ?? 0,
  })),
)

const categoryBreakdown = computed(() => {
  const byCat = groupBy(allSpendings.value, (s) => s.category || 'Uncategorized')
  return Array.from(byCat.entries()).map(([label, items]) => ({
    label,
    value: sumBy(items, (i) => i.amount),
  }))
})

const groupBreakdown = computed(() =>
  spendingByGroup.value.filter((g) => g.value > 0),
)

const dailyAcrossGroups = computed(() => {
  const days = lastNDays(30)
  const byDay = groupBy(allSpendings.value, (s) => formatDay(new Date(s.date)))
  return days.map((d) => ({
    label: d.slice(5),
    value: sumBy(byDay.get(d) ?? [], (i) => i.amount),
  }))
})

const topGroup = computed(() => {
  const sorted = spendingByGroup.value.slice().sort((a, b) => b.value - a.value)
  return sorted[0]?.value ? sorted[0] : null
})

const avgPerGroup = computed(() => {
  if (!groups.value.length) return 0
  return totalAcrossGroups.value / groups.value.length
})

// --- List / create --------------------------------------------------------

async function load() {
  loading.value = true
  error.value = null
  try {
    groups.value = (await groupApi.list()) ?? []
    // Preload owner + member info so list can show names
    const ids = new Set<string>()
    for (const g of groups.value) {
      ids.add(g.owner_id)
      for (const m of g.member_ids ?? []) ids.add(m)
    }
    const missing = Array.from(ids).filter((id) => !memberCache.value[id])
    const results = await Promise.allSettled(missing.map((id) => userApi.getById(id)))
    const next = { ...memberCache.value }
    results.forEach((r, i) => {
      if (r.status === 'fulfilled' && r.value) next[missing[i]] = r.value
    })
    memberCache.value = next
    loadStats()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function createGroup() {
  const uid = auth.userIdFromToken()
  if (!uid) {
    error.value = 'Missing user id'
    return
  }
  creating.value = true
  error.value = null
  try {
    await groupApi.create({ name: name.value, owner_id: uid, member_ids: [uid] })
    name.value = ''
    showForm.value = false
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to create'
  } finally {
    creating.value = false
  }
}

async function remove(id: string) {
  if (!confirm('Delete this group?')) return
  try {
    await groupApi.remove(id)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

const myId = auth.userIdFromToken() ?? ''

onMounted(load)
</script>

<template>
  <section>
    <header class="row spread">
      <h1>Groups</h1>
      <button class="ghost" @click="showForm = !showForm">
        {{ showForm ? 'Cancel' : '+ New' }}
      </button>
    </header>

    <form v-if="showForm" class="inline-form" @submit.prevent="createGroup">
      <input v-model="name" placeholder="Group name" required />
      <button type="submit" :disabled="creating">
        {{ creating ? '…' : 'Create' }}
      </button>
    </form>

    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <template v-else>
      <ul v-if="groups.length" class="list">
        <li v-for="g in groups" :key="g._id ?? g.id">
          <router-link :to="`/groups/${g._id ?? g.id}`">
            <strong>{{ g.name }}</strong>
            <div class="muted small">
              {{ g.member_ids?.length ?? 0 }} members
              · owner {{ memberName(g.owner_id) }}
            </div>
          </router-link>
          <div class="row">
            <button class="linklike" @click="openMembers(g)">Members</button>
            <button
              v-if="g.owner_id === myId"
              class="danger"
              @click="remove(String(g._id ?? g.id))"
            >
              Delete
            </button>
          </div>
        </li>
      </ul>
      <div v-else class="empty">No groups yet. Create your first one above.</div>
    </template>

    <section v-if="groups.length" class="stats-section">
      <h2>Stats</h2>
      <p v-if="statsLoading" class="muted small">Loading stats…</p>

      <div class="stats">
        <div class="card">
          <div class="stat-label">Total spent</div>
          <div class="stat-value">
            {{ totalAcrossGroups.toLocaleString() }} {{ statsCurrency }}
          </div>
        </div>
        <div class="card">
          <div class="stat-label">Groups</div>
          <div class="stat-value">{{ groups.length }}</div>
        </div>
        <div class="card">
          <div class="stat-label">Avg per group</div>
          <div class="stat-value">{{ Math.round(avgPerGroup).toLocaleString() }}</div>
        </div>
        <div class="card">
          <div class="stat-label">Top group</div>
          <div class="stat-value" style="font-size: 16px">
            {{ topGroup?.label ?? '—' }}
          </div>
        </div>
      </div>

      <BarChart title="Spending by group" :data="spendingByGroup" />

      <div class="chart-grid">
        <DonutChart
          title="Share by group"
          :data="groupBreakdown"
          :currency="statsCurrency"
        />
        <DonutChart
          title="Categories across groups"
          :data="categoryBreakdown"
          :currency="statsCurrency"
        />
      </div>

      <BarChart title="Members per group" :data="membersByGroup" />
      <BarChart title="Last 30 days · all groups" :data="dailyAcrossGroups" />
    </section>

    <!-- Members modal -->

    <div v-if="modalGroup" class="modal-backdrop" @click.self="closeModal">
      <div class="modal">
        <div class="modal-header">
          <div class="modal-title">Members · {{ modalGroup.name }}</div>
          <button class="icon-btn ghost" aria-label="Close" @click="closeModal">×</button>
        </div>
        <div class="modal-body">
          <p v-if="modalError" class="error">{{ modalError }}</p>

          <div v-if="modalGroup.owner_id === myId">
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
                  v-if="modalGroup.member_ids?.includes(userId(u))"
                  class="linklike"
                  disabled
                >
                  Added
                </button>
                <button
                  v-else
                  class="ghost"
                  :disabled="modalBusy"
                  @click="addMember(u)"
                >
                  Add
                </button>
              </div>
            </div>
          </div>

          <h2 style="margin-top: 18px">Current ({{ modalGroup.member_ids?.length ?? 0 }})</h2>
          <div
            v-for="m in modalGroup.member_ids"
            :key="m"
            class="user-row"
          >
            <div class="avatar">{{ initials(memberName(m)) }}</div>
            <div class="info">
              <div class="name">
                {{ memberName(m) }}
                <span v-if="m === modalGroup.owner_id" class="badge">owner</span>
                <span v-if="m === myId" class="badge">you</span>
              </div>
              <div class="email">{{ memberEmail(m) }}</div>
            </div>
            <button
              v-if="m !== modalGroup.owner_id && modalGroup.owner_id === myId"
              class="danger"
              :disabled="modalBusy"
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

<style scoped>
.stats-section {
  margin-top: 32px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
}
.stats-section > * + * {
  margin-top: 12px;
}
</style>

