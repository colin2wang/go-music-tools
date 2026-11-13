<script setup lang="ts">
import { computed, ref, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CheckCircle2, XCircle, SkipForward, Clock3, Square, ChevronDown, ChevronRight,
  RotateCcw, Trash2, Layers, Zap, Timer, AlertCircle, Loader2, Ban, CircleSlash,
} from 'lucide-vue-next'
import { useTaskStore } from '@/stores/task'
import type { TaskEntry, TaskStatus } from '@/types'

const props = defineProps<{ entry: TaskEntry; position?: number }>()
const { t } = useI18n()
const tasks = useTaskStore()
const expanded = ref(false)
const onlyFailed = ref(false)

const KIND_KEYS: Record<string, string> = {
  'download:song': 'task.kindSong',
  'download:album': 'task.kindAlbum',
  'download:playlist': 'task.kindPlaylist',
  'oomusic:single': 'task.kindOOSingle',
  'oomusic:batch': 'task.kindOOBatch',
}

const isFinished = computed(() => props.entry.status !== 'running' && props.entry.status !== 'queued')
const isRetryTask = computed(() => props.entry.kind.endsWith(':retry'))

const laneLabel = computed(() =>
  props.entry.lane === 'oomusic' ? t('task.laneOOMusic') : t('task.laneWorkbench'),
)
const laneIcon = computed<Component>(() => props.entry.lane === 'oomusic' ? Zap : Layers)

const kindLabel = computed(() => {
  const base = props.entry.kind.replace(/:retry$/, '')
  const key = KIND_KEYS[base] ?? (base.startsWith('library:') ? 'task.kindLibrary' : '')
  const label = key ? t(key) : base
  return isRetryTask.value ? `${label} · ${t('task.retryBadge')}` : label
})

const statusMeta = computed<{ label: string; cls: string; icon: Component }>(() => {
  const map: Record<TaskStatus, { label: string; cls: string; icon: Component }> = {
    queued: { label: t('task.statusQueued'), cls: 'badge-queued', icon: Clock3 },
    running: { label: t('task.statusRunning'), cls: 'badge-running', icon: Loader2 },
    done: { label: t('task.statusDone'), cls: 'badge-done', icon: CheckCircle2 },
    failed: { label: t('task.statusFailed'), cls: 'badge-failed', icon: XCircle },
    cancelled: { label: t('task.statusCancelled'), cls: 'badge-cancelled', icon: Ban },
    removed: { label: t('task.statusRemoved'), cls: 'badge-cancelled', icon: CircleSlash },
  }
  return map[props.entry.status] ?? map.queued
})

const percent = computed(() => {
  const e = props.entry
  if (e.total <= 0) return isFinished.value ? 100 : 0
  return Math.min(100, Math.round((e.current / e.total) * 100))
})

const elapsed = computed(() => {
  const e = props.entry
  let ms = e.elapsed_ms
  if (e.status === 'running' && e.started_at > 0) {
    ms = Math.max(0, Date.now() - e.started_at * 1000)
  }
  if (!ms) return '—'
  if (ms < 1000) return `${Math.round(ms)}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${Math.floor(ms / 60000)}m${Math.round((ms % 60000) / 1000)}s`
})

const items = computed(() => {
  const list = props.entry.items.slice(-40).reverse()
  return onlyFailed.value ? list.filter((item) => item.status === 'failed') : list
})

const canRetry = computed(() => isFinished.value && props.entry.can_retry && props.entry.failed > 0)
const retryHint = computed(() => {
  if (!props.entry.can_retry) return t('task.retryDisabled')
  if (props.entry.failed <= 0) return t('task.retryNoFailed')
  return t('task.retryHint', { n: props.entry.failed })
})

