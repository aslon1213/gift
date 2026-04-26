<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import Icon from './Icon.vue'
import type { Credit, FinanceRequest } from '../api/types'
import { borrowingApi, lendingApi } from '../api/endpoints'
import { auth } from '../stores/auth'
import { toast } from '../stores/toast'
import { moneyWhole } from '../utils/format'
import { t } from '../i18n'

interface PendingItem {
  request: FinanceRequest
  credit: Credit
  side: 'borrowing' | 'lending'
}

const props = defineProps<{
  // Optional pre-loaded credit lists, so the home page doesn't have to refetch
  // when it already has them in memory. When omitted, the sheet loads them.
  borrowed?: Credit[]
  lent?: Credit[]
  open: boolean
}>()

const emit = defineEmits<{
  close: []
  // Bubbled when a request decision succeeds — lets the parent refresh its
  // own list without us coupling to its data store.
  'request-resolved': [credit: Credit, side: 'borrowing' | 'lending']
}>()

const router = useRouter()
const myId = computed(() => auth.userIdFromToken() ?? '')

// Local mirrors so optimistic updates after approve/reject don't need a parent
// re-render cycle. Seeded from the props on open and refilled on each fetch.
const localBorrowed = ref<Credit[]>([])
const localLent = ref<Credit[]>([])
const loading = ref(false)
const acting = ref<string | null>(null)

async function loadIfNeeded() {
  if (props.borrowed && props.lent) {
    localBorrowed.value = props.borrowed
    localLent.value = props.lent
    return
  }
  loading.value = true
  try {
    const [b, l] = await Promise.all([borrowingApi.list(), lendingApi.list()])
    localBorrowed.value = b ?? []
    localLent.value = l ?? []
  } finally {
    loading.value = false
  }
}

watch(
  () => props.open,
  (now) => {
    if (now) loadIfNeeded()
  },
  { immediate: true },
)

// Whenever the parent's lists change while we're open, mirror the change so
// counts stay live (e.g. user resolved a request from the detail screen).
watch(
  () => [props.borrowed, props.lent],
  () => {
    if (props.borrowed) localBorrowed.value = props.borrowed
    if (props.lent) localLent.value = props.lent
  },
)

// Pending requests where the *other side* opened the request — so I'm the
// reviewer. Requests I opened myself are surfaced as "awaiting them" elsewhere
// (on the credit detail screen) and don't belong in the inbox.
const pending = computed<PendingItem[]>(() => {
  const items: PendingItem[] = []
  for (const c of localBorrowed.value) {
    for (const r of c.finance_requests ?? []) {
      if (r.status === 'pending' && r.requested_by !== myId.value) {
        items.push({ request: r, credit: c, side: 'borrowing' })
      }
    }
  }
  for (const c of localLent.value) {
    for (const r of c.finance_requests ?? []) {
      if (r.status === 'pending' && r.requested_by !== myId.value) {
        items.push({ request: r, credit: c, side: 'lending' })
      }
    }
  }
  // Newest first — clients don't always sort, so do it here.
  items.sort((a, b) => {
    const ad = new Date(a.request.created_at ?? 0).getTime()
    const bd = new Date(b.request.created_at ?? 0).getTime()
    return bd - ad
  })
  return items
})

function counterpartyName(c: Credit, side: 'borrowing' | 'lending'): string {
  const cp = side === 'borrowing' ? c.from : c.to
  if (!cp) return '—'
  if (cp.is_oid) return cp.oid.slice(-6)
  return cp.str || '—'
}

function creditId(c: Credit): string {
  return String(c._id ?? c.id ?? '')
}

async function decide(item: PendingItem, action: 'approve' | 'reject') {
  const cid = creditId(item.credit)
  if (!cid) return
  const key = `${cid}:${item.request.id}`
  acting.value = key
  try {
    const api = item.side === 'borrowing' ? borrowingApi : lendingApi
    const fn = action === 'approve' ? api.approveRequest : api.rejectRequest
    const updated = await fn(cid, item.request.id)
    // Mirror locally so the row disappears from the inbox immediately.
    const list = item.side === 'borrowing' ? localBorrowed : localLent
    list.value = list.value.map((c) => (creditId(c) === cid ? updated : c))
    emit('request-resolved', updated, item.side)
    toast.flash(action === 'approve' ? t('ledger_detail.approved_toast') : t('ledger_detail.rejected_toast'))
  } catch (e) {
    toast.flash(e instanceof Error ? e.message : 'Decision failed')
  } finally {
    acting.value = null
  }
}

