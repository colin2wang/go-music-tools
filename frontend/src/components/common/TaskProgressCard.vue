<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ListChecks, Trash2, Square, Inbox, CheckCircle2, AlertCircle, Info,
} from 'lucide-vue-next'
import { useTaskStore } from '@/stores/task'
import TaskCard from './TaskCard.vue'

const { t } = useI18n()
const tasks = useTaskStore()

const noticeText = computed(() => {
  const n = tasks.notice
  if (!n) return ''
  return t(n.key, n.params ?? {})
})
</script>

<template>
  <div class="board">
    <div class="board-toolbar">
      <div class="flex items-center gap-2 text-[12px] font-semibold text-[var(--text-primary)]">
        <ListChecks :size="14" class="text-[var(--brand-300)]" />
        {{ t('task.boardTitle') }}
        <span class="active-chip">{{ t('task.activeSuffix', { n: tasks.activeCount }) }}</span>
      </div>
      <div class="ml-auto flex items-center gap-1.5">
        <button
          type="button"
          class="board-button"
          :disabled="!tasks.running"
          @click="tasks.cancelAll()"
        >
          <Square :size="10" /> {{ t('task.cancelAll') }}
        </button>
        <button
          type="button"
          class="board-button"
          :disabled="!tasks.queuedTasks.length"
          @click="tasks.clearQueue()"
        >
          <Trash2 :size="11" /> {{ t('task.clearQueue') }}
        </button>
      </div>
    </div>

    <Transition name="notice-fade">
      <div v-if="tasks.notice" class="notice-strip" :class="tasks.notice.ok ? 'notice-ok' : 'notice-err'">
        <component :is="tasks.notice.ok ? CheckCircle2 : AlertCircle" :size="12" class="shrink-0" />
        <span class="min-w-0 flex-1">{{ noticeText }}</span>
      </div>
    </Transition>

    <div v-if="!tasks.entries.length" class="empty-state">
      <div class="empty-orb"><Inbox :size="26" /></div>
      <p class="mt-3 text-[12.5px] font-medium text-[var(--text-secondary)]">{{ t('task.emptyTitle') }}</p>
      <p class="mt-1 max-w-[360px] text-[10.5px] leading-relaxed text-[var(--text-muted)]">{{ t('task.emptyHint') }}</p>
      <div class="mt-3 flex items-center gap-2 text-[9.5px] text-[var(--text-muted)]">
        <Info :size="10" /> {{ t('task.laneHint') }}
      </div>
    </div>

    <template v-else>
      <section v-if="tasks.activeTasks.length" class="board-section">
        <h3 class="section-heading">
          <span class="section-dot dot-running" />
          {{ t('task.sectionRunning', { n: tasks.activeTasks.length }) }}
          <span class="section-rule" />
        </h3>
        <TransitionGroup name="card" tag="div" class="card-grid">
          <TaskCard v-for="task in tasks.activeTasks" :key="task.task_id" :entry="task" />
        </TransitionGroup>
      </section>

      <section v-if="tasks.queuedTasks.length" class="board-section">
        <h3 class="section-heading">
          <span class="section-dot dot-queued" />
          {{ t('task.sectionQueued', { n: tasks.queuedTasks.length }) }}
          <span class="section-rule" />
        </h3>
        <TransitionGroup name="card" tag="div" class="card-grid">
          <TaskCard
            v-for="(task, index) in tasks.queuedTasks"
            :key="task.task_id"
            :entry="task"
            :position="index + 1"
          />
        </TransitionGroup>
      </section>

      <section v-if="tasks.finishedTasks.length" class="board-section">
        <h3 class="section-heading">
          <span class="section-dot dot-finished" />
          {{ t('task.sectionFinished', { n: tasks.finishedTasks.length }) }}
          <span class="section-rule" />
        </h3>
        <TransitionGroup name="card" tag="div" class="card-grid">
          <TaskCard v-for="task in tasks.finishedTasks" :key="task.task_id" :entry="task" />
        </TransitionGroup>
      </section>
    </template>
  </div>
</template>

<style scoped>
.board { display: flex; flex-direction: column; gap: 12px; }

.board-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  border: 1px solid var(--border-soft);
  border-radius: 12px;
  background: linear-gradient(120deg, rgba(108, 92, 231, 0.1), rgba(34, 211, 238, 0.04));
  padding: 8px 11px;
}
.active-chip {
  border-radius: 999px;
  background: rgba(34, 211, 238, 0.14);
  padding: 2px 8px;
  font-size: 9.5px;
  font-weight: 600;
  color: var(--brand-300);
}
.board-button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  background: rgba(148, 163, 184, 0.05);
  padding: 5px 10px;
  font-size: 10.5px;
  color: var(--text-secondary);
  transition: all 160ms ease;
}
.board-button:hover:not(:disabled) {
  border-color: rgba(139, 124, 246, 0.5);
  color: var(--brand-300);
  background: rgba(108, 92, 231, 0.12);
  transform: translateY(-1px);
}
.board-button:disabled { opacity: 0.38; cursor: not-allowed; }

.notice-strip {
  display: flex;
  align-items: center;
  gap: 7px;
  border: 1px solid transparent;
  border-radius: 9px;
  padding: 7px 10px;
  font-size: 10.5px;
}
.notice-ok { color: var(--ok); background: rgba(34, 197, 94, 0.09); border-color: rgba(34, 197, 94, 0.25); }
.notice-err { color: var(--err); background: rgba(239, 68, 68, 0.09); border-color: rgba(239, 68, 68, 0.28); }
.notice-fade-enter-active, .notice-fade-leave-active { transition: opacity 180ms ease, transform 180ms ease; }
.notice-fade-enter-from, .notice-fade-leave-to { opacity: 0; transform: translateY(-5px); }

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 150px;
  border: 1px dashed var(--border-soft);
  border-radius: 14px;
  background: radial-gradient(circle at 50% 0%, rgba(108, 92, 231, 0.1), transparent 70%);
  text-align: center;
}
.empty-orb {
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  border-radius: 16px;
  color: var(--brand-300);
  background: rgba(108, 92, 231, 0.12);
  border: 1px solid rgba(139, 124, 246, 0.28);
  box-shadow: 0 0 26px -8px rgba(108, 92, 231, 0.8);
  animation: float 3.2s ease-in-out infinite;
}
@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-5px); }
}

.board-section { display: flex; flex-direction: column; gap: 8px; }
.section-heading {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 10.5px;
  font-weight: 600;
  letter-spacing: 0.02em;
  color: var(--text-secondary);
}
.section-dot { width: 6px; height: 6px; border-radius: 999px; flex-shrink: 0; }
.dot-running { background: var(--brand-300); box-shadow: 0 0 9px rgba(34, 211, 238, 0.8); animation: pulse-dot 1.8s ease-in-out infinite; }
.dot-queued { background: var(--info); }
.dot-finished { background: var(--ok); }
@keyframes pulse-dot { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
.section-rule { flex: 1; height: 1px; background: linear-gradient(90deg, var(--border-soft), transparent); }

.card-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 8px;
}
@media (min-width: 1400px) {
  .card-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
.card-enter-active, .card-leave-active { transition: opacity 220ms ease, transform 220ms ease; }
.card-enter-from { opacity: 0; transform: translateY(8px) scale(0.985); }
.card-leave-to { opacity: 0; transform: translateY(-6px) scale(0.985); }
.card-move { transition: transform 220ms ease; }
</style>
