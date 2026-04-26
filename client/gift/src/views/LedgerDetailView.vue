<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import Icon from '../components/Icon.vue'
import Avatar from '../components/Avatar.vue'
import { borrowingApi, lendingApi, type CreditActionResult } from '../api/endpoints'
import type { Credit, FinanceRequest, FinanceRequestType } from '../api/types'
import { auth } from '../stores/auth'
import { toast } from '../stores/toast'
import { colorForId, money, moneyWhole } from '../utils/format'
import { t } from '../i18n'

type Side = 'borrowing' | 'lending'

const props = defineProps<{ side: Side; id: string }>()
const router = useRouter()

const credit = ref<Credit | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const acting = ref(false)

const myId = computed(() => auth.userIdFromToken() ?? '')
const sideKey = computed(() => (props.side === 'borrowing' ? 'borrowed' : 'lent'))
const sideColor = computed(() => (props.side === 'borrowing' ? 'var(--hot)' : 'var(--moss)'))

const apiFor = computed(() => (props.side === 'borrowing' ? borrowingApi : lendingApi))

async function load() {
  loading.value = true
  error.value = null
  try {
    credit.value = await apiFor.value.get(props.id)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

// Counterparty: for a borrowing it's `from`; for a lending it's `to`.
const counterparty = computed(() => {
  const c = credit.value
  if (!c) return null
  return props.side === 'borrowing' ? c.from : c.to
})

const counterpartyName = computed(() => {
  const cp = counterparty.value
  if (!cp) return '—'
  if (cp.is_oid) return cp.oid.slice(-6)
  return cp.str || '—'
})

const counterpartyKey = computed(() => {
  const cp = counterparty.value
  if (!cp) return ''
  return cp.is_oid ? cp.oid : cp.str
})

const isInApp = computed(() => counterparty.value?.is_oid ?? false)

const actualAmount = computed(() => credit.value?.amount ?? 0)
const resolvedAmount = computed(() => credit.value?.resolved_amount ?? 0)
const remaining = computed(() => Math.max(0, actualAmount.value - resolvedAmount.value))
const pct = computed(() => (actualAmount.value > 0 ? resolvedAmount.value / actualAmount.value : 0))
const isSettled = computed(() => remaining.value <= 0.005 && actualAmount.value > 0)
const currency = computed(() => credit.value?.currency || 'USD')

const requests = computed<FinanceRequest[]>(() => credit.value?.finance_requests ?? [])

// --- amend modal ----------------------------------------------------------
const showAmend = ref(false)
const amendType = ref<FinanceRequestType>('increase_resolved_amount')
const amendAmount = ref<number | null>(null)
const amendNote = ref('')

function openAmend(type: FinanceRequestType) {
  amendType.value = type
  amendAmount.value = null
  amendNote.value = ''
  showAmend.value = true
}

function closeAmend() {
  showAmend.value = false
}

// Maps the amend type to the helper action endpoint. Both repay/take live
// under borrowings; give/collect live under lendings — the API layer handles
// route selection, we just pick the verb that matches the proposed delta.
async function submitAmend() {
  if (!credit.value || !amendAmount.value || amendAmount.value <= 0) return
  acting.value = true
  try {
    let result: CreditActionResult
    const body = { amount: amendAmount.value, description: amendNote.value || undefined }
    if (props.side === 'borrowing') {
      // increase_resolved → repay; increase_amount → take.
      // decrease_* are only available via the FinanceRequest path on two-OID
      // credits — the helper endpoints don't expose them, so we surface a
      // toast and bail rather than send something the server will reject.
      if (amendType.value === 'increase_resolved_amount') {
        result = await borrowingApi.repay(props.id, body)
      } else if (amendType.value === 'increase_amount') {
        result = await borrowingApi.take(props.id, body)
      } else {
        toast.flash('Decrease requests not supported via helper actions yet')
        acting.value = false
        return
      }
    } else {
      if (amendType.value === 'increase_resolved_amount') {
        result = await lendingApi.collect(props.id, body)
      } else if (amendType.value === 'increase_amount') {
        result = await lendingApi.give(props.id, body)
      } else {
        toast.flash('Decrease requests not supported via helper actions yet')
        acting.value = false
        return
      }
    }
    credit.value = result.credit
    toast.flash(
      result.applied
        ? t('ledger_detail.applied_toast', { amount: money(amendAmount.value, currency.value) })
        : t('ledger_detail.queued_toast'),
    )
    showAmend.value = false
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Action failed'
  } finally {
    acting.value = false
  }
}

// --- request review -------------------------------------------------------
async function reviewRequest(req: FinanceRequest, action: 'approve' | 'reject') {
  if (!credit.value) return
  acting.value = true
  try {
    const fn = action === 'approve' ? apiFor.value.approveRequest : apiFor.value.rejectRequest
    credit.value = await fn(props.id, req.id)
    toast.flash(action === 'approve' ? t('ledger_detail.approved_toast') : t('ledger_detail.rejected_toast'))
  } catch (e) {
    toast.flash(e instanceof Error ? e.message : 'Decision failed')
  } finally {
    acting.value = false
  }
}

function statusClass(s: FinanceRequest['status']): string {
  return `status-${s}`
}

function back() {
  router.push({ path: '/ledger', query: { side: sideKey.value } })
}

watch(() => [props.id, props.side], load)
onMounted(load)
</script>

<template>
  <section>
    <button class="back linklike" @click="back">
      <Icon name="arrowLeft" :size="14" />
      {{ props.side === 'borrowing' ? t('ledger_detail.borrowed_eyebrow') : t('ledger_detail.lent_eyebrow') }}
    </button>

    <p v-if="loading" class="muted">{{ t('common.loading') }}</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <template v-else-if="credit">
      <!-- Counterparty header -->
      <div class="cp-head">
        <Avatar
          :name="counterpartyName"
          :color="isInApp ? colorForId(counterpartyKey) : 'var(--ink-ghost)'"
          :size="56"
        />
        <div style="min-width: 0">
          <div class="cp-name">{{ counterpartyName }}</div>
          <div class="cp-meta">
            <span class="cp-tag" :class="{ inapp: isInApp }">
              {{ isInApp ? t('ledger.in_app') : t('ledger.off_app') }}
            </span>
          </div>
        </div>
      </div>

      <!-- Big numbers card -->
      <div class="card-ink total-card">
        <div class="mono-label">
          {{ props.side === 'borrowing' ? t('ledger_detail.you_owe_outstanding') : t('ledger_detail.they_owe_outstanding') }}
        </div>
        <div class="money-big">
          <span class="cur">{{ currency }}</span>
          {{ Math.floor(remaining).toLocaleString() }}<span class="decimals">.{{ (remaining % 1).toFixed(2).slice(2) }}</span>
        </div>
        <div class="ink-progress">
          <div
            class="ink-progress-fill"
            :style="{ width: `${pct * 100}%`, background: isSettled ? '#4ADE80' : sideColor }"
          ></div>
        </div>
        <div class="ink-grid">
          <div>
            <div class="mono-sub">{{ t('ledger_detail.actual_principal') }}</div>
            <div class="ink-num">{{ moneyWhole(actualAmount, currency) }}</div>
          </div>
          <div>
            <div class="mono-sub">{{ t('ledger_detail.resolved_paid') }}</div>
            <div class="ink-num moss">{{ moneyWhole(resolvedAmount, currency) }}</div>
          </div>
        </div>
      </div>

      <!-- Memo -->
      <div class="memo-block">
        <div class="eyebrow">{{ t('ledger_detail.memo') }}</div>
        <div class="memo-body">
          <em>{{ credit.description || t('ledger_detail.no_memo') }}</em>
        </div>
      </div>

      <!-- Action buttons -->
      <div class="actions">
        <button
          class="btn btn-primary btn-lg btn-block"
          :disabled="acting"
          @click="openAmend('increase_resolved_amount')"
        >
          <Icon name="plus" :size="14" />
          {{ props.side === 'borrowing' ? t('ledger_detail.log_repayment') : t('ledger_detail.log_collection') }}
        </button>
        <button
          class="btn btn-secondary btn-lg btn-block"
          :disabled="acting"
          @click="openAmend('increase_amount')"
        >
          <Icon name="plus" :size="14" />
          {{ props.side === 'borrowing' ? t('ledger_detail.take_more') : t('ledger_detail.add_more') }}
        </button>
      </div>

      <!-- Amendment timeline -->
      <div class="section-block">
        <div class="row spread">
          <span class="eyebrow">{{ t('ledger_detail.requests_count', { n: requests.length }) }}</span>
        </div>
        <p class="lead small">{{ t('ledger_detail.disputes_intro') }}</p>

        <div v-if="!requests.length" class="empty" style="margin-top: 12px">
          {{ t('ledger_detail.no_amendments') }}
        </div>
        <div v-else class="req-list">
          <div v-for="r in requests" :key="r.id" class="req-card">
            <div class="req-head">
              <span class="status-pill" :class="statusClass(r.status)">
                <span class="dot"></span>
                {{ t('ledger_detail.status.' + r.status) }}
              </span>
              <span class="req-meta">
                {{ t('ledger_detail.req_type.' + r.type) }}
                · {{ r.requested_by === myId ? t('ledger_detail.proposed_by_you') : counterpartyName }}
              </span>
            </div>

            <div class="req-amount">
              <span class="cur">{{ currency }}</span>
              {{ moneyWhole(r.amount, currency) }}
            </div>

            <div v-if="r.description" class="req-note">
              <em>{{ r.description }}</em>
            </div>

            <div v-if="r.status === 'pending'" class="req-actions">
              <span class="req-await">
                {{ r.requested_by === myId ? t('ledger_detail.awaiting_them') : t('ledger_detail.awaiting_you') }}
              </span>
              <div v-if="r.requested_by !== myId" class="btn-row">
                <button
                  class="btn btn-secondary btn-sm"
                  :disabled="acting"
                  @click="reviewRequest(r, 'reject')"
                >
                  {{ t('ledger_detail.action.reject') }}
                </button>
                <button
                  class="btn btn-primary btn-sm"
                  :disabled="acting"
                  @click="reviewRequest(r, 'approve')"
                >
                  <Icon name="check" :size="12" />
                  {{ t('ledger_detail.action.approve') }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- Amend modal -->
    <Teleport to="body">
      <div v-if="showAmend" class="modal-backdrop" @click.self="closeAmend">
        <div class="modal">
          <div class="modal-header">
            <button class="linklike" @click="closeAmend">
              <Icon name="close" :size="16" /> {{ t('ledger_detail.cancel').toUpperCase() }}
            </button>
            <div class="eyebrow">{{ t('ledger_detail.req_type.' + amendType) }}</div>
          </div>
          <div class="modal-body">
            <label class="field">
              <span>{{ t('ledger_detail.amount_input_label') }} ({{ currency }})</span>
              <input v-model.number="amendAmount" type="number" min="0" step="0.01" />
            </label>
            <label class="field" style="margin-top: 14px">
              <span>{{ t('ledger_detail.note_input_label') }}</span>
              <input v-model="amendNote" />
            </label>
            <div style="margin-top: 28px">
              <button
                class="btn btn-primary btn-lg btn-block"
                :disabled="!amendAmount || amendAmount <= 0 || acting"
                @click="submitAmend"
              >
                <Icon name="check" :size="16" />
                {{ t('ledger_detail.confirm') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<style scoped>
.back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  color: var(--ink-soft);
  text-decoration: none;
}

.cp-head {
  margin-top: 14px;
  display: flex;
  align-items: center;
  gap: 14px;
}
.cp-head .cp-name {
  font-family: var(--serif);
  font-size: 30px;
  line-height: 1.05;
  letter-spacing: -0.01em;
  color: var(--ink);
}
.cp-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
}
.cp-tag {
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.08em;
  color: var(--ink-mute);
  background: var(--line);
  padding: 2px 6px;
  border-radius: 4px;
}
.cp-tag.inapp {
  color: var(--moss);
  background: rgba(47, 95, 79, 0.08);
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
.ink-progress {
  margin-top: 14px;
  height: 5px;
  background: rgba(245, 241, 232, 0.12);
  border-radius: 3px;
  overflow: hidden;
}
.ink-progress-fill {
  height: 100%;
}
.ink-grid {
  margin-top: 14px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  padding-top: 14px;
  border-top: 1px solid rgba(245, 241, 232, 0.1);
}
.mono-sub {
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.1em;
  color: rgba(245, 241, 232, 0.5);
}
.ink-num {
  font-family: var(--serif);
  font-size: 24px;
  margin-top: 3px;
  line-height: 1;
  color: var(--paper);
}
.ink-num.moss {
  color: #86efac;
}

.memo-block {
  margin-top: 18px;
}
.memo-body {
  font-family: var(--serif);
  font-style: italic;
  font-size: 18px;
  color: var(--ink);
  margin-top: 6px;
  line-height: 1.4;
}

.actions {
  margin-top: 18px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.section-block {
  margin-top: 30px;
}
.lead.small {
  font-size: 12px;
  color: var(--ink-soft);
  margin: 8px 0 0;
  line-height: 1.5;
}

.req-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 14px;
}
.req-card {
  background: var(--paper);
  border: 1px solid var(--line);
  border-radius: var(--r-lg);
  padding: 14px 16px;
}
.req-head {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 8px;
  border-radius: 4px;
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.1em;
}
.status-pill .dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
}
.status-pending {
  background: rgba(214, 73, 51, 0.1);
  color: var(--hot);
}
.status-pending .dot {
  background: var(--hot);
}
.status-approved {
  background: rgba(47, 95, 79, 0.08);
  color: var(--moss);
}
.status-approved .dot {
  background: var(--moss);
}
.status-rejected {
  background: rgba(20, 23, 31, 0.06);
  color: var(--ink-mute);
}
.status-rejected .dot {
  background: var(--ink-mute);
}
.req-meta {
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.1em;
  color: var(--ink-mute);
}
.req-amount {
  font-family: var(--serif);
  font-size: 28px;
  line-height: 1;
  color: var(--ink);
  margin-top: 10px;
}
.req-amount .cur {
  font-size: 14px;
  color: var(--ink-mute);
  margin-right: 2px;
}
.req-note {
  font-size: 13px;
  color: var(--ink-soft);
  line-height: 1.5;
  padding: 10px 12px;
  margin-top: 10px;
  background: var(--paper-deep);
  border-left: 2px solid var(--line-hard);
  border-radius: 0 8px 8px 0;
}
.req-actions {
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px dashed var(--line);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.req-await {
  font-family: var(--mono);
  font-size: 9px;
  letter-spacing: 0.08em;
  color: var(--ink-mute);
}
.btn-row {
  display: flex;
  gap: 6px;
}
</style>
