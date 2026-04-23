<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { groupApi, spendingApi, userApi } from '../api/endpoints'
import type { Group, Spending, User } from '../api/types'
import { auth } from '../stores/auth'
import { toast } from '../stores/toast'
import Icon from '../components/Icon.vue'
import Avatar from '../components/Avatar.vue'
import type { IconName } from '../components/icons'
import { colorForId } from '../utils/format'
import { sumBy } from '../utils/charts'

// Whole-dollar money formatters — avoid `.00` cents on mobile so big numbers fit in the card.
function moneyWhole(n: number, cur = '$'): string {
  return (n < 0 ? '−' : '') + cur + Math.abs(Math.round(n)).toLocaleString('en-US')
}
function signedWhole(n: number, cur = '$'): string {
  const sign = n >= 0 ? '+' : '−'
  return sign + cur + Math.abs(Math.round(n)).toLocaleString('en-US')
}

const groups = ref<Group[]>([])
const groupSpendings = ref<Record<string, Spending[]>>({})
const memberCache = ref<Record<string, User>>({})
const loading = ref(true)
const error = ref<string | null>(null)

// Create modal
const showCreate = ref(false)
const createStep = ref(0)
const creating = ref(false)
const newName = ref('')
const newIcon = ref<IconName>('plane')
const newTint = ref('#D64933')
const newMemberName = ref('')
const newMembers = ref<{ name: string; color: string }[]>([])

const ICON_OPTIONS: IconName[] = ['plane', 'bed', 'home2', 'fork', 'ticket', 'gift', 'car', 'cart']
const TINT_OPTIONS = ['#D64933', '#2F5F4F', '#B8915A', '#4A5577', '#8B4A55', '#5B6E4A']

function pickIconFor(g: Group): IconName {
  const name = g.name.toLowerCase()
  if (/(trip|bali|travel|holiday|vacation)/.test(name)) return 'plane'
  if (/(apt|apartment|home|house|flat)/.test(name)) return 'home2'
  if (/(poker|game|sunday)/.test(name)) return 'ticket'
  if (/(food|dinner|brunch)/.test(name)) return 'fork'
  if (/(wedding|gift|party)/.test(name)) return 'gift'
  if (/(coffee|cafe)/.test(name)) return 'coffee'
  return 'people'
}

function tintFor(id: string): string {
  return colorForId(id)
}