function openRecord(item: PendingItem) {
  emit('close')
  router.push(`/${item.side === 'borrowing' ? 'borrowings' : 'lendings'}/${creditId(item.credit)}`)
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="modal-backdrop" @click.self="emit('close')">
      <div class="modal">
        <div class="modal-header">
          <button class="linklike" @click="emit('close')">
            <Icon name="close" :size="16" /> {{ t('common.cancel').toUpperCase() }}
          </button>
          <div class="eyebrow">
            {{ pending.length
              ? t('notifications.eyebrow_pending', { n: pending.length })
              : t('notifications.eyebrow_empty') }}
          </div>
        </div>
        <div class="modal-body">
          <h1 class="display" style="margin: 0">{{ t('notifications.title') }}</h1>

          <p v-if="loading" class="muted" style="margin-top: 14px">{{ t('common.loading') }}</p>

          <div v-else-if="!pending.length" class="empty-state">
            <div class="empty-title">{{ t('notifications.empty_title') }}</div>
            <div class="empty-sub">{{ t('notifications.empty_sub') }}</div>
          </div>

          <div v-else class="inbox">
            <div
              v-for="item in pending"
              :key="creditId(item.credit) + ':' + item.request.id"
              class="inbox-item"
            >
              <div class="item-head">
                <span class="src-pill" :class="item.side">
                  {{ item.side === 'borrowing'
                    ? t('notifications.from_borrowing')
                    : t('notifications.from_lending') }}
                </span>
                <span class="cp-name">{{ counterpartyName(item.credit, item.side) }}</span>
              </div>

              <div class="item-type">
                {{ t('ledger_detail.req_type.' + item.request.type) }}
              </div>

              <div class="item-amount">
                <span class="cur">{{ item.credit.currency || 'USD' }}</span>
                {{ moneyWhole(item.request.amount, item.credit.currency || 'USD') }}
              </div>

              <div v-if="item.request.description" class="item-note">
                <em>{{ item.request.description }}</em>
              </div>

              <div class="item-actions">
                <button class="linklike open-link" @click="openRecord(item)">
                  {{ t('notifications.review_in_detail') }}
                </button>
                <div class="btn-row">
                  <button
                    class="btn btn-secondary btn-sm"
                    :disabled="acting === creditId(item.credit) + ':' + item.request.id"
                    @click="decide(item, 'reject')"
                  >
                    {{ t('ledger_detail.action.reject') }}
                  </button>
                  <button
                    class="btn btn-primary btn-sm"
                    :disabled="acting === creditId(item.credit) + ':' + item.request.id"
                    @click="decide(item, 'approve')"
                  >
                    <Icon name="check" :size="12" />
                    {{ t('ledger_detail.action.approve') }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.empty-state {
  margin-top: 22px;
  padding: 28px 18px;
  text-align: center;
  border: 1px dashed var(--line-hard);
  border-radius: var(--r-lg);
}
.empty-title {
  font-family: var(--serif);
  font-size: 20px;
  color: var(--ink);
}
.empty-sub {
  font-family: var(--mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  color: var(--ink-mute);
  margin-top: 6px;
}

.inbox {
  margin-top: 18px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.inbox-item {
  background: var(--paper);
  border: 1px solid var(--line);
  border-radius: var(--r-lg);
  padding: 14px 16px;
}
.item-head {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.src-pill {
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.1em;
  padding: 3px 8px;
  border-radius: 4px;
  background: var(--line);
  color: var(--ink-mute);
}
.src-pill.borrowing {
  background: rgba(214, 73, 51, 0.1);
  color: var(--hot);
}
.src-pill.lending {
  background: rgba(47, 95, 79, 0.08);
  color: var(--moss);
}
.cp-name {
  font-family: var(--serif);
  font-size: 16px;
  color: var(--ink);
}
.item-type {
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.1em;
  color: var(--ink-mute);
  margin-top: 8px;
}
.item-amount {
  font-family: var(--serif);
  font-size: 28px;
  line-height: 1;
  color: var(--ink);
  margin-top: 6px;
}
.item-amount .cur {
  font-size: 14px;
  color: var(--ink-mute);
  margin-right: 2px;
}
.item-note {
  font-size: 13px;
  color: var(--ink-soft);
  line-height: 1.5;
  padding: 10px 12px;
  margin-top: 10px;
  background: var(--paper-deep);
  border-left: 2px solid var(--line-hard);
  border-radius: 0 8px 8px 0;
}
.item-actions {
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px dashed var(--line);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.btn-row {
  display: flex;
  gap: 6px;
}
.open-link {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  color: var(--ink-mute);
}
</style>