function clockTime(seconds: number): string {
  if (!seconds) return ''
  return new Date(seconds * 1000).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

function itemIcon(status: string): Component {
  if (status === 'failed') return XCircle
  if (status === 'skipped') return SkipForward
  return CheckCircle2
}
</script>

<template>
  <article class="task-card" :class="`card-${entry.status}`">
    <div class="flex items-start gap-2.5">
      <span class="lane-chip" :class="entry.lane === 'oomusic' ? 'lane-oo' : 'lane-work'">
        <component :is="laneIcon" :size="11" /> {{ laneLabel }}
      </span>

      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <span class="truncate text-[12px] font-semibold text-[var(--text-primary)]" :title="entry.kind">{{ kindLabel }}</span>
          <span v-if="position" class="queue-position">#{{ position }}</span>
        </div>
        <p class="mt-1 truncate text-[10.5px] text-[var(--text-muted)]" :title="entry.stage || entry.current_item">
          {{ entry.stage || entry.current_item || t('task.none') }}
        </p>
      </div>

      <span class="status-badge" :class="statusMeta.cls">
        <component :is="statusMeta.icon" :size="11" :class="entry.status === 'running' ? 'animate-spin' : ''" />
        {{ statusMeta.label }}
      </span>
    </div>

    <div v-if="entry.status !== 'queued'" class="mt-2.5">
      <div class="mb-1.5 flex items-center justify-between gap-2 text-[10px] text-[var(--text-secondary)]">
        <span class="min-w-0 truncate" :title="entry.current_item">{{ entry.current_item || '—' }}</span>
        <span class="shrink-0 font-mono text-[var(--brand-300)]">{{ entry.current }}/{{ entry.total }} · {{ percent }}%</span>
      </div>
      <div class="progress-track">
        <div class="progress-fill" :style="{ width: percent + '%' }" />
      </div>
    </div>

    <div class="mt-2 flex flex-wrap items-center gap-2 text-[10.5px]">
      <span class="stat-pill stat-ok"><CheckCircle2 :size="11" /> {{ entry.success }}</span>
      <span class="stat-pill stat-warn"><SkipForward :size="11" /> {{ entry.skipped }}</span>
      <span class="stat-pill stat-err"><XCircle :size="11" /> {{ entry.failed }}</span>
      <span class="ml-auto flex items-center gap-1 text-[var(--text-muted)]"><Timer :size="11" /> {{ elapsed }}</span>
    </div>

    <div v-if="entry.error" class="error-strip">
      <AlertCircle :size="11" class="mt-[1px] shrink-0" />
      <span class="break-all">{{ entry.error }}</span>
    </div>

    <div class="mt-2.5 flex items-center gap-1.5">
      <button
        v-if="entry.item_count"
        type="button"
        class="mini-button"
        @click="expanded = !expanded"
      >
        <component :is="expanded ? ChevronDown : ChevronRight" :size="11" />
        {{ expanded ? t('task.collapseItems') : t('task.expandItems', { n: entry.item_count }) }}
      </button>
      <span v-else class="text-[10px] text-[var(--text-muted)]">{{ t('task.noItems') }}</span>

      <div class="ml-auto flex items-center gap-1.5">
        <button
          v-if="entry.status === 'running'"
          type="button"
          class="mini-button mini-danger"
          @click="tasks.cancelTask(entry.task_id)"
        >
          <Square :size="10" /> {{ t('task.cancel') }}
        </button>
        <button
          v-if="entry.status === 'queued'"
          type="button"
          class="mini-button mini-danger"
          @click="tasks.removeQueued(entry.task_id)"
        >
          <Trash2 :size="11" /> {{ t('task.remove') }}
        </button>
        <button
          v-if="isFinished"
          type="button"
          class="mini-button"
          :class="canRetry ? 'mini-accent' : 'mini-disabled'"
          :disabled="!canRetry"
          :title="retryHint"
          @click="tasks.retryFailed(entry.task_id)"
        >
          <RotateCcw :size="11" /> {{ t('task.retry') }}
        </button>
      </div>
    </div>

    <div v-if="expanded" class="items-panel">
      <div class="mb-2 flex items-center gap-1">
        <button
          type="button"
          class="filter-chip"
          :class="!onlyFailed ? 'filter-active' : ''"
          @click="onlyFailed = false"
        >{{ t('task.itemsAll') }}</button>
        <button
          type="button"
          class="filter-chip"
          :class="onlyFailed ? 'filter-active' : ''"
          @click="onlyFailed = true"
        >{{ t('task.itemsFailed') }} ({{ entry.failed }})</button>
        <span v-if="entry.item_count > entry.items.length" class="ml-auto text-[9.5px] text-[var(--text-muted)]">
          {{ t('task.itemsMore', { n: entry.items.length, m: entry.item_count }) }}
        </span>
      </div>
      <div class="max-h-[150px] space-y-1 overflow-y-auto pr-1">
        <div
          v-for="(item, index) in items"
          :key="index + item.name"
          class="item-row"
          :class="item.status === 'failed' ? 'item-failed' : ''"
        >
          <component
            :is="itemIcon(item.status)"
            :size="12"
            class="shrink-0"
            :class="item.status === 'failed' ? 'text-[var(--err)]' : item.status === 'skipped' ? 'text-[var(--warn)]' : 'text-[var(--ok)]'"
          />
          <span class="min-w-0 flex-1 truncate" :title="item.name">{{ item.name }}</span>
          <span class="max-w-[45%] truncate font-mono text-[9.5px] text-[var(--text-muted)]" :title="item.message">{{ item.message }}</span>
          <span v-if="item.elapsed_ms" class="shrink-0 font-mono text-[9.5px] text-[var(--text-muted)]">{{ item.elapsed_ms }}ms</span>
        </div>
        <p v-if="!items.length" class="py-2 text-center text-[10.5px] text-[var(--text-muted)]">{{ t('task.noMatchedItems') }}</p>
      </div>
    </div>

    <div class="mt-2 flex flex-wrap gap-x-3 gap-y-0.5 text-[9.5px] text-[var(--text-muted)]">
      <span v-if="entry.enqueued_at">{{ t('task.enqueuedAt') }} {{ clockTime(entry.enqueued_at) }}</span>
      <span v-if="entry.started_at">{{ t('task.startedAt') }} {{ clockTime(entry.started_at) }}</span>
      <span v-if="entry.finished_at">{{ t('task.finishedAt') }} {{ clockTime(entry.finished_at) }}</span>
      <span class="ml-auto truncate font-mono opacity-70" :title="entry.task_id">{{ entry.task_id.slice(0, 8) }}</span>
    </div>
  </article>
</template>

<style scoped>
.task-card {
  position: relative;
  overflow: hidden;
  border: 1px solid var(--border-soft);
  border-radius: 14px;
  background: linear-gradient(135deg, rgba(27, 36, 64, 0.72), rgba(19, 26, 46, 0.72));
  padding: 11px 13px;
  transition: border-color 180ms ease, transform 180ms ease, box-shadow 180ms ease;
}
.task-card::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 2px;
  background: var(--border-soft);
  transition: background 180ms ease;
}
.task-card:hover {
  border-color: rgba(139, 124, 246, 0.38);
  transform: translateY(-1px);
  box-shadow: 0 10px 24px -16px rgba(108, 92, 231, 0.9);
}
.card-running::before { background: var(--gradient-brand); }
.card-done::before { background: var(--ok); }
.card-failed::before { background: var(--err); }
.card-queued::before { background: var(--info); }
.card-cancelled::before, .card-removed::before { background: var(--warn); }

