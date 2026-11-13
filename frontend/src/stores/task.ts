import { defineStore } from 'pinia'
import { bindings } from '@/api/bindings'
import { EventNames, on } from '@/api/events'
import type {
  LaneState, QueuedTask, TaskDone, TaskEntry, TaskItemEvent, TaskItemResult,
  TaskLane, TaskProgress, TaskQueueSnapshot, TaskRecord, TaskStatus,
} from '@/types'

export interface TaskSummary {
  current: number
  total: number
  success: number
  skipped: number
  failed: number
  remaining: number
  percent: number
  count: number
}

export interface TaskNotice {
  ok: boolean
  key: string
  params?: Record<string, string | number>
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

function isFinished(status: TaskStatus): boolean {
  return status !== 'running' && status !== 'queued'
}

function summarize(entries: TaskEntry[]): TaskSummary {
  const acc = { current: 0, total: 0, success: 0, skipped: 0, failed: 0 }
  for (const e of entries) {
    acc.current += e.current
    acc.total += e.total
    acc.success += e.success
    acc.skipped += e.skipped
    acc.failed += e.failed
  }
  return {
    ...acc,
    remaining: Math.max(0, acc.total - acc.current),
    percent: acc.total > 0 ? Math.min(100, Math.round((acc.current / acc.total) * 100)) : 0,
    count: entries.length,
  }
}

function entryProgress(e: TaskEntry): TaskProgress {
  return {
    task_id: e.task_id,
    kind: e.kind,
    lane: e.lane,
    stage: e.stage,
    status: e.status === 'running' ? 'running' : 'done',
    current_item: e.current_item,
    current: e.current,
    total: e.total,
    success: e.success,
    skipped: e.skipped,
    failed: e.failed,
    done: isFinished(e.status),
    cancelled: e.cancelled,
    error: e.error,
    enqueued_at: e.enqueued_at,
    started_at: e.started_at,
    elapsed_ms: e.elapsed_ms,
    can_retry: e.can_retry,
  }
}

export const useTaskStore = defineStore('task', {
  state: () => ({
    /** 任务注册表，与后端 Board/Snapshot 一致，最新提交在前。 */
    entries: [] as TaskEntry[],
    lanes: [] as LaneState[],
    // 以下字段为兼容既有页面保留，均由注册表派生或事件补齐。
    progress: null as TaskProgress | null,
    done: null as TaskDone | null,
    items: [] as TaskItemResult[],
    running: false,
    queue: [] as QueuedTask[],
    history: [] as TaskRecord[],
    notice: null as TaskNotice | null,
    started: false,
    off: [] as (() => void)[],
  }),

  getters: {
    activeTasks: (state): TaskEntry[] => state.entries.filter((e) => e.status === 'running'),
    queuedTasks: (state): TaskEntry[] => state.entries.filter((e) => e.status === 'queued'),
    finishedTasks: (state): TaskEntry[] => state.entries.filter((e) => isFinished(e.status)),
    activeCount: (state): number =>
      state.entries.filter((e) => e.status === 'running' || e.status === 'queued').length,
    summary: (state): TaskSummary => summarize(state.entries.filter((e) => e.status !== 'removed')),
    percent: (state): number => summarize(state.entries.filter((e) => e.status !== 'removed')).percent,
    hasTask: (state): boolean => state.entries.length > 0,
    lastFinished: (state): TaskEntry | null => state.entries.find((e) => isFinished(e.status)) ?? null,
    laneSummary: (state) => (lane: TaskLane): TaskSummary =>
      summarize(state.entries.filter((e) => e.lane === lane && e.status !== 'removed')),
    laneActive: (state) => (lane: TaskLane): TaskEntry | null =>
      state.entries.find((e) => e.lane === lane && e.status === 'running') ?? null,
    laneItems: (state) => (lane: TaskLane): TaskItemResult[] => {
      const e = state.entries.find((x) => x.lane === lane && x.status === 'running')
        ?? state.entries.find((x) => x.lane === lane)
      return e?.items ?? []
    },
  },

  actions: {
    start() {
      if (this.started) return
      this.off.push(
        on<TaskProgress>(EventNames.TASK_PROGRESS, (p) => this.applyProgress(p)),
        on<TaskItemEvent>(EventNames.TASK_ITEM, (e) => this.applyItem(e)),
        on<TaskDone>(EventNames.TASK_DONE, (d) => this.applyDone(d)),
        on<TaskQueueSnapshot>(EventNames.TASK_QUEUED, (s) => this.applyQueue(s)),
      )
      this.started = true
    },

    applyProgress(p: TaskProgress) {
      if (!p?.task_id) return
      const idx = this.entries.findIndex((e) => e.task_id === p.task_id)
      if (idx >= 0) {
        const prev = this.entries[idx]
        const status: TaskStatus = p.done
          ? (p.cancelled ? 'cancelled' : p.error ? 'failed' : 'done')
          : (prev.status === 'queued' ? 'running' : prev.status)
        this.entries[idx] = {
          ...prev,
          lane: p.lane ?? prev.lane,
          kind: p.kind || prev.kind,
          status,
          stage: p.stage ?? prev.stage,
          current_item: p.current_item ?? prev.current_item,
          current: p.current ?? prev.current,
          total: p.total ?? prev.total,
          success: p.success ?? prev.success,
          skipped: p.skipped ?? prev.skipped,
          failed: p.failed ?? prev.failed,
          cancelled: p.cancelled ?? prev.cancelled,
          error: p.error ?? prev.error,
          enqueued_at: p.enqueued_at ?? prev.enqueued_at,
          started_at: p.started_at ?? prev.started_at,
          elapsed_ms: p.elapsed_ms ?? prev.elapsed_ms,
          can_retry: p.can_retry ?? prev.can_retry,
        }
      } else {
        // 事件早于初始化到达时先放一个占位条目，后续快照会全量对齐。
        this.entries.unshift({
          task_id: p.task_id,
          kind: p.kind,
          lane: p.lane ?? 'workbench',
          status: p.done ? (p.cancelled ? 'cancelled' : p.error ? 'failed' : 'done') : 'running',
          stage: p.stage,
          enqueued_at: p.enqueued_at ?? 0,
          started_at: p.started_at ?? 0,
          finished_at: 0,
          elapsed_ms: p.elapsed_ms ?? 0,
          current_item: p.current_item,
          current: p.current,
          total: p.total,
          success: p.success,
          skipped: p.skipped,
          failed: p.failed,
          cancelled: p.cancelled,
          error: p.error,
          items: [],
          item_count: 0,
          can_retry: p.can_retry ?? false,
        })
      }
      this.progress = p
      this.syncDerived()
    },

    applyItem(e: TaskItemEvent) {
      if (!e?.item || !e.task_id) return
      const idx = this.entries.findIndex((x) => x.task_id === e.task_id)
      if (idx >= 0) {
        const prev = this.entries[idx]
        this.entries[idx] = {
          ...prev,
          items: [...prev.items, e.item],
          item_count: prev.item_count + 1,
        }
      }
      if (!this.progress?.task_id || this.progress.task_id === e.task_id) {
        this.items.push(e.item)
      }
    },

    applyDone(d: TaskDone) {
      if (!d?.task_id) return
      const idx = this.entries.findIndex((e) => e.task_id === d.task_id)
      if (idx >= 0) {
        const prev = this.entries[idx]
        const status: TaskStatus = d.cancelled
          ? 'cancelled'
          : (d.error || d.status === 'failed' ? 'failed' : 'done')
        this.entries[idx] = {
          ...prev,
          status,
          finished_at: d.finished_at ?? prev.finished_at,
          started_at: d.started_at ?? prev.started_at,
          elapsed_ms: d.elapsed_ms ?? prev.elapsed_ms,
          cancelled: d.cancelled,
          error: d.error ?? prev.error,
          items: d.items?.length ? [...d.items] : prev.items,
          item_count: d.items?.length ?? prev.item_count,
        }
      }
      this.done = d
      this.items = d.items ?? []
      void this.refreshHistory()
      this.syncDerived()
    },

    /** 后端在任何队列/注册表变化后都会发快照，这里做一次全量对齐。 */
    applyQueue(s: TaskQueueSnapshot | null) {
      if (!s) return
      if (s.tasks) this.entries = [...s.tasks]
      if (s.lanes) this.lanes = [...s.lanes]
      if (s.queued) this.queue = [...s.queued]
      if (s.current) this.progress = s.current
      this.syncDerived()
    },

    syncDerived() {
      this.running = this.activeTasks.length > 0
      const focus = this.activeTasks[0]
      if (focus) {
        this.items = [...focus.items]
        this.progress = entryProgress(focus)
      }
    },

    async refreshBoard() {
      const board = await bindings.getTaskBoard().catch(() => null)
      if (!board) return
      this.entries = board.tasks ?? []
      this.lanes = board.lanes ?? []
      this.queue = this.queuedTasks.map((e) => ({
        task_id: e.task_id, kind: e.kind, lane: e.lane,
        status: e.status, total: e.total, enqueued_at: e.enqueued_at,
      }))
      this.syncDerived()
    },

    async refresh() {
      await this.refreshBoard()
      const status = await bindings.getTaskStatus().catch(() => null)
      if (status?.task_id) this.progress = status
      const snapshot = await bindings.getTaskQueue().catch(() => null)
      if (snapshot?.queued) this.queue = snapshot.queued
    },

    async refreshHistory() {
      this.history = (await bindings.getTaskHistory(20)) ?? []
    },

    flash(ok: boolean, key: string, params?: Record<string, string | number>) {
      this.notice = { ok, key, params }
      window.setTimeout(() => {
        if (this.notice?.key === key) this.notice = null
      }, 3200)
    },

    /** 顶栏按钮：取消两条通道里全部运行中的任务。 */
    async cancelAll() {
      try {
        const ok = await bindings.cancelTask()
        this.flash(ok, ok ? 'task.feedbackCancelAll' : 'task.feedbackNoRunning')
        await this.refreshBoard()
      } catch (err) {
        this.flash(false, 'task.feedbackFailed', { msg: errorMessage(err) })
      }
    },

    /** 兼容旧入口，语义已改为取消全部。 */
    cancel() {
      return this.cancelAll()
    },

    async cancelTask(taskID: string, notify = true) {
      try {
        const ok = await bindings.cancelTaskById(taskID)
        if (notify) this.flash(ok, ok ? 'task.feedbackCancelOne' : 'task.feedbackNotFound')
        await this.refreshBoard()
      } catch (err) {
        if (notify) this.flash(false, 'task.feedbackFailed', { msg: errorMessage(err) })
      }
    },

    async cancelLane(lane: TaskLane) {
      const ids = this.entries
        .filter((e) => e.lane === lane && e.status === 'running')
        .map((e) => e.task_id)
      if (!ids.length) {
        this.flash(false, 'task.feedbackNoRunning')
        return
      }
      for (const id of ids) await this.cancelTask(id, false)
      this.flash(true, 'task.feedbackCancelLane')
    },

    async removeQueued(taskID: string) {
      try {
        const ok = await bindings.removeQueuedTask(taskID)
        this.flash(ok, ok ? 'task.feedbackRemoved' : 'task.feedbackNotFound')
        await this.refreshBoard()
      } catch (err) {
        this.flash(false, 'task.feedbackFailed', { msg: errorMessage(err) })
      }
    },

    async clearQueue() {
      try {
        const n = await bindings.clearTaskQueue()
        this.flash(true, n > 0 ? 'task.feedbackCleared' : 'task.feedbackEmptyQueue', { n })
        await this.refreshBoard()
      } catch (err) {
        this.flash(false, 'task.feedbackFailed', { msg: errorMessage(err) })
      }
    },

    async retryFailed(taskID: string) {
      try {
        await bindings.retryFailedItems(taskID)
        this.flash(true, 'task.feedbackRetry')
        await this.refreshBoard()
      } catch (err) {
        this.flash(false, 'task.feedbackFailed', { msg: errorMessage(err) })
      }
    },

    /** 兼容旧页面：只显示一个乐观占位进度，不写入注册表。 */
    beginOptimistic(kind: string, total: number) {
      this.done = null
      this.items = []
      this.progress = {
        task_id: '', kind, stage: '准备中', status: 'running',
        current_item: '', current: 0, total, success: 0, skipped: 0,
        failed: 0, done: false, cancelled: false, error: '',
      }
    },

    stop() {
      this.off.forEach((fn) => fn())
      this.off = []
      this.started = false
    },
  },
})
