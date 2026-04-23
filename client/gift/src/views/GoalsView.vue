<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Icon from '../components/Icon.vue'
import { goalApi } from '../api/endpoints'
import type { Goal } from '../api/types'
import { toast } from '../stores/toast'

const goals = ref<Goal[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

// Create modal
const showCreate = ref(false)
const creating = ref(false)
const newName = ref('')
const newTarget = ref<number | null>(null)
const newCurrent = ref<number | null>(null)
const newDeadline = ref('')
const newTint = ref('#D64933')

// Contribute modal
const contribFor = ref<Goal | null>(null)
const contribAmount = ref<number | null>(null)
const contributing = ref(false)

const TINTS = ['#D64933', '#2F5F4F', '#B8915A', '#4A5577', '#8B4A55', '#5B6E4A']

function tintFor(g: Goal): string {
  const idx =
    (g.id ?? g.name)
      .split('')
      .reduce((a, c) => (a * 31 + c.charCodeAt(0)) >>> 0, 0) % TINTS.length
  return TINTS[idx]
}

function deadlineLabel(iso: string): string {
  if (!iso) return 'Ongoing'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return 'Ongoing'
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

async function load() {
  loading.value = true
  error.value = null
  try {
    goals.value = (await goalApi.list()) ?? []
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

function openCreate() {
  newName.value = ''
  newTarget.value = null
  newCurrent.value = null
  newDeadline.value = ''
  newTint.value = TINTS[0]
  showCreate.value = true
}

async function createGoal() {
  if (!newName.value.trim() || !newTarget.value) return
  creating.value = true
  try {
    await goalApi.create({
      name: newName.value.trim(),
      target_amount: newTarget.value,
      current_amount: newCurrent.value ?? 0,
      currency: '$',
      deadline: newDeadline.value ? new Date(newDeadline.value).toISOString() : '',
    })
    showCreate.value = false
    toast.flash('Goal created')
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to create'
  } finally {
    creating.value = false
  }
}

function openContribute(g: Goal) {
  contribFor.value = g
  contribAmount.value = null
}

async function confirmContribute() {
  if (!contribFor.value || !contribAmount.value || contribAmount.value <= 0) return
  contributing.value = true
  try {
    await goalApi.contribute(String(contribFor.value.id), contribAmount.value)
    toast.flash(`+$${contribAmount.value} toward ${contribFor.value.name}`)
    contribFor.value = null
    contribAmount.value = null
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to contribute'
  } finally {
    contributing.value = false
  }
}

async function remove(id: string) {
  if (!confirm('Delete this goal?')) return
  try {
    await goalApi.remove(id)
    await load()
    toast.flash('Goal deleted')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

onMounted(load)
</script>

<template>
  <section class="goals">
    <header class="row spread top-actions">
      <span class="eyebrow">SELF-HOST · GOALS</span>
    </header>

    <h1 class="hero">
      What we're <em>saving for.</em>
    </h1>

    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <template v-else>
      <div v-if="goals.length" class="goal-list">
        <div v-for="g in goals" :key="String(g.id)" class="goal-card">
          <div class="row spread">
            <div class="row" style="gap: 10px; align-items: center">
              <div class="tint-mark" :style="{ background: tintFor(g) }"></div>
              <div class="eyebrow">
                TARGET · {{ deadlineLabel(g.deadline).toUpperCase() }}
              </div>
            </div>
            <button
              class="icon-btn delete-btn"
              @click="remove(String(g.id))"
              aria-label="Delete"
            >
              <Icon name="close" :size="14" />
            </button>
          </div>
          <div class="goal-title">{{ g.name }}</div>
          <div class="goal-progress">
            <div class="goal-saved">
              <span class="cur">$</span>{{ Math.round(g.current_amount).toLocaleString() }}
            </div>
            <div class="goal-target">
              / ${{ Math.round(g.target_amount).toLocaleString() }} ·
              {{ g.target_amount ? Math.round((g.current_amount / g.target_amount) * 100) : 0 }}%
            </div>
          </div>
          <div class="goal-rail">
            <div
              class="goal-fill"
              :style="{
                width:
                  Math.min(g.target_amount ? g.current_amount / g.target_amount : 0, 1) *
                    100 +
                  '%',
                background: tintFor(g),
              }"
            ></div>
          </div>
          <div class="row spread goal-foot">
            <div class="goal-remaining">
              ${{ Math.max(0, Math.round(g.target_amount - g.current_amount)).toLocaleString() }} TO GO
            </div>
            <button class="contribute-btn" @click="openContribute(g)">
              + CONTRIBUTE
            </button>
          </div>
        </div>
      </div>

      <div v-else class="empty">No goals yet. Set your first target.</div>

      <button class="dashed" @click="openCreate">
        <Icon name="plus" :size="16" /> New goal
      </button>
    </template>

    <!-- Create modal -->
    <Teleport to="body">
      <div v-if="showCreate" class="modal-backdrop" @click.self="showCreate = false">
        <div class="modal">
          <div class="modal-header">
            <button class="linklike" @click="showCreate = false">
              <Icon name="close" :size="16" /> CANCEL
            </button>
            <div class="eyebrow">NEW GOAL</div>
          </div>

          <div class="modal-body">
            <h1 class="display">A new <em>target.</em></h1>

            <label class="field" style="margin-top: 20px">
              <span>NAME</span>
              <input
                v-model="newName"
                placeholder="Bali · Group kitty"
              />
            </label>

            <div class="stack-form split" style="margin-top: 14px">
              <label class="field">
                <span>TARGET ($)</span>
                <input
                  v-model.number="newTarget"
                  type="number"
                  min="0"
                  step="0.01"
                  placeholder="5000"
                />
              </label>
              <label class="field">
                <span>SAVED SO FAR ($)</span>
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
              <span>DEADLINE</span>
              <input v-model="newDeadline" type="date" />
            </label>

            <div style="margin-top: 28px">
              <button
                class="btn btn-primary btn-lg btn-block"
                :disabled="!newName.trim() || !newTarget || creating"
                @click="createGoal"
              >
                <Icon name="check" :size="18" />
                {{ creating ? 'Saving…' : 'Create goal' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Contribute modal -->
    <Teleport to="body">
      <div
        v-if="contribFor"
        class="modal-backdrop"
        @click.self="contribFor = null"
      >
        <div class="modal">
          <div class="modal-header">
            <button class="linklike" @click="contribFor = null">
              <Icon name="close" :size="16" /> CANCEL
            </button>
            <div class="eyebrow">CONTRIBUTE</div>
          </div>
          <div class="modal-body">
            <h1 class="display">Toward <em>{{ contribFor.name }}.</em></h1>
            <label class="field" style="margin-top: 20px">
              <span>AMOUNT ($)</span>
              <input
                v-model.number="contribAmount"
                type="number"
                min="0"
                step="0.01"
                placeholder="100"
                autofocus
              />
            </label>
            <div style="margin-top: 28px">
              <button
                class="btn btn-accent btn-lg btn-block"
                :disabled="!contribAmount || contribAmount <= 0 || contributing"
                @click="confirmContribute"
              >
                <Icon name="check" :size="18" />
                {{ contributing ? 'Saving…' : 'Add contribution' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<style scoped>
.goals {
  padding: 4px 0;
}

.top-actions {
  margin-bottom: 8px;
}

.hero {
  font-size: 44px;
}

.goal-list {
  margin-top: 22px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.goal-card {
  background: var(--paper);
  border: 1px solid var(--line);
  border-radius: var(--r-lg);
  padding: 18px 20px;
}

.tint-mark {
  width: 10px;
  height: 10px;
  border-radius: 2px;
}

.goal-title {
  font-family: var(--serif);
  font-size: 26px;
  line-height: 1.1;
  margin: 8px 0 2px;
  letter-spacing: -0.01em;
}

.goal-progress {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-top: 10px;
}

.goal-saved {
  font-family: var(--serif);
  font-size: 34px;
  color: var(--ink);
  line-height: 1;
}

.goal-saved .cur {
  font-size: 14px;
  color: var(--ink-mute);
}

.goal-target {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--ink-mute);
}

.goal-rail {
  margin-top: 12px;
  height: 6px;
  border-radius: 3px;
  background: var(--line);
  position: relative;
  overflow: hidden;
}

.goal-fill {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  border-radius: 3px;
}

.goal-foot {
  margin-top: 12px;
}

.goal-remaining {
  font-family: var(--mono);
  font-size: 10px;
  color: var(--ink-mute);
  letter-spacing: 0.06em;
}

.contribute-btn {
  background: var(--ink);
  color: var(--paper);
  border: none;
  border-radius: 8px;
  padding: 6px 12px;
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.08em;
  cursor: pointer;
}

.contribute-btn:hover {
  background: #000;
}

.delete-btn {
  color: var(--ink-ghost);
}

.delete-btn:hover {
  color: var(--hot);
}

.dashed {
  margin-top: 20px;
  padding: 18px;
}
</style>
