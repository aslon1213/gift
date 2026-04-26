<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { budgetApi, groupApi, spendingApi, userApi } from '../api/endpoints'
import type { Budget, Group, Spending, User } from '../api/types'
import { auth } from '../stores/auth'
import { toast } from '../stores/toast'
import Icon from '../components/Icon.vue'
import Avatar from '../components/Avatar.vue'
import VoiceInputButton from '../components/VoiceInputButton.vue'
import type { IconName } from '../components/icons'
import { colorForId, money } from '../utils/format'
import { sumBy } from '../utils/charts'
import { parseSpendingFromAudio, type SpendingDraft } from '../ai/parse'
import { t } from '../i18n'

const groups = ref<Group[]>([])
const spendings = ref<Spending[]>([])
const members = ref<Record<string, User>>({})
const budgets = ref<Budget[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

function budgetTagsFor(s: Spending): string[] {
  const ids = s.budgets ?? []
  if (!ids.length || !budgets.value.length) return []
  const byId = new Map(budgets.value.map((b) => [String(b.id), b.category]))
  const tags: string[] = []
  for (const id of ids) {
    const cat = byId.get(String(id))
    if (cat) tags.push(cat)
  }
  return tags
}

const filterGroup = ref<string>('')
const filterCategory = ref<string>('')

// ---------- Add-expense modal ----------

interface Category {
  id: IconName
  label: string
}
const CATEGORIES: Category[] = [
  { id: 'fork', label: 'Food' },
  { id: 'plane', label: 'Travel' },
  { id: 'bed', label: 'Stay' },
  { id: 'car', label: 'Transport' },
  { id: 'ticket', label: 'Activity' },
  { id: 'cart', label: 'Groceries' },
  { id: 'coffee', label: 'Coffee' },
  { id: 'home2', label: 'Home' },
]

const showAdd = ref(false)
const step = ref(0)
const amountStr = ref('')
const category = ref<Category>(CATEGORIES[0])
const payerId = ref<string>('')
const splitMode = ref<'equal' | 'by_shares' | 'custom'>('equal')
const includedIds = ref<Set<string>>(new Set())
const description = ref('')
const addDate = ref(new Date().toISOString().slice(0, 10))
const addCurrency = ref('USD')
const addGroupId = ref<string>('')
const submitting = ref(false)
const keepAfterClose = ref(false)

const myId = computed(() => auth.userIdFromToken() ?? '')

const amountNum = computed(() => parseFloat(amountStr.value || '0'))

const activeGroup = computed<Group | null>(() => {
  if (!addGroupId.value) return null
  return (
    groups.value.find(
      (g) => String(g._id ?? g.id) === addGroupId.value,
    ) ?? null
  )
})

const groupMembers = computed<User[]>(() => {
  const g = activeGroup.value
  if (!g) return []
  const ids = Array.from(new Set([g.owner_id, ...(g.member_ids ?? [])]))
  return ids.map((id) => members.value[id]).filter(Boolean) as User[]
})

const includedList = computed<User[]>(() =>
  groupMembers.value.filter((u) =>
    includedIds.value.has(String(u._id ?? u.id)),
  ),
)

const perPerson = computed(() =>
  includedList.value.length ? amountNum.value / includedList.value.length : 0,
)

function openAdd() {
  step.value = 0
  amountStr.value = ''
  category.value = CATEGORIES[0]
  description.value = ''
  addDate.value = new Date().toISOString().slice(0, 10)
  submitting.value = false
  addCurrency.value = spendings.value[0]?.currency ?? 'USD'
  // default group: first available or filter
  const first = filterGroup.value || String(groups.value[0]?._id ?? groups.value[0]?.id ?? '')
  addGroupId.value = first
  payerId.value = myId.value
  includedIds.value = new Set(
    (activeGroup.value
      ? Array.from(
          new Set([
            activeGroup.value.owner_id,
            ...(activeGroup.value.member_ids ?? []),
          ]),
        )
      : []),
  )
  showAdd.value = true
}

function closeAdd() {
  showAdd.value = false
  keepAfterClose.value = false
}

watch(addGroupId, async (gid) => {
  if (!gid) return
  const g = groups.value.find((x) => String(x._id ?? x.id) === gid)
  if (!g) return
  await loadGroupMembers(g)
  payerId.value = myId.value || g.owner_id
  includedIds.value = new Set(
    Array.from(new Set([g.owner_id, ...(g.member_ids ?? [])])),
  )
})

function onKey(k: string) {
  if (k === '⌫') {
    amountStr.value = amountStr.value.slice(0, -1)
    return
  }
  if (k === '.' && amountStr.value.includes('.')) return
  if (amountStr.value.replace('.', '').length >= 7) return
  amountStr.value = amountStr.value + k
}

function displayAmount(): string {
  if (!amountStr.value) return '0.00'
  if (amountStr.value.includes('.')) {
    const [i, d] = amountStr.value.split('.')
    return (i || '0') + '.' + (d + '00').slice(0, 2)
  }
  return parseInt(amountStr.value).toLocaleString() + '.00'
}

function toggleIncluded(uid: string) {
  const next = new Set(includedIds.value)
  if (next.has(uid)) next.delete(uid)
  else next.add(uid)
  includedIds.value = next
}

async function loadGroupMembers(g: Group) {
  const ids = Array.from(new Set([g.owner_id, ...(g.member_ids ?? [])]))
  const missing = ids.filter((id) => !members.value[id])
  const results = await Promise.allSettled(
    missing.map((id) => userApi.getById(id)),
  )
  const next = { ...members.value }
  results.forEach((r, i) => {
    if (r.status === 'fulfilled' && r.value) next[missing[i]] = r.value
  })
  members.value = next
}

function applySpendingDraft(draft: SpendingDraft) {
  if (draft.amount != null && draft.amount > 0) {
    amountStr.value = String(draft.amount)
  }
  if (draft.currency) {
    addCurrency.value = draft.currency.toUpperCase()
  }
  if (draft.category) {
    const match = CATEGORIES.find(
      (c) => c.label.toLowerCase() === draft.category!.toLowerCase(),
    )
    if (match) category.value = match
  }
  if (draft.description) {
    description.value = draft.description
  }
  if (draft.date) {
    const d = new Date(draft.date)
    if (!Number.isNaN(d.getTime())) addDate.value = d.toISOString().slice(0, 10)
  }
  toast.flash(t('voice.filled_from_speech'))
}

function onVoiceError(msg: string) {
  toast.flash(msg)
}

async function saveSpending() {
  if (amountNum.value <= 0 || !addGroupId.value) return
  submitting.value = true
  try {
    await spendingApi.create({
      group_id: addGroupId.value,
      amount: amountNum.value,
      currency: addCurrency.value,
      category: category.value.label,
      description: description.value,
      date: new Date(addDate.value).toISOString(),
    })
    showAdd.value = false
    toast.flash(
      `Spending saved · split ${includedList.value.length} way${
        includedList.value.length > 1 ? 's' : ''
      }`,
    )
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Create failed'
  } finally {
    submitting.value = false
  }
}

// ---------- list loading ----------

const currency = computed(() => spendings.value[0]?.currency ?? '$')
const total = computed(() => sumBy(spendings.value, (s) => s.amount))

async function loadGroups() {
  groups.value = (await groupApi.list()) ?? []
  const ids = new Set<string>()
  for (const g of groups.value) {
    ids.add(g.owner_id)
    for (const m of g.member_ids ?? []) ids.add(m)
  }
  const missing = Array.from(ids).filter((id) => !members.value[id])
  const results = await Promise.allSettled(
    missing.map((id) => userApi.getById(id)),
  )
  const next = { ...members.value }
  results.forEach((r, i) => {
    if (r.status === 'fulfilled' && r.value) next[missing[i]] = r.value
  })
  members.value = next
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

async function remove(id: string) {
  if (!confirm('Delete this spending?')) return
  try {
    await spendingApi.remove(id)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

function groupName(id: string): string {
  const g = groups.value.find((x) => String(x._id ?? x.id) === id)
  return g?.name ?? 'Unknown'
}

function memberName(uid: string): string {
  if (uid === myId.value) return 'You'
  return members.value[uid]?.name || 'Someone'
}

function iconFor(cat?: string): IconName {
  const m: Record<string, IconName> = {
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
  const k = (cat ?? '').toLowerCase().trim()
  return m[k] ?? 'wallet'
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

async function loadBudgets() {
  try {
    budgets.value = (await budgetApi.list()) ?? []
  } catch {
    // Tag rendering is non-critical — silently leave the list empty.
  }
}

onMounted(async () => {
  await Promise.all([loadGroups(), loadBudgets()])
  await load()
})

watch([filterGroup, filterCategory], load)
</script>

<template>
  <section>
    <div class="eyebrow">LEDGER · SPENDINGS</div>
    <div class="row spread" style="align-items: flex-start; margin-top: 4px">
      <h1 class="hero">Money <em>out.</em></h1>
      <button
        class="btn btn-accent"
        style="align-self: center"
        :disabled="!groups.length"
        @click="openAdd"
      >
        <Icon name="plus" :size="16" /> New
      </button>
    </div>

    <p v-if="!groups.length" class="muted">
      You need a group first. <router-link to="/groups">Create one</router-link>.
    </p>

    <!-- Stats -->
    <div class="stats">
      <div class="stat">
        <div class="stat-label">TOTAL</div>
        <div class="stat-value">{{ money(total, currency) }}</div>
      </div>
      <div class="stat">
        <div class="stat-label">ENTRIES</div>
        <div class="stat-value">{{ spendings.length }}</div>
      </div>
    </div>

    <!-- Filters -->
    <div class="filters">
      <label class="field">
        <span>GROUP</span>
        <select v-model="filterGroup">
          <option value="">All</option>
          <option
            v-for="g in groups"
            :key="g._id ?? g.id"
            :value="String(g._id ?? g.id)"
          >
            {{ g.name }}
          </option>
        </select>
      </label>
      <label class="field">
        <span>CATEGORY</span>
        <input v-model="filterCategory" placeholder="Filter" />
      </label>
    </div>

    <!-- List -->
    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <template v-else>
      <div v-if="spendings.length" class="ledger">
        <div
          v-for="(s, i) in spendings"
          :key="String(s._id ?? s.id)"
          class="row-entry"
          :class="{ last: i === spendings.length - 1 }"
        >
          <div class="glyph">
            <Icon :name="iconFor(s.category)" :size="18" />
          </div>
          <div style="min-width: 0">
            <div class="title">
              {{ s.description || s.category || 'Spending' }}
            </div>
            <div class="sub">
              {{ when(s.date) }} · {{ memberName(s.user_id) }} · {{ groupName(s.group_id) }}
            </div>
            <div v-if="budgetTagsFor(s).length" class="tag-row">
              <span
                v-for="(tag, ti) in budgetTagsFor(s)"
                :key="ti"
                class="budget-tag"
              >
                <Icon name="gauge" :size="10" />
                {{ tag }}
              </span>
            </div>
          </div>
          <div class="figure">
            <span class="cur">{{ s.currency || '$' }}</span>
            {{ s.amount.toFixed(2) }}
            <button
              class="linklike"
              style="margin-left: 6px; color: var(--ink-ghost)"
              @click="remove(String(s._id ?? s.id))"
              aria-label="Delete"
            >
              <Icon name="close" :size="14" />
            </button>
          </div>
        </div>
      </div>
      <div v-else class="empty">NO SPENDINGS MATCH</div>
    </template>

    <!-- Add-expense modal (3-step) -->
    <Teleport to="body">
      <div v-if="showAdd" class="modal-backdrop" @click.self="closeAdd">
        <div class="modal add-modal">
          <div class="modal-header">
            <button class="linklike" @click="closeAdd">
              <Icon name="close" :size="16" /> CANCEL
            </button>
            <div class="eyebrow">
              STEP {{ step + 1 }} / 3 ·
              <b>{{ activeGroup?.name || 'SELECT GROUP' }}</b>
            </div>
          </div>
          <div class="progress-rail" style="margin: 14px 22px 0">
            <div class="segment" :class="{ on: step >= 0 }"></div>
            <div class="segment" :class="{ on: step >= 1 }"></div>
            <div class="segment" :class="{ on: step >= 2 }"></div>
          </div>

          <!-- Step 0 — amount only -->
          <template v-if="step === 0">
            <div class="modal-body amount-body">
              <div class="row spread" style="align-items: center">
                <div class="eyebrow">NEW SPENDING</div>
                <VoiceInputButton
                  :parser="parseSpendingFromAudio"
                  @result="applySpendingDraft"
                  @error="onVoiceError"
                />
              </div>
              <div class="amount-display">
                <span
                  class="cur"
                  :class="{ dim: !amountStr }"
                >$</span>
                <span
                  class="value"
                  :class="{ dim: !amountStr }"
                >{{ displayAmount() }}</span>
              </div>
              <div class="amount-meta">
                {{ addCurrency }} ·
                <select
                  v-model="addGroupId"
                  class="group-select"
                >
                  <option
                    v-for="g in groups"
                    :key="g._id ?? g.id"
                    :value="String(g._id ?? g.id)"
                  >
                    {{ g.name.toUpperCase() }}
                  </option>
                </select>
              </div>
            </div>

            <div class="numpad">
              <div class="numpad-grid">
                <button
                  v-for="k in ['1','2','3','4','5','6','7','8','9','.','0','⌫']"
                  :key="k"
                  class="numpad-key"
                  type="button"
                  @click="onKey(k)"
                >
                  {{ k }}
                </button>
              </div>
            </div>

            <div class="modal-footer">
              <button
                class="btn btn-primary btn-lg btn-block"
                :disabled="amountNum <= 0 || !addGroupId"
                @click="step = 1"
              >
                Continue <Icon name="arrowRight" :size="16" />
              </button>
            </div>
          </template>

          <!-- Step 1 — who paid + split + description -->
          <template v-if="step === 1">
            <div class="modal-body">
              <h2 class="serif-h">
                Who <em>paid?</em>
              </h2>
              <div class="eyebrow">
                ${{ amountNum.toFixed(2) }}
              </div>

              <div class="payer-list">
                <button
                  v-for="m in groupMembers"
                  :key="String(m._id ?? m.id)"
                  class="payer-row"
                  :class="{ on: payerId === String(m._id ?? m.id) }"
                  @click="payerId = String(m._id ?? m.id)"
                >
                  <Avatar
                    :name="m.name"
                    :color="colorForId(String(m._id ?? m.id))"
                    :size="36"
                  />
                  <span class="lbl">
                    {{ memberName(String(m._id ?? m.id)) === 'You' ? 'You paid' : `${m.name} paid` }}
                  </span>
                  <Icon v-if="payerId === String(m._id ?? m.id)" name="check" :size="18" />
                </button>
              </div>

              <h2 class="serif-h" style="margin-top: 28px">
                Split <em>between.</em>
              </h2>
              <div class="eyebrow">
                {{ includedList.length }} OF {{ groupMembers.length }} · ${{ perPerson.toFixed(2) }} EACH
              </div>

              <div class="split-modes">
                <button
                  v-for="m in [
                    { id: 'equal' as const, label: 'Equal' },
                    { id: 'by_shares' as const, label: 'By shares' },
                    { id: 'custom' as const, label: 'Custom' },
                  ]"
                  :key="m.id"
                  class="mode"
                  :class="{ on: splitMode === m.id }"
                  @click="splitMode = m.id"
                >
                  {{ m.label }}
                </button>
              </div>

              <div class="include-list">
                <button
                  v-for="(m, i) in groupMembers"
                  :key="String(m._id ?? m.id)"
                  class="include-row"
                  :class="{
                    off: !includedIds.has(String(m._id ?? m.id)),
                    last: i === groupMembers.length - 1,
                  }"
                  @click="toggleIncluded(String(m._id ?? m.id))"
                >
                  <Avatar
                    :name="m.name"
                    :color="colorForId(String(m._id ?? m.id))"
                    :size="32"
                  />
                  <span class="name">{{ m.name }}</span>
                  <span class="share">
                    {{
                      includedIds.has(String(m._id ?? m.id))
                        ? '$' + perPerson.toFixed(2)
                        : '—'
                    }}
                  </span>
                  <span
                    class="check"
                    :class="{ on: includedIds.has(String(m._id ?? m.id)) }"
                  >
                    <Icon
                      v-if="includedIds.has(String(m._id ?? m.id))"
                      name="check"
                      :size="14"
                    />
                  </span>
                </button>
              </div>

              <label class="field" style="margin-top: 24px">
                <span>DESCRIPTION</span>
                <input
                  v-model="description"
                  placeholder="e.g. Dinner at Locavore"
                />
              </label>
            </div>

            <div class="modal-footer" style="display: flex; gap: 10px">
              <button class="btn btn-secondary" @click="step = 0">
                <Icon name="arrowLeft" :size="16" />
              </button>
              <button
                class="btn btn-primary btn-lg"
                style="flex: 1"
                :disabled="!includedList.length"
                @click="step = 2"
              >
                Continue <Icon name="arrowRight" :size="16" />
              </button>
            </div>
          </template>

          <!-- Step 2 — date, currency, category, confirm -->
          <template v-if="step === 2">
            <div class="modal-body">
              <h2 class="serif-h">One last <em>detail.</em></h2>

              <div class="split-grid" style="margin-top: 18px">
                <label class="field">
                  <span>DATE</span>
                  <input type="date" v-model="addDate" />
                </label>
                <label class="field">
                  <span>CURRENCY</span>
                  <input v-model="addCurrency" />
                </label>
              </div>

              <div class="eyebrow" style="margin: 22px 0 10px">CATEGORY</div>
              <div class="pill-row">
                <button
                  v-for="c in CATEGORIES"
                  :key="c.id"
                  class="pill"
                  :class="{ on: category.id === c.id }"
                  @click="category = c"
                >
                  <Icon :name="c.id" :size="16" />
                  {{ c.label }}
                </button>
              </div>

              <!-- Summary -->
              <div class="summary">
                <div class="s-label">SUMMARY</div>
                <div class="s-amount">
                  <span class="cur">$</span>{{ amountNum.toFixed(2) }}
                </div>
                <div class="s-rows">
                  <div class="row spread">
                    <span>PAID BY</span
                    ><span>{{ memberName(payerId) }}</span>
                  </div>
                  <div class="row spread">
                    <span>SPLIT</span
                    ><span>
                      {{ includedList.length }} ways · ${{ perPerson.toFixed(2) }} each
                    </span>
                  </div>
                  <div class="row spread">
                    <span>CATEGORY</span><span>{{ category.label }}</span>
                  </div>
                  <div class="row spread">
                    <span>GROUP</span><span>{{ activeGroup?.name }}</span>
                  </div>
                </div>
              </div>
            </div>

            <div class="modal-footer" style="display: flex; gap: 10px">
              <button class="btn btn-secondary" @click="step = 1">
                <Icon name="arrowLeft" :size="16" />
              </button>
              <button
                class="btn btn-accent btn-lg"
                style="flex: 1"
                :disabled="submitting"
                @click="saveSpending"
              >
                <Icon name="check" :size="18" />
                {{ submitting ? 'Saving…' : 'Save spending' }}
              </button>
            </div>
          </template>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<style scoped>
.ledger {
  margin-top: 8px;
}

.add-modal {
  display: flex;
  flex-direction: column;
}

/* Step 0 */
.amount-body {
  padding: 22px 22px 10px;
}
.amount-display {
  display: flex;
  align-items: baseline;
  margin-top: 14px;
}
.amount-display .cur {
  font-family: var(--serif);
  font-size: 40px;
  font-style: italic;
  color: var(--ink);
}
.amount-display .value {
  font-family: var(--serif);
  font-size: 86px;
  font-weight: 400;
  line-height: 1;
  letter-spacing: -0.03em;
  color: var(--ink);
}
.amount-display .dim {
  color: var(--ink-ghost);
}
.amount-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  font-family: var(--mono);
  font-size: 11px;
  color: var(--ink-mute);
  letter-spacing: 0.08em;
  margin-top: 8px;
}
.group-select {
  padding: 4px 6px;
  min-height: 0;
  width: auto;
  background: transparent;
  border: none;
  font-family: var(--mono);
  font-size: 11px;
  color: var(--ink);
  letter-spacing: 0.08em;
  cursor: pointer;
}

/* Shared */
.serif-h {
  font-family: var(--serif);
  font-weight: 400;
  font-size: 32px;
  line-height: 1;
  letter-spacing: -0.01em;
  margin: 8px 0 4px;
}
.serif-h em {
  font-style: italic;
  color: var(--hot);
}

/* Step 1 */
.payer-list {
  margin-top: 18px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.payer-row {
  display: grid;
  grid-template-columns: 36px 1fr 22px;
  gap: 12px;
  align-items: center;
  padding: 12px 14px;
  background: transparent;
  color: var(--ink);
  border: 1px solid var(--line);
  border-radius: var(--r);
  cursor: pointer;
  text-align: left;
}
.payer-row.on {
  background: var(--ink);
  color: var(--paper);
  border-color: var(--ink);
}
.payer-row .lbl {
  font-family: var(--sans);
  font-size: 15px;
  font-weight: 500;
}

.split-modes {
  display: flex;
  gap: 6px;
  margin-top: 14px;
}
.split-modes .mode {
  flex: 1;
  padding: 9px 0;
  background: transparent;
  color: var(--ink-soft);
  border: 1px solid var(--line);
  border-radius: 10px;
  font-family: var(--sans);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
}
.split-modes .mode.on {
  background: var(--ink);
  color: var(--paper);
  border-color: var(--ink);
}

.include-list {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
}
.include-row {
  display: grid;
  grid-template-columns: 32px 1fr auto 22px;
  gap: 12px;
  align-items: center;
  padding: 12px 4px;
  border-bottom: 1px solid var(--line);
  background: none;
  border-left: none;
  border-right: none;
  border-top: none;
  cursor: pointer;
  text-align: left;
}
.include-row.last {
  border-bottom: none;
}
.include-row.off {
  opacity: 0.35;
}
.include-row .name {
  font-family: var(--sans);
  font-size: 14px;
  color: var(--ink);
}
.include-row .share {
  font-family: var(--mono);
  font-size: 12px;
  color: var(--ink);
}
.include-row.off .share {
  color: var(--ink-ghost);
}
.include-row .check {
  width: 20px;
  height: 20px;
  border-radius: 6px;
  border: 1.5px solid var(--line-hard);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--paper);
}
.include-row .check.on {
  background: var(--ink);
  border-color: var(--ink);
}

/* Step 2 */
.split-grid {
  margin-top: 18px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.summary {
  margin-top: 24px;
  background: var(--ink);
  color: var(--paper);
  border-radius: var(--r-lg);
  padding: 18px 20px;
}
.summary .s-label {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.1em;
  color: rgba(245, 241, 232, 0.55);
}
.summary .s-amount {
  font-family: var(--serif);
  font-size: 44px;
  line-height: 1;
  margin-top: 6px;
  letter-spacing: -0.02em;
}
.summary .s-amount .cur {
  font-size: 18px;
  color: rgba(245, 241, 232, 0.55);
  vertical-align: top;
  margin-right: 1px;
}
.summary .s-rows {
  margin-top: 14px;
  display: grid;
  gap: 6px;
  font-family: var(--mono);
  font-size: 11px;
}
.summary .s-rows .row span:first-child {
  color: rgba(245, 241, 232, 0.55);
}

.tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 6px;
}
.budget-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--paper-deep);
  color: var(--ink);
  border: 1px solid var(--line-hard);
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  white-space: nowrap;
}
</style>
