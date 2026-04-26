<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { budgetApi, groupApi, spendingApi, userApi } from '../api/endpoints'
import type { Budget, Group, Spending, User } from '../api/types'
import { auth } from '../stores/auth'
import { toast } from '../stores/toast'
import Icon from '../components/Icon.vue'
import Avatar from '../components/Avatar.vue'
import Sparkline from '../components/Sparkline.vue'
import VoiceInputButton from '../components/VoiceInputButton.vue'
import type { IconName } from '../components/icons'
import { colorForId, money, signed } from '../utils/format'
import { formatDay, groupBy, lastNDays, sumBy } from '../utils/charts'
import { parseSpendingFromAudio, type SpendingDraft } from '../ai/parse'
import { t } from '../i18n'

const props = defineProps<{ id: string }>()
const router = useRouter()

const group = ref<Group | null>(null)
const spendings = ref<Spending[]>([])
const members = ref<Record<string, User>>({})
const loading = ref(true)
const error = ref<string | null>(null)
const busy = ref(false)

const myId = computed(() => auth.userIdFromToken() ?? '')
const isOwner = computed(
  () => !!group.value && group.value.owner_id === myId.value,
)

// --- member modal ---------------------------------------------------------
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
  if (!confirm(t('group_detail.confirm_remove_member'))) return
  busy.value = true
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
  if (uid === myId.value) return t('group_detail.you')
  return members.value[uid]?.name || t('group_detail.someone')
}

// --- rename --------------------------------------------------------------
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
  try {
    const g = await groupApi.update(props.id, { name: renameValue.value.trim() })
    group.value = g
    renaming.value = false
    toast.flash(t('group_detail.toast_renamed'))
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Rename failed'
  } finally {
    busy.value = false
  }
}

// --- add spending --------------------------------------------------------
const showSpendingForm = ref(false)
const spendingForm = ref({
  amount: 0,
  currency: 'USD',
  category: '',
  description: '',
  date: new Date().toISOString().slice(0, 10),
})
const creatingSpending = ref(false)

function applySpendingVoiceDraft(draft: SpendingDraft) {
  // The form is open from the moment the user taps the voice button, so we
  // just merge in whatever the model captured. Anything left null stays at
  // the user's previous value.
  if (draft.amount != null && draft.amount > 0) {
    spendingForm.value.amount = draft.amount
  }
  if (draft.currency) {
    spendingForm.value.currency = draft.currency.toUpperCase()
  }
  if (draft.category) {
    spendingForm.value.category = draft.category
  }
  if (draft.description) {
    spendingForm.value.description = draft.description
  }
  if (draft.date) {
    const d = new Date(draft.date)
    if (!Number.isNaN(d.getTime())) {
      spendingForm.value.date = d.toISOString().slice(0, 10)
    }
  }
  showSpendingForm.value = true
  toast.flash(t('voice.filled_from_speech'))
}

function onSpendingVoiceError(msg: string) {
  toast.flash(msg)
}

async function createSpending() {
  creatingSpending.value = true
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
    toast.flash(t('group_detail.toast_saved'))
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Create failed'
  } finally {
    creatingSpending.value = false
  }
}

