<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { CheckCircle2, XCircle, SkipForward, Square, Activity, Clock3 } from 'lucide-vue-next'
import { useTaskStore } from '@/stores/task'
import StatCard from './StatCard.vue'

const { t } = useI18n()
const tasks = useTaskStore()

const stage = computed(() => tasks.progress?.stage ?? t('task.none'))
const items = computed(() => tasks.items.slice(-40).reverse())
const elapsed = computed(() => {
  if (!tasks.done) return ''
  return tasks.done.elapsed_ms < 1000
    ? `${tasks.done.elapsed_ms}ms`
    : `${(tasks.done.elapsed_ms / 1000).toFixed(1)}s`
})

function statusIcon(status: string) {
  if (status === 'failed') return XCircle
  if (status === 'skipped') return SkipForward
  return CheckCircle2
}

function statusColor(status: string) {
  if (status === 'failed') return 'text-[var(--err)]'
  if (status === 'skipped') return 'text-[var(--warn)]'
  return 'text-[var(--ok)]'
}
</script>

<template>
  <div class="glass-panel p-5">
    <div class="mb-4 flex flex-wrap items-center gap-3">
      <div>
        <div class="flex items-center gap-2 text-[14px] font-semibold">
          <Activity :size="15" class="text-[var(--brand-300)]" /> {{ t('task.title') }}
        </div>
        <p class="mt-1 text-[11.5px] text-[var(--text-muted)]">{{ stage }}</p>
      </div>
      <div class="ml-auto flex items-center gap-2">
        <span v-if="elapsed" class="flex items-center gap-1 text-[11px] text-[var(--text-muted)]">
          <Clock3 :size="12" /> {{ elapsed }}
        </span>
        <button type="button" class="ghost-button flex h-8 items-center gap-1.5 px-3 text-[11.5px]" :disabled="!tasks.running" @click="tasks.cancel()">
          <Square :size="12" /> {{ t('task.cancel') }}
        </button>
      </div>
    </div>

    <div class="mb-2 flex items-center justify-between text-[11px] text-[var(--text-secondary)]">
      <span>{{ tasks.progress?.current_item || t('task.waiting') }}</span>
      <span>{{ tasks.progress?.current ?? 0 }} / {{ tasks.progress?.total || 0 }}</span>
    </div>
    <div class="h-2 overflow-hidden rounded-full bg-[rgba(148,163,184,0.12)]">
      <div
        class="h-full rounded-full transition-all duration-300"
        style="background: var(--gradient-brand)"
        :style="{ width: tasks.percent + '%' }"
      />
    </div>

    <div class="mt-4 grid grid-cols-2 gap-2.5 lg:grid-cols-4">
      <StatCard :label="t('common.success')" :value="tasks.progress?.success ?? 0" tone="ok" :icon="CheckCircle2" />
      <StatCard :label="t('common.skipped')" :value="tasks.progress?.skipped ?? 0" tone="warn" :icon="SkipForward" />
      <StatCard :label="t('common.failed')" :value="tasks.progress?.failed ?? 0" tone="err" :icon="XCircle" />
      <StatCard :label="t('task.percent')" :value="`${tasks.percent}%`" tone="brand" :icon="Activity" />
    </div>

    <div v-if="tasks.done?.error" class="mt-3 rounded-lg border border-[rgba(239,68,68,0.35)] bg-[rgba(239,68,68,0.07)] p-3 text-[11.5px] text-[var(--err)]">
      {{ tasks.done.error }}
    </div>

    <div v-if="items.length" class="mt-4 max-h-[210px] overflow-y-auto rounded-xl border border-[var(--border-soft)]">
      <div
        v-for="(item, index) in items"
        :key="index + item.name"
        class="flex items-center gap-2 border-b border-[var(--border-soft)] px-3 py-2 text-[11.5px] last:border-b-0"
      >
        <component :is="statusIcon(item.status)" :size="13" :class="statusColor(item.status)" />
        <span class="min-w-0 flex-1 truncate" :title="item.name">{{ item.name }}</span>
        <span class="max-w-[45%] truncate font-mono text-[10.5px] text-[var(--text-muted)]" :title="item.message">
          {{ item.message }}
        </span>
      </div>
    </div>
  </div>
</template>
