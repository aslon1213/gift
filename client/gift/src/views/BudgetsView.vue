<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Icon from '../components/Icon.vue'
import Ring from '../components/Ring.vue'
import VoiceInputButton from '../components/VoiceInputButton.vue'
import type { IconName } from '../components/icons'
import { budgetApi } from '../api/endpoints'
import type { Budget } from '../api/types'
import { toast } from '../stores/toast'
import { userStore } from '../stores/user'
import { currencySymbol, moneyWhole } from '../utils/format'
import { parseBudgetFromAudio, type BudgetDraft } from '../ai/parse'
import { t } from '../i18n'

const currency = computed(() => userStore.currency.value)
const curSymbol = computed(() => currencySymbol(currency.value))

const budgets = ref<Budget[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

// Create form
const showCreate = ref(false)
const creating = ref(false)
const newCategory = ref('')
const newLimit = ref<number | null>(null)
const newCurrent = ref<number | null>(null)
const newPeriod = ref('monthly')

const CATEGORY_ICON: Record<string, IconName> = {
  travel: 'plane',
  flights: 'plane',
  stay: 'bed',
  hotel: 'bed',
  rent: 'bed',
  food: 'fork',
  dining: 'fork',
  'eating out': 'fork',
  transport: 'car',
  ride: 'car',
  activity: 'ticket',
  groceries: 'cart',
  cafe: 'coffee',
  coffee: 'coffee',
  home: 'home2',
}

function iconFor(cat: string): IconName {
  return CATEGORY_ICON[cat.toLowerCase().trim()] ?? 'wallet'
}

// `amount` is now the spent-so-far value tracked by the server; `limit` is the cap.
function limitFor(b: Budget): number {
  return b.limit || 0
}
function spentFor(b: Budget): number {
  return b.amount || 0
}
function pctFor(b: Budget): number {
  return limitFor(b) ? spentFor(b) / limitFor(b) : 0
}

const total = computed(() => budgets.value.reduce((a, b) => a + limitFor(b), 0))
const used = computed(() => budgets.value.reduce((a, b) => a + spentFor(b), 0))

async function load() {
  loading.value = true
  error.value = null
  try {
    budgets.value = (await budgetApi.list()) ?? []
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

function openCreate() {
  newCategory.value = ''
  newLimit.value = null
  newCurrent.value = null
  newPeriod.value = 'monthly'
  showCreate.value = true
}

const VALID_PERIODS = ['weekly', 'monthly', 'trip', 'yearly'] as const

function applyBudgetDraft(draft: BudgetDraft) {
  if (draft.category) newCategory.value = draft.category
  if (draft.limit != null && draft.limit > 0) newLimit.value = draft.limit
  if (draft.amount != null && draft.amount >= 0) newCurrent.value = draft.amount
  if (draft.period && (VALID_PERIODS as readonly string[]).includes(draft.period)) {
    newPeriod.value = draft.period
  }
  toast.flash(t('voice.filled_from_speech'))
}

function onVoiceError(msg: string) {
  toast.flash(msg)
}

async function createBudget() {
  if (!newCategory.value.trim() || !newLimit.value) return
  creating.value = true
  try {
    await budgetApi.create({
      category: newCategory.value.trim(),
      limit: newLimit.value,
      amount: newCurrent.value ?? 0,
      period: newPeriod.value,
      currency: currency.value,
      start_date: new Date().toISOString(),
    })
    showCreate.value = false
    toast.flash('Budget added')
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to create'
  } finally {
    creating.value = false
  }
}

async function remove(id: string) {
  if (!confirm('Delete this budget?')) return
  try {
    await budgetApi.remove(id)
    await load()
    toast.flash('Budget deleted')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

onMounted(load)
</script>

<template>
  <section class="budgets">
    <h1 class="hero">{{ t('budgets.stay_honest') }}</h1>
    <p class="lead">{{ t('budgets.lead') }}</p>

    <p v-if="loading" class="muted">{{ t('common.loading') }}</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <template v-else>
      <div v-if="budgets.length" class="overall">
        <div>
          <div class="eyebrow">{{ t('budgets.overall') }}</div>
          <div class="overall-money">
            {{ moneyWhole(used, currency) }}
            <span class="overall-total">/ {{ moneyWhole(total, currency) }}</span>
          </div>
          <div class="overall-sub">
            {{ total ? Math.round((used / total) * 100) : 0 }}% {{ t('budgets.used') }} ·
            {{ moneyWhole(Math.max(0, total - used), currency) }} {{ t('budgets.left') }}
          </div>
        </div>
        <Ring
          :pct="total ? used / total : 0"
          :size="72"
          :stroke="6"
          color="var(--ink)"
          bg="var(--line)"
        />
      </div>

      <div class="section-head eyebrow">{{ t('budgets.by_category') }}</div>

      <div v-if="budgets.length" class="cat-list">
        <div
          v-for="(b, i) in budgets"
          :key="String(b.id)"
          class="cat-row"
          :class="{ last: i === budgets.length - 1 }"
        >
          <div class="ring-wrap">
            <Ring
              :pct="pctFor(b)"
              :size="44"
              :stroke="3"
              :color="pctFor(b) > 1 ? '#D64933' : '#14171F'"
              bg="var(--line)"
            />
            <div
              class="ring-icon"
              :style="{ color: pctFor(b) > 1 ? 'var(--hot)' : 'var(--ink)' }"
            >
              <Icon :name="iconFor(b.category)" :size="16" />
            </div>
          </div>
          <div>
            <div class="cat-label">
              {{ b.category }}
              <span v-if="pctFor(b) > 1" class="over-tag">{{ t('budgets.over') }}</span>
            </div>
            <div class="cat-sub" :class="{ over: pctFor(b) > 1 }">
              {{ moneyWhole(spentFor(b), currency) }} /
              {{ moneyWhole(limitFor(b), currency) }} ·
              {{ Math.round(pctFor(b) * 100) }}% ·
              {{ b.period.toUpperCase() }}
            </div>
          </div>
          <button
            class="icon-btn delete-btn"
            @click="remove(String(b.id))"
            aria-label="Delete budget"
          >
            <Icon name="close" :size="14" />
          </button>
        </div>
      </div>

      <div v-else class="empty">{{ t('common.no_data') }}</div>

      <button class="dashed" @click="openCreate">
        <Icon name="plus" :size="14" /> {{ t('budgets.new_category') }}
      </button>
    </template>

    <!-- Create modal -->
    <Teleport to="body">
      <div v-if="showCreate" class="modal-backdrop" @click.self="showCreate = false">
        <div class="modal">
          <div class="modal-header">
            <button class="linklike" @click="showCreate = false">
              <Icon name="close" :size="16" /> {{ t('common.cancel') }}
            </button>
            <div class="eyebrow">{{ t('budgets.new_category') }}</div>
          </div>

          <div class="modal-body">
            <div class="row spread" style="align-items: center; margin-bottom: 10px">
              <h1 class="display" style="margin: 0">{{ t('budgets.cap_a_category') }}</h1>
              <VoiceInputButton
                :parser="parseBudgetFromAudio"
                @result="applyBudgetDraft"
                @error="onVoiceError"
              />
            </div>

            <label class="field" style="margin-top: 20px">
              <span>{{ t('budgets.category') }}</span>
              <input
                v-model="newCategory"
                :placeholder="t('budgets.category_placeholder')"
              />
            </label>

            <div class="stack-form split" style="margin-top: 14px">
              <label class="field">
                <span>{{ t('budgets.limit') }} ({{ currency }})</span>
                <input
                  v-model.number="newLimit"
                  type="number"
                  min="0"
                  step="0.01"
                  :placeholder="curSymbol + '600'"
                />
              </label>
              <label class="field">
                <span>{{ t('budgets.already_spent') }} ({{ currency }})</span>
                <input
                  v-model.number="newCurrent"
                  type="number"
                  min="0"
                  step="0.01"
                  placeholder="0"
                />
              </label>
            </div>

            <label class="field" style="margin-top: 14px">
              <span>{{ t('budgets.period') }}</span>
              <div style="display: flex; gap: 6px">
                <button
                  v-for="p in ['weekly', 'monthly', 'trip', 'yearly']"
                  :key="p"
                  type="button"
                  class="pill"
                  :class="{ on: newPeriod === p }"
                  @click="newPeriod = p"
                >
                  {{ t('budgets.period.' + p) }}
                </button>
              </div>
            </label>

            <div style="margin-top: 28px">
              <button
                class="btn btn-primary btn-lg btn-block"
                :disabled="!newCategory.trim() || !newLimit || creating"
                @click="createBudget"
              >
                <Icon name="check" :size="18" />
                {{ creating ? t('common.loading') : t('budgets.create_budget') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<style scoped>
.budgets {
  padding: 4px 0;
}

.lead {
  font-size: 13px;
  color: var(--ink-soft);
  margin: 8px 0 0;
  max-width: 300px;
}

.overall {
  margin-top: 22px;
  padding: 18px 20px;
  background: var(--paper-deep);
  border: 1px solid var(--line);
  border-radius: var(--r-lg);
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 16px;
  align-items: center;
}

.overall-money {
  font-family: var(--serif);
  font-size: clamp(22px, 6.5vw, 30px);
  line-height: 1;
  margin-top: 4px;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: clip;
}

.overall-money .cur {
  font-size: clamp(12px, 3.5vw, 14px);
  color: var(--ink-mute);
}

.overall-total {
  font-size: clamp(12px, 3.5vw, 14px);
  color: var(--ink-mute);
}

.overall-sub {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--ink-mute);
  margin-top: 6px;
}

.section-head {
  margin: 26px 0 8px;
}

.cat-list {
  display: flex;
  flex-direction: column;
}

.cat-row {
  display: grid;
  grid-template-columns: 44px 1fr auto;
  gap: 14px;
  align-items: center;
  padding: 14px 0;
  border-bottom: 1px solid var(--line);
}

.cat-row.last {
  border-bottom: none;
}

.ring-wrap {
  position: relative;
  width: 44px;
  height: 44px;
}

.ring-icon {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cat-label {
  font-size: 15px;
  font-weight: 500;
  color: var(--ink);
  text-transform: capitalize;
}

.over-tag {
  color: var(--hot);
  font-family: var(--mono);
  font-size: 10px;
  margin-left: 6px;
  letter-spacing: 0.08em;
}

.cat-sub {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--ink-mute);
  margin-top: 3px;
}

.cat-sub.over {
  color: var(--hot);
}

.delete-btn {
  color: var(--ink-ghost);
}

.delete-btn:hover {
  color: var(--hot);
}

.dashed {
  margin-top: 20px;
  padding: 14px;
  font-size: 13px;
}
</style>