async function removeSpending(id: string) {
  if (!confirm(t('group_detail.confirm_delete_spending'))) return
  try {
    await spendingApi.remove(id)
    spendings.value = spendings.value.filter((s) => String(s._id ?? s.id) !== id)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

// --- swipe-to-link-budget ------------------------------------------------
const SWIPE_OPEN_PX = 96 // matches the action button width
const SWIPE_TRIGGER_PX = 48 // halfway → snap open

const offsets = ref<Record<string, number>>({})
const openSwipeId = ref<string | null>(null)
const drag = ref<{
  id: string
  startX: number
  startY: number
  startOffset: number
  active: boolean
} | null>(null)

function rowOffset(id: string): number {
  return offsets.value[id] ?? 0
}

function setOffset(id: string, value: number) {
  offsets.value = { ...offsets.value, [id]: value }
}

function closeSwipe() {
  if (openSwipeId.value) {
    setOffset(openSwipeId.value, 0)
    openSwipeId.value = null
  }
}

function onSwipeDown(id: string, e: PointerEvent) {
  if (e.pointerType === 'mouse' && e.button !== 0) return
  drag.value = {
    id,
    startX: e.clientX,
    startY: e.clientY,
    startOffset: rowOffset(id),
    active: false,
  }
}

function onSwipeMove(id: string, e: PointerEvent) {
  const d = drag.value
  if (!d || d.id !== id) return
  const dx = e.clientX - d.startX
  const dy = e.clientY - d.startY
  if (!d.active) {
    // Lock direction once horizontal motion clearly dominates.
    if (Math.abs(dx) < 8 || Math.abs(dx) < Math.abs(dy)) return
    d.active = true
    ;(e.target as Element)?.setPointerCapture?.(e.pointerId)
  }
  const next = Math.max(-SWIPE_OPEN_PX, Math.min(0, d.startOffset + dx))
  setOffset(id, next)
}

function onSwipeUp(id: string, e: PointerEvent) {
  const d = drag.value
  drag.value = null
  if (!d || d.id !== id) return
  ;(e.target as Element)?.releasePointerCapture?.(e.pointerId)
  if (!d.active) return
  const open = Math.abs(rowOffset(id)) >= SWIPE_TRIGGER_PX
  setOffset(id, open ? -SWIPE_OPEN_PX : 0)
  if (open) {
    if (openSwipeId.value && openSwipeId.value !== id) {
      setOffset(openSwipeId.value, 0)
    }
    openSwipeId.value = id
  } else if (openSwipeId.value === id) {
    openSwipeId.value = null
  }
}

// --- budget picker -------------------------------------------------------
const budgets = ref<Budget[]>([])
const linkingFor = ref<Spending | null>(null)
const linkingBusy = ref(false)

async function loadBudgets() {
  try {
    budgets.value = (await budgetApi.list()) ?? []
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load budgets'
  }
}

async function openLinkBudget(s: Spending) {
  if (!budgets.value.length) await loadBudgets()
  linkingFor.value = s
  closeSwipe()
}

function closeLinkBudget() {
  linkingFor.value = null
  linkingBusy.value = false
}

function isLinked(s: Spending | null, b: Budget): boolean {
  if (!s) return false
  const bid = String(b.id ?? '')
  return !!bid && (s.budgets ?? []).includes(bid)
}

async function toggleBudgetLink(b: Budget) {
  if (!linkingFor.value) return
  const sid = String(linkingFor.value._id ?? linkingFor.value.id)
  const bid = String(b.id)
  if (!sid || !bid) return
  const linked = isLinked(linkingFor.value, b)
  linkingBusy.value = true
  try {
    if (linked) {
      await spendingApi.unlinkBudget(sid, bid)
      // Mirror the change locally so the picker badge updates immediately and
      // the row's "linked" chip reflects the unlink without a refetch.
      mutateBudgets(sid, (ids) => ids.filter((x) => x !== bid))
      toast.flash(t('group_detail.toast_unlinked', { name: b.category }))
    } else {
      await spendingApi.linkBudget(sid, bid)
      mutateBudgets(sid, (ids) => (ids.includes(bid) ? ids : [...ids, bid]))
      toast.flash(t('group_detail.toast_linked', { name: b.category }))
    }
    closeLinkBudget()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Link failed'
    linkingBusy.value = false
  }
}

function mutateBudgets(spendingId: string, fn: (ids: string[]) => string[]) {
  spendings.value = spendings.value.map((s) => {
    if (String(s._id ?? s.id) !== spendingId) return s
    const next = fn(s.budgets ?? [])
    return { ...s, budgets: next }
  })
  if (linkingFor.value && String(linkingFor.value._id ?? linkingFor.value.id) === spendingId) {
    linkingFor.value = {
      ...linkingFor.value,
      budgets: fn(linkingFor.value.budgets ?? []),
    }
  }
}

// --- derived -------------------------------------------------------------
const currency = computed(() => spendings.value[0]?.currency ?? '$')
const total = computed(() => sumBy(spendings.value, (s) => s.amount))

const todaySpent = computed(() => {
  const now = new Date().toDateString()
  return sumBy(
    spendings.value.filter((s) => new Date(s.date).toDateString() === now),
    (s) => s.amount,
  )
})
const txToday = computed(
  () =>
    spendings.value.filter(
      (s) => new Date(s.date).toDateString() === new Date().toDateString(),
    ).length,
)

const spark = computed(() => {
  const days = lastNDays(10)
  const byDay = groupBy(spendings.value, (s) => formatDay(new Date(s.date)))
  return days.map((d) => sumBy(byDay.get(d) ?? [], (s) => s.amount))
})

const memberList = computed<User[]>(() => {
  if (!group.value) return []
  const ids = Array.from(
    new Set([group.value.owner_id, ...(group.value.member_ids ?? [])]),
  )
  return ids.map((id) => members.value[id]).filter(Boolean) as User[]
})

interface Balance {
  user: User
  net: number
}

const balances = computed<Balance[]>(() => {
  if (!memberList.value.length) return []
  const per = memberList.value.length ? total.value / memberList.value.length : 0
  return memberList.value.map((u) => {
    const uid = String(u._id ?? u.id)
    const paid = sumBy(
      spendings.value.filter((s) => String(s.user_id) === uid),
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
  if (diff === 0) return t('group_detail.when_today', { time: d.toTimeString().slice(0, 5) })
  if (diff === 1) return t('group_detail.when_yesterday')
  if (diff < 7) return d.toLocaleDateString(undefined, { weekday: 'short' }).toUpperCase()
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }).toUpperCase()
}

const heroTitle = computed(() => {
  const g = group.value
  if (!g) return { first: '', second: '' }
  const parts = g.name.trim().split(/\s+/)
  if (parts.length === 1) return { first: parts[0], second: '' }
  return { first: parts[0], second: parts.slice(1).join(' ') }
})

async function load() {
  loading.value = true
  error.value = null
  try {
    const g = await groupApi.get(props.id)
    group.value = g
    const [sp] = await Promise.all([
      spendingApi.query({ group_id: props.id }),
      g ? loadMembers([g.owner_id, ...(g.member_ids ?? [])]) : Promise.resolve(),
      loadBudgets(),
    ])
    spendings.value = sp ?? []
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

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

async function deleteGroup() {
  if (!confirm(t('group_detail.confirm_delete_group'))) return
  try {
    await groupApi.remove(props.id)
    router.push('/groups')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

onMounted(load)
watch(() => props.id, load)
</script>

<template>
  <section>
    <router-link to="/groups" class="back">
      <Icon name="arrowLeft" :size="14" /> {{ t('group_detail.back') }}
    </router-link>

    <p v-if="loading" class="muted">{{ t('common.loading') }}</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <template v-else-if="group">
      <div class="eyebrow">{{ t('group_detail.group_active') }}</div>
      <div v-if="renaming" class="inline-form" style="margin: 10px 0">
        <input
          v-model="renameValue"
          :disabled="busy"
          autofocus
          @keyup.enter="saveRename"
          @keyup.esc="renaming = false"
        />
        <button class="btn btn-primary" @click="saveRename" :disabled="busy">{{ t('common.save') }}</button>
      </div>
      <div v-else class="hero-row">
        <h1 class="hero">
          {{ heroTitle.first }}<br />
          <em>{{ heroTitle.second || '.' }}</em>
        </h1>
        <div class="avatar-stack" v-if="memberList.length">
          <Avatar
            v-for="m in memberList.slice(0, 4)"
            :key="String(m._id ?? m.id)"
            :name="m.name"
            :color="colorForId(String(m._id ?? m.id))"
            :size="30"
            ring
          />
        </div>
      </div>
      <div class="group-meta">
        <span>{{ t('group_detail.member_count', { n: group.member_ids?.length ?? 0 }) }}</span>
        <span>·</span>
        <span>{{ t('group_detail.spending_count', { n: spendings.length }) }}</span>
        <button v-if="isOwner" class="linklike" @click="startRename">{{ t('group_detail.rename') }}</button>
        <button class="linklike" @click="openMembers">{{ t('group_detail.members') }}</button>
      </div>

      <!-- Total card -->
      <div class="card-ink total-card">
        <div class="mono-label">{{ t('group_detail.total_spent') }}</div>
        <div class="money-big">
          <span class="cur">{{ currency }}</span>
          {{ Math.floor(total).toLocaleString()
          }}<span class="decimals">.{{ (total % 1).toFixed(2).slice(2) }}</span>
        </div>
        <div class="row spread meta">
          <span>
            <span class="dot"></span>
            {{ t('group_detail.today_txns', { amount: money(todaySpent, currency), n: txToday }) }}
          </span>
          <span>{{ t('group_detail.ten_days') }}</span>
        </div>
        <div class="spark">
          <Sparkline :data="spark" :width="314" :height="48" stroke="#F5F1E8" />
        </div>
      </div>

      <!-- Quick add -->
      <div style="display: flex; gap: 10px; margin-top: 18px">
        <button
          class="btn btn-accent btn-lg"
          style="flex: 1"
          @click="showSpendingForm = !showSpendingForm"
        >
          <Icon name="plus" :size="18" />
          {{ showSpendingForm ? t('group_detail.cancel') : t('group_detail.spending') }}
        </button>
        <router-link to="/incomes" class="btn btn-secondary btn-lg" style="flex: 1">
          <Icon name="arrowDown" :size="18" /> {{ t('group_detail.income') }}
        </router-link>
      </div>

      <form
        v-if="showSpendingForm"
        class="stack-form"
        @submit.prevent="createSpending"
      >
        <div class="row spread" style="align-items: center">
          <div class="eyebrow">{{ t('group_detail.new_spending') }}</div>
          <VoiceInputButton
            :parser="parseSpendingFromAudio"
            @result="applySpendingVoiceDraft"
            @error="onSpendingVoiceError"
          />
        </div>
        <div class="split">
          <label class="field">
            <span>{{ t('group_detail.amount_label') }}</span>
            <input
              v-model.number="spendingForm.amount"
              type="number"
              min="0"
              step="0.01"
              required
            />
          </label>
          <label class="field">
            <span>{{ t('group_detail.currency_label') }}</span>
            <input v-model="spendingForm.currency" />
          </label>
        </div>
        <label class="field">
          <span>{{ t('group_detail.category_label') }}</span>
          <input v-model="spendingForm.category" :placeholder="t('group_detail.category_placeholder')" />
        </label>
        <label class="field">
          <span>{{ t('group_detail.description_label') }}</span>
          <input v-model="spendingForm.description" />
        </label>
        <label class="field">
          <span>{{ t('group_detail.date_label') }}</span>
          <input v-model="spendingForm.date" type="date" required />
        </label>
        <button
          type="submit"
          class="btn btn-primary btn-lg"
          :disabled="creatingSpending"
        >
          {{ creatingSpending ? t('group_detail.saving') : t('group_detail.save_spending') }}
          <Icon name="check" :size="16" />
        </button>
      </form>

      <!-- Balances -->
      <div v-if="balances.length" class="section-block">
        <div class="row spread">
          <span class="eyebrow">{{ t('group_detail.balances') }}</span>
          <span class="eyebrow">{{ t('group_detail.settle_up') }}</span>
        </div>
        <h3 class="serif">{{ t('group_detail.who_is_up') }} <em>{{ t('group_detail.up') }}</em></h3>
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
                width: (Math.abs(b.net) / balanceMax) * 50 + '%',
                background: b.net >= 0 ? 'var(--moss)' : 'var(--hot)',
              }"
            ></div>
          </div>
          <div
            class="figure"
            :style="{ color: b.net >= 0 ? 'var(--moss)' : 'var(--hot)' }"
          >
            {{ signed(b.net, currency) }}
          </div>
        </div>
      </div>

      <!-- Spendings -->
      <div class="section-block">
        <div class="row spread">
          <span class="eyebrow">{{ t('group_detail.ledger') }}</span>
          <span class="eyebrow">{{ t('group_detail.entries', { n: spendings.length }) }}</span>
        </div>
        <h3 class="serif">{{ t('group_detail.recent') }} <em>{{ t('group_detail.spendings_em') }}</em></h3>
        <div v-if="!spendings.length" class="empty">{{ t('group_detail.no_spendings') }}</div>
        <div v-else>
          <div
            v-for="(s, i) in spendings"
            :key="String(s._id ?? s.id)"
            class="swipe-row"
            :class="{ last: i === spendings.length - 1 }"
          >
            <button
              class="swipe-action"
              type="button"
              @click="openLinkBudget(s)"
            >
              <Icon name="gauge" :size="16" />
              <span>{{ t('group_detail.budget_action') }}</span>
            </button>
            <div
              class="row-entry swipe-content"
              :style="{ transform: `translateX(${rowOffset(String(s._id ?? s.id))}px)` }"
              @pointerdown="onSwipeDown(String(s._id ?? s.id), $event)"
              @pointermove="onSwipeMove(String(s._id ?? s.id), $event)"
              @pointerup="onSwipeUp(String(s._id ?? s.id), $event)"
              @pointercancel="onSwipeUp(String(s._id ?? s.id), $event)"
            >
              <div class="glyph"><Icon :name="iconFor(s.category)" :size="18" /></div>
              <div style="min-width: 0">
                <div class="title">
                  {{ s.description || s.category || t('group_detail.spending_fallback') }}
                </div>
                <div class="sub">
                  {{ when(s.date) }} · {{ memberName(s.user_id) }} {{ t('group_detail.paid') }}
                </div>
                <div v-if="budgetTagsFor(s).length" class="tag-row">
                  <span
                    v-for="(t, ti) in budgetTagsFor(s)"
                    :key="ti"
                    class="budget-tag"
                  >
                    <Icon name="gauge" :size="10" />
                    {{ t }}
                  </span>
                </div>
              </div>
              <div class="figure">
                <span class="cur">{{ s.currency || '$' }}</span>
                {{ s.amount.toFixed(2) }}
                <button
                  v-if="s.user_id === myId"
                  class="linklike"
                  style="margin-left: 4px; color: var(--ink-ghost)"
                  :title="'Delete'"
                  @click.stop="removeSpending(String(s._id ?? s.id))"
                >
                  <Icon name="close" :size="14" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="isOwner" class="danger-zone">
        <div class="eyebrow" style="color: var(--hot); margin-bottom: 8px">
          {{ t('group_detail.danger_zone') }}
        </div>
        <button class="btn btn-danger" @click="deleteGroup">
          <Icon name="close" :size="14" /> {{ t('group_detail.delete_group') }}
        </button>
      </div>
    </template>

    <!-- Budget picker modal -->
    <Teleport to="body">
      <div
        v-if="linkingFor"
        class="modal-backdrop"
        @click.self="closeLinkBudget"
      >
        <div class="modal">
          <div class="modal-header">
            <button class="linklike" @click="closeLinkBudget">
              <Icon name="close" :size="16" /> {{ t('common.cancel').toUpperCase() }}
            </button>
            <div class="eyebrow">{{ t('group_detail.link_budget') }}</div>
          </div>
          <div class="modal-body">
            <h1 class="display">
              {{ t('group_detail.pin_to') }} <em>{{ t('group_detail.a_budget') }}</em>
            </h1>
            <p class="muted small" style="margin-top: 6px">
              {{ linkingFor.description || linkingFor.category || t('group_detail.spending_fallback') }}
              · {{ linkingFor.currency || '$' }}{{ linkingFor.amount.toFixed(2) }}
            </p>

            <div v-if="!budgets.length" class="empty" style="margin-top: 18px">
              {{ t('group_detail.no_budgets_yet') }}
            </div>
            <div v-else class="picker-list" style="margin-top: 18px">
              <button
                v-for="b in budgets"
                :key="String(b.id)"
                class="picker-row"
                :class="{ on: isLinked(linkingFor, b) }"
                :disabled="linkingBusy"
                @click="toggleBudgetLink(b)"
              >
                <span class="picker-symbol">
                  <Icon :name="iconFor(b.category)" :size="18" />
                </span>
                <span class="picker-label">
                  {{ b.category }}
                  <span class="muted small" style="display: block; margin-top: 2px">
                    {{ b.currency || '' }} {{ b.amount }} / {{ b.limit }} · {{ b.period }}
                  </span>
                </span>
                <span v-if="isLinked(linkingFor, b)" class="badge linked-badge">
                  {{ t('group_detail.linked_tap_to_unlink') }}
                </span>
                <Icon v-else name="chevR" :size="14" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Members modal -->
    <Teleport to="body">
      <div v-if="showMembers && group" class="modal-backdrop" @click.self="closeMembers">
        <div class="modal">
          <div class="modal-header">
            <button class="linklike" @click="closeMembers">
              <Icon name="close" :size="16" /> {{ t('group_detail.close') }}
            </button>
            <div class="eyebrow">{{ t('group_detail.members_of', { name: group.name }) }}</div>
          </div>
          <div class="modal-body">
            <p v-if="modalError" class="error">{{ modalError }}</p>

            <div v-if="isOwner">
              <input
                v-model="searchQuery"
                :placeholder="t('group_detail.search_people')"
                @input="onSearchInput"
              />
              <div v-if="searching" class="muted small" style="margin-top: 8px">
                {{ t('group_detail.searching') }}
              </div>
              <div
                v-else-if="searchQuery && !searchResults.length"
                class="muted small"
                style="margin-top: 8px"
              >
                {{ t('group_detail.no_matches') }}
              </div>
              <div v-else-if="searchResults.length" style="margin-top: 12px">
                <div
                  v-for="u in searchResults"
                  :key="userId(u)"
                  class="user-row"
                >
                  <Avatar :name="u.name" :color="colorForId(userId(u))" :size="36" />
                  <div class="info">
                    <div class="name">{{ u.name }}</div>
                    <div class="email">{{ u.email }}</div>
                  </div>
                  <button
                    v-if="group.member_ids?.includes(userId(u))"
                    class="linklike"
                    disabled
                  >
                    {{ t('group_detail.added') }}
                  </button>
                  <button
                    v-else
                    class="btn btn-primary btn-sm"
                    :disabled="busy"
                    @click="addMember(u)"
                  >
                    {{ t('group_detail.add') }}
                  </button>
                </div>
              </div>
            </div>

            <div class="eyebrow" style="margin-top: 22px">
              {{ t('group_detail.current_count', { n: group.member_ids?.length ?? 0 }) }}
            </div>
            <div
              v-for="m in group.member_ids"
              :key="m"
              class="user-row"
            >
              <Avatar
                :name="memberName(m)"
                :color="colorForId(m)"
                :size="36"
              />
              <div class="info">
                <div class="name">
                  {{ memberName(m) }}
                  <span v-if="m === group.owner_id" class="badge">{{ t('group_detail.owner') }}</span>
                  <span v-if="m === myId" class="badge">{{ t('group_detail.you_badge') }}</span>
                </div>
                <div class="email">{{ members[m]?.email || '' }}</div>
              </div>
              <button
                v-if="m !== group.owner_id && isOwner"
                class="btn btn-danger btn-sm"
                :disabled="busy"
                @click="removeMember(m)"
              >
                {{ t('group_detail.remove') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<style scoped>
.hero-row {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
  margin-top: 8px;
}

.group-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  font-family: var(--mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  color: var(--ink-mute);
  margin-top: 10px;
}

.total-card {
  margin-top: 22px;
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

.section-block {
  margin-top: 30px;
}

.danger-zone {
  margin-top: 36px;
  padding-top: 16px;
  border-top: 1px solid var(--line);
}

/* Swipe-to-link rows */
.swipe-row {
  position: relative;
  overflow: hidden;
  border-bottom: 1px solid var(--line);
}
.swipe-row.last {
  border-bottom: none;
}
.swipe-row .swipe-content {
  background: var(--paper);
  /* Cancel the bottom border applied by the global .row-entry rule —
   * the wrapper draws it for us so it stays put while the row slides. */
  border-bottom: none !important;
  transition: transform 0.18s cubic-bezier(0.22, 0.9, 0.28, 1);
  touch-action: pan-y;
  user-select: none;
  cursor: grab;
}
.swipe-row .swipe-content:active {
  cursor: grabbing;
}
.swipe-row .swipe-action {
  position: absolute;
  top: 0;
  bottom: 0;
  right: 0;
  width: 96px;
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  background: var(--ink);
  color: var(--paper);
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  border: none;
  cursor: pointer;
}
.swipe-row .swipe-action:hover {
  background: #000;
}

/* Local copy of the picker-list rules used by Settings, so the budget picker
 * looks consistent. Kept scoped to this view. */
.picker-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.picker-row {
  display: grid;
  grid-template-columns: 36px 1fr 22px;
  gap: 14px;
  align-items: center;
  padding: 14px 18px;
  background: var(--paper);
  border: 1px solid var(--line);
  border-radius: var(--r);
  cursor: pointer;
  text-align: left;
  color: var(--ink);
  transition: border-color 0.15s, background 0.15s;
}
.picker-row:hover:not(:disabled) {
  border-color: var(--ink);
}
.picker-row:disabled {
  opacity: 0.6;
  cursor: wait;
}
.picker-symbol {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--ink);
}
.picker-label {
  font-family: var(--sans);
  font-size: 15px;
  font-weight: 500;
}

/* Highlight already-linked picker rows */
.picker-row.on {
  border-color: var(--ink);
  background: var(--paper-deep);
}

.linked-badge {
  background: var(--ink);
  color: var(--paper);
  border-radius: 999px;
  padding: 4px 8px;
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.08em;
  white-space: nowrap;
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
