import { defineStore } from 'pinia'
import { bindings } from '@/api/bindings'
import { EventNames, on } from '@/api/events'
import type { TaskDone, TaskItemEvent, TaskItemResult, TaskProgress, TaskRecord } from '@/types'

export const useTaskStore = defineStore('task', {
  state: () => ({
    progress: null as TaskProgress | null,
    done: null as TaskDone | null,
    items: [] as TaskItemResult[],
    running: false,
    history: [] as TaskRecord[],
    started: false,
    off: [] as (() => void)[],
  }),

  getters: {
    percent(state): number {
      const p = state.progress
      if (!p || p.total <= 0) return 0
      return Math.min(100, Math.round((p.current / p.total) * 100))
    },
    hasTask(state): boolean {
      return state.progress !== null
    },
  },

  actions: {
    start() {
      if (this.started) return
      this.off.push(
        on<TaskProgress>(EventNames.TASK_PROGRESS, (p) => this.applyProgress(p)),
        on<TaskItemEvent>(EventNames.TASK_ITEM, (e) => this.applyItem(e)),
        on<TaskDone>(EventNames.TASK_DONE, (d) => this.applyDone(d)),
      )
      this.started = true
    },

    applyProgress(p: TaskProgress) {
      if (!p) return
      if (p.task_id && this.progress?.task_id !== p.task_id) {
        this.items = []
        this.done = null
      }
      this.progress = p
      this.running = !p.done
    },

    applyItem(e: TaskItemEvent) {
      if (!e?.item) return
      if (e.task_id && this.progress?.task_id && e.task_id !== this.progress.task_id) return
      this.items.push(e.item)
    },

    applyDone(d: TaskDone) {
      if (!d) return
      this.done = d
      this.items = d.items ?? []
      this.running = false
      void this.refreshHistory()
    },

    async refresh() {
      this.progress = await bindings.getTaskStatus()
      if (this.progress && !this.progress.done) this.running = true
    },

    async refreshHistory() {
      this.history = (await bindings.getTaskHistory(20)) ?? []
    },

    async cancel() {
      await bindings.cancelTask()
    },

    beginOptimistic(kind: string, total: number) {
      this.done = null
      this.items = []
      this.running = true
      this.progress = {
        task_id: '',
        kind,
        stage: '准备中',
        current_item: '',
        current: 0,
        total,
        success: 0,
        skipped: 0,
        failed: 0,
        done: false,
        cancelled: false,
        error: '',
      }
    },

    stop() {
      this.off.forEach((fn) => fn())
      this.off = []
      this.started = false
    },
  },
})