.lane-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  border-radius: 999px;
  padding: 3px 8px;
  font-size: 9.5px;
  font-weight: 600;
  letter-spacing: 0.02em;
}
.lane-work { color: var(--brand-300); background: rgba(34, 211, 238, 0.12); border: 1px solid rgba(34, 211, 238, 0.24); }
.lane-oo { color: var(--brand-400); background: rgba(139, 124, 246, 0.14); border: 1px solid rgba(139, 124, 246, 0.3); }

.queue-position {
  flex-shrink: 0;
  border-radius: 6px;
  background: rgba(56, 189, 248, 0.12);
  padding: 1px 5px;
  font-family: ui-monospace, monospace;
  font-size: 9.5px;
  color: var(--info);
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  border-radius: 999px;
  padding: 3px 8px;
  font-size: 9.5px;
  font-weight: 600;
  border: 1px solid transparent;
}
.badge-running { color: var(--brand-300); background: rgba(34, 211, 238, 0.1); border-color: rgba(34, 211, 238, 0.3); }
.badge-queued { color: var(--info); background: rgba(56, 189, 248, 0.1); border-color: rgba(56, 189, 248, 0.24); }
.badge-done { color: var(--ok); background: rgba(34, 197, 94, 0.1); border-color: rgba(34, 197, 94, 0.24); }
.badge-failed { color: var(--err); background: rgba(239, 68, 68, 0.1); border-color: rgba(239, 68, 68, 0.28); }
.badge-cancelled { color: var(--warn); background: rgba(245, 158, 11, 0.1); border-color: rgba(245, 158, 11, 0.24); }