async function load() {
  loading.value = true
  error.value = null
  try {
    groups.value = (await groupApi.list()) ?? []
    // preload users
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

    // preload spendings per group for NET / TRACKED stats
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
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

// Per-group figures
function groupTotal(g: Group): number {
  const id = String(g._id ?? g.id)
  return sumBy(groupSpendings.value[id] ?? [], (s) => s.amount)
}

function groupBalance(g: Group): number {
  const id = String(g._id ?? g.id)
  const sp = groupSpendings.value[id] ?? []
  const myId = auth.userIdFromToken()
  if (!myId) return 0
  const total = sumBy(sp, (s) => s.amount)
  const per = (g.member_ids?.length ?? 1) > 0 ? total / (g.member_ids?.length ?? 1) : 0
  const paid = sumBy(
    sp.filter((s) => String(s.user_id) === myId),
    (s) => s.amount,
  )
  return paid - per
}

function groupCurrency(g: Group): string {
  const id = String(g._id ?? g.id)
  return (groupSpendings.value[id] ?? [])[0]?.currency ?? '$'
}

const summary = computed(() => {
  const allSp = Object.values(groupSpendings.value).flat()
  const tracked = sumBy(allSp, (s) => s.amount)
  const now = new Date()
  const thisMonth = sumBy(
    allSp.filter((s) => {
      const d = new Date(s.date)
      return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth()
    }),
    (s) => s.amount,
  )
  const nets = groups.value.reduce((a, g) => a + groupBalance(g), 0)
  const cur = allSp[0]?.currency ?? '$'
  return { tracked, thisMonth, nets, cur }
})

// Create flow
function openCreate() {
  createStep.value = 0
  newName.value = ''
  newIcon.value = 'plane'
  newTint.value = '#D64933'
  newMemberName.value = ''
  newMembers.value = []
  showCreate.value = true
}

function closeCreate() {
  showCreate.value = false
}

function addStagedMember() {
  const val = newMemberName.value.trim()
  if (!val) return
  const palette = ['#2F5F4F', '#B8915A', '#4A5577', '#8B4A55', '#5B6E4A']
  const color = palette[newMembers.value.length % palette.length]
  newMembers.value.push({ name: val, color })
  newMemberName.value = ''
}

function removeStagedMember(i: number) {
  newMembers.value.splice(i, 1)
}

async function createGroup() {
  const uid = auth.userIdFromToken()
  if (!uid) {
    error.value = 'Missing user id'
    return
  }
  creating.value = true
  try {
    await groupApi.create({
      name: newName.value.trim(),
      owner_id: uid,
      member_ids: [uid],
    })
    showCreate.value = false
    toast.flash('Group created · invites sent')
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
    toast.flash('Group deleted')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

const myId = auth.userIdFromToken() ?? ''

onMounted(load)
</script>

<template>
  <section>
    <header class="row spread top-row">
      <div class="eyebrow">
        SIGNED IN · <b>{{ memberCache[myId]?.email || memberCache[myId]?.name || 'you' }}</b>
      </div>
      <button class="icon-btn"><Icon name="search" :size="18" /></button>
    </header>

    <h1 class="hero">Your <em>groups.</em></h1>
    <p class="subtitle">
      {{ groups.length }} active · {{
        Object.values(groupSpendings).reduce((a, b) => a + b.length, 0)
      }} spendings tracked
    </p>

    <!-- Summary strip -->
    <div v-if="groups.length" class="card-ink summary-strip">
      <div>
        <div class="s-label">NET</div>
        <div class="s-val" :style="{ color: summary.nets >= 0 ? 'var(--moss)' : 'var(--hot)' }">
          {{ signedWhole(summary.nets, summary.cur) }}
        </div>
      </div>
      <div>
        <div class="s-label">TRACKED</div>
        <div class="s-val">{{ moneyWhole(summary.tracked, summary.cur) }}</div>
      </div>
      <div>
        <div class="s-label">THIS MO</div>
        <div class="s-val">{{ moneyWhole(summary.thisMonth, summary.cur) }}</div>
      </div>
    </div>

    <div class="row spread section-head">
      <span class="eyebrow">{{ groups.length }} GROUPS</span>
      <button class="linklike accent-link" @click="openCreate">
        <Icon name="plus" :size="12" /> NEW
      </button>
    </div>

    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <template v-else>
      <div v-if="groups.length" class="group-list">
        <div
          v-for="g in groups"
          :key="String(g._id ?? g.id)"
          class="group-card"
        >
          <router-link :to="`/groups/${g._id ?? g.id}`" class="card-body">
            <div
              class="tile"
              :style="{ background: tintFor(String(g._id ?? g.id)) }"
            >
              <Icon :name="pickIconFor(g)" :size="20" />
            </div>
            <div class="body">
              <div class="name">{{ g.name }}</div>
              <div class="sub">
                {{ g.member_ids?.length ?? 0 }} MEMBERS · OWNED BY
                {{ memberCache[g.owner_id]?.name?.toUpperCase() || '—' }}
              </div>
            </div>
            <div class="meta">
              <div
                class="bal"
                :style="{
                  color:
                    groupBalance(g) === 0
                      ? 'var(--ink-mute)'
                      : groupBalance(g) > 0
                      ? 'var(--moss)'
                      : 'var(--hot)',
                }"
              >
                {{
                  groupBalance(g) === 0
                    ? 'SETTLED'
                    : signedWhole(groupBalance(g), groupCurrency(g))
                }}
              </div>
              <div class="total">
                {{ moneyWhole(groupTotal(g), groupCurrency(g)) }} TOTAL
              </div>
            </div>
          </router-link>
          <button
            v-if="g.owner_id === myId"
            class="linklike danger-link"
            @click="remove(String(g._id ?? g.id))"
            aria-label="Delete group"
          >
            <Icon name="close" :size="14" />
          </button>
        </div>
      </div>

      <button v-else class="dashed" @click="openCreate">
        <Icon name="plus" :size="16" /> Create your first group
      </button>

      <button v-if="groups.length" class="dashed" @click="openCreate">
        <Icon name="plus" :size="16" /> Create a new group
      </button>
    </template>

    <div class="foot">↳ gift v0.4.2 · self-hosted</div>

    <!-- Create modal -->
    <Teleport to="body">
      <div v-if="showCreate" class="modal-backdrop" @click.self="closeCreate">
        <div class="modal create-modal">
          <div class="modal-header">
            <button class="linklike" @click="closeCreate">
              <Icon name="close" :size="16" /> CANCEL
            </button>
            <div class="eyebrow">NEW GROUP · {{ createStep + 1 }}/2</div>
          </div>
          <div class="progress-rail" style="margin: 14px 22px 0">
            <div class="segment" :class="{ on: createStep >= 0 }"></div>
            <div class="segment" :class="{ on: createStep >= 1 }"></div>
          </div>

          <!-- Step 0 -->
          <div v-if="createStep === 0" class="modal-body">
            <div class="eyebrow">STEP 1 · NAME IT</div>
            <h1 class="display" style="margin-top: 10px">
              What are you <em>splitting?</em>
            </h1>

            <div class="name-row">
              <div
                class="name-tile"
                :style="{ background: newTint }"
              >
                <Icon :name="newIcon" :size="34" />
              </div>
              <input
                v-model="newName"
                class="serif"
                placeholder="Bali 2026"
              />
            </div>

            <label class="field" style="margin-top: 24px">
              <span>ICON</span>
              <div class="icon-grid">
                <button
                  v-for="id in ICON_OPTIONS"
                  :key="id"
                  type="button"
                  class="icon-tile"
                  :class="{ on: newIcon === id }"
                  @click="newIcon = id"
                >
                  <Icon :name="id" :size="20" />
                </button>
              </div>
            </label>

            <label class="field" style="margin-top: 20px">
              <span>COLOR</span>
              <div class="tint-row">
                <button
                  v-for="c in TINT_OPTIONS"
                  :key="c"
                  type="button"
                  class="tint"
                  :style="{ background: c }"
                  :class="{ on: newTint === c }"
                  @click="newTint = c"
                ></button>
              </div>
            </label>

            <div style="margin-top: 28px">
              <button
                class="btn btn-primary btn-lg btn-block"
                :disabled="!newName.trim()"
                @click="createStep = 1"
              >
                Continue <Icon name="arrowRight" :size="16" />
              </button>
            </div>
          </div>

          <!-- Step 1 -->
          <div v-if="createStep === 1" class="modal-body">
            <div class="eyebrow">STEP 2 · INVITE YOUR PEOPLE</div>
            <h1 class="display" style="margin-top: 10px">
              Who's <em>in?</em>
            </h1>

            <div style="display: flex; gap: 8px; margin-bottom: 18px">
              <input
                v-model="newMemberName"
                placeholder="Name or @username"
                @keydown.enter.prevent="addStagedMember"
              />
              <button
                class="btn btn-primary"
                :disabled="!newMemberName.trim()"
                @click="addStagedMember"
              >
                <Icon name="plus" :size="16" /> Add
              </button>
            </div>

            <div class="eyebrow" style="margin-bottom: 8px">
              {{ newMembers.length + 1 }} MEMBER{{ newMembers.length !== 0 ? 'S' : '' }}
            </div>
            <div>
              <div class="user-row">
                <Avatar name="You" :color="newTint" :size="36" />
                <div class="info">
                  <div class="name">You</div>
                  <div class="email">ADMIN · YOU</div>
                </div>
                <span></span>
              </div>
              <div
                v-for="(m, i) in newMembers"
                :key="i"
                class="user-row"
              >
                <Avatar :name="m.name" :color="m.color" :size="36" />
                <div class="info">
                  <div class="name">{{ m.name }}</div>
                  <div class="email">INVITE PENDING</div>
                </div>
                <button class="icon-btn" @click="removeStagedMember(i)" aria-label="Remove">
                  <Icon name="close" :size="16" />
                </button>
              </div>
            </div>

            <div class="invite-box">
              <div class="eyebrow" style="margin-bottom: 6px">
                OR SHARE INVITE LINK
              </div>
              <div class="row">
                <div class="link-val">gift.local/join/k3f9-xn2</div>
                <button class="btn btn-primary btn-sm">COPY</button>
              </div>
            </div>

            <div style="margin-top: 28px; display: flex; gap: 10px">
              <button class="btn btn-secondary" @click="createStep = 0">
                <Icon name="arrowLeft" :size="16" />
              </button>
              <button
                class="btn btn-accent btn-lg"
                style="flex: 1"
                :disabled="creating || !newName.trim()"
                @click="createGroup"
              >
                <Icon name="check" :size="18" />
                {{ creating ? 'Creating…' : 'Create group' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<style scoped>
.top-row {
  margin-bottom: 22px;
}
.subtitle {
  font-family: var(--sans);
  font-size: 13px;
  color: var(--ink-soft);
  margin: 6px 0 0;
}

.summary-strip {
  margin-top: 22px;
  padding: clamp(14px, 4.2vw, 16px) clamp(14px, 4.5vw, 18px);
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: clamp(8px, 3vw, 14px);
  border-radius: var(--r-lg);
  overflow: hidden;
}
.summary-strip > div {
  min-width: 0;
}
.summary-strip > div + div {
  border-left: 1px solid rgba(245, 241, 232, 0.12);
  padding-left: clamp(8px, 3vw, 14px);
}
.summary-strip .s-label {
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.1em;
  color: rgba(245, 241, 232, 0.5);
}
.summary-strip .s-val {
  font-family: var(--serif);
  /* Scales from 15px on iPhone SE up to 22px on tablet+ */
  font-size: clamp(15px, 5.5vw, 22px);
  margin-top: 2px;
  color: var(--paper);
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: clip;
}

.section-head {
  margin-top: 26px;
  margin-bottom: 10px;
}
.accent-link {
  color: var(--hot);
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.accent-link:hover {
  color: var(--accent-hover);
}
.danger-link {
  color: var(--ink-ghost);
}
.danger-link:hover {
  color: var(--hot);
}

.group-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 18px;
}
.group-card {
  position: relative;
}
.group-card .card-body {
  background: var(--paper);
  border: 1px solid var(--line);
  border-radius: var(--r-lg);
  padding: clamp(14px, 4vw, 16px) clamp(14px, 4.5vw, 18px);
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr) auto;
  gap: clamp(10px, 3vw, 14px);
  align-items: center;
  text-decoration: none;
  color: inherit;
  transition: border-color 0.15s;
}
.group-card .card-body:hover {
  border-color: var(--ink);
}
.group-card .tile {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  color: var(--paper);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.group-card .body {
  min-width: 0;
}
.group-card .body .name {
  font-family: var(--serif);
  font-size: clamp(17px, 5.5vw, 22px);
  line-height: 1.1;
  color: var(--ink);
  letter-spacing: -0.01em;
  /* Long group names truncate cleanly instead of pushing the balance off-card */
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.group-card .body .sub {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  color: var(--ink-mute);
  margin-top: 3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.group-card .meta {
  text-align: right;
  min-width: 0;
  max-width: 40%;
}
.group-card .meta .bal {
  font-family: var(--mono);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.group-card .meta .total {
  font-family: var(--mono);
  font-size: 9px;
  color: var(--ink-ghost);
  margin-top: 3px;
  letter-spacing: 0.06em;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.group-card > .linklike {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 4px;
}

/* Create modal */
.name-row {
  display: flex;
  gap: 14px;
  align-items: center;
  margin-top: 4px;
}
.name-tile {
  width: 72px;
  height: 72px;
  border-radius: 20px;
  color: var(--paper);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.icon-grid {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 4px;
}
.icon-tile {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  background: var(--paper-deep);
  color: var(--ink-soft);
  border: 1px solid var(--line);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.15s;
}
.icon-tile.on {
  background: var(--ink);
  color: var(--paper);
  border-color: var(--ink);
}
.tint-row {
  display: flex;
  gap: 10px;
  margin-top: 4px;
}
.tint {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: none;
  cursor: pointer;
}
.tint.on {
  box-shadow: 0 0 0 2px var(--paper), 0 0 0 3.5px var(--ink);
}

.invite-box {
  margin-top: 24px;
  padding: 14px 16px;
  background: var(--paper-deep);
  border: 1px dashed var(--line-hard);
  border-radius: var(--r);
}
.invite-box .link-val {
  flex: 1;
  font-family: var(--mono);
  font-size: 12px;
  color: var(--ink);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