.progress-track {
  height: 5px;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.12);
}
.progress-fill {
  height: 100%;
  border-radius: 999px;
  background: var(--gradient-brand);
  box-shadow: 0 0 10px rgba(34, 211, 238, 0.5);
  transition: width 300ms cubic-bezier(0.22, 1, 0.36, 1);
}

.stat-pill {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  border-radius: 6px;
  padding: 2px 6px;
  font-family: ui-monospace, monospace;
  font-size: 10px;
}
.stat-ok { color: var(--ok); background: rgba(34, 197, 94, 0.1); }
.stat-warn { color: var(--warn); background: rgba(245, 158, 11, 0.1); }
.stat-err { color: var(--err); background: rgba(239, 68, 68, 0.1); }

.error-strip {
  display: flex;
  gap: 6px;
  margin-top: 8px;
  border: 1px solid rgba(239, 68, 68, 0.32);
  border-radius: 8px;
  background: rgba(239, 68, 68, 0.07);
  padding: 6px 8px;
  font-size: 10.5px;
  color: var(--err);
}

.mini-button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: 1px solid var(--border-soft);
  border-radius: 7px;
  background: rgba(148, 163, 184, 0.05);
  padding: 4px 8px;
  font-size: 10px;
  color: var(--text-secondary);
  transition: all 150ms ease;
}
.mini-button:hover:not(:disabled) { border-color: rgba(139, 124, 246, 0.5); color: var(--brand-300); background: rgba(108, 92, 231, 0.1); }
.mini-danger:hover:not(:disabled) { border-color: rgba(239, 68, 68, 0.5); color: var(--err); background: rgba(239, 68, 68, 0.08); }
.mini-accent { color: var(--brand-300); border-color: rgba(34, 211, 238, 0.3); }
.mini-disabled { opacity: 0.42; cursor: not-allowed; }

.items-panel {
  margin-top: 10px;
  border-top: 1px solid var(--border-soft);
  padding-top: 9px;
}
.filter-chip {
  border: 1px solid transparent;
  border-radius: 6px;
  padding: 3px 8px;
  font-size: 9.5px;
  color: var(--text-muted);
  transition: all 150ms ease;
}
.filter-chip:hover { color: var(--text-primary); }
.filter-active { color: var(--brand-300); background: rgba(108, 92, 231, 0.14); border-color: rgba(139, 124, 246, 0.3); }

.item-row {
  display: flex;
  align-items: center;
  gap: 7px;
  border-radius: 6px;
  padding: 4px 6px;
  font-size: 10.5px;
  color: var(--text-secondary);
  transition: background 150ms ease;
}
.item-row:hover { background: rgba(148, 163, 184, 0.07); }
.item-failed { background: rgba(239, 68, 68, 0.09); color: var(--text-primary); }
.item-failed:hover { background: rgba(239, 68, 68, 0.15); }
</style>
