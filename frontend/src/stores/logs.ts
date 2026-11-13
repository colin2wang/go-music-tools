import { defineStore } from 'pinia'
import { bindings } from '@/api/bindings'
import { EventNames, on } from '@/api/events'
import { useConfigStore } from '@/stores/config'
import type { LogEntry, LogLevel } from '@/types'

const LEVEL_ORDER: Record<LogLevel, number> = { DEBUG: 0, INFO: 1, WARN: 2, ERROR: 3 }

export const useLogStore = defineStore('logs', {
  state: () => ({
    entries: [] as LogEntry[],
    levelFilter: 'ALL' as 'ALL' | LogLevel,
    runtimeLevel: 'info' as 'debug' | 'info' | 'warn' | 'error',
    keyword: '',
    autoScroll: true,
    expanded: true,
    panelHeight: 220,
    fullscreen: false,
    unreadErrors: 0,
    maxEntries: 5000,
    rate: 0,
    started: false,
    off: null as null | (() => void),
    rateMark: Date.now(),
    rateCount: 0,
  }),

  getters: {
    filtered(state): LogEntry[] {
      const kw = state.keyword.trim().toLowerCase()
      const min = state.levelFilter === 'ALL' ? -1 : LEVEL_ORDER[state.levelFilter]
      return state.entries.filter((entry) => {
        if (min >= 0 && (LEVEL_ORDER[entry.level] ?? 1) < min) return false
        if (kw) {
          const haystack = `${entry.message} ${entry.source} ${JSON.stringify(entry.fields ?? {})}`.toLowerCase()
          if (!haystack.includes(kw)) return false
        }
        return true
      })
    },
    errorCount(state): number {
      return state.entries.filter((entry) => entry.level === 'ERROR').length
    },
  },

  actions: {
    async refresh() {
      const history = await bindings.getRecentLogs()
      this.entries = history ?? []
      const cfg = useConfigStore()
      const level = cfg.values.logging?.level
      if (level === 'debug' || level === 'info' || level === 'warn' || level === 'error') {
        this.runtimeLevel = level
      }
    },

    start() {
      if (this.started) return
      this.off = on<LogEntry[]>(EventNames.LOG_BATCH, (batch) => this.append(batch ?? []))
      this.started = true
    },

    append(batch: LogEntry[]) {
      if (batch.length === 0) return
      this.entries.push(...batch)
      if (this.entries.length > this.maxEntries) {
        this.entries.splice(0, this.entries.length - this.maxEntries)
      }
      const errors = batch.filter((entry) => entry.level === 'ERROR').length
      if (!this.expanded && errors > 0) this.unreadErrors += errors

      this.rateCount += batch.length
      const now = Date.now()
      if (now - this.rateMark >= 1000) {
        this.rate = Math.round((this.rateCount * 1000) / (now - this.rateMark))
        this.rateMark = now
        this.rateCount = 0
      }
    },

    async clear() {
      await bindings.clearLogs()
      this.entries = []
      this.unreadErrors = 0
    },

    async exportLogs(): Promise<string> {
      return bindings.exportLogs()
    },

    async openDir() {
      await bindings.openLogDir()
    },

    setLevel(level: 'ALL' | LogLevel) {
      this.levelFilter = level
    },

    /** 运行时切换后端日志级别，立即生效（不写入 config.yml）。 */
    async setRuntimeLevel(level: 'debug' | 'info' | 'warn' | 'error') {
      const prev = this.runtimeLevel
      this.runtimeLevel = level
      try {
        await bindings.setLogLevel(level)
      } catch {
        this.runtimeLevel = prev
      }
    },

    setExpanded(value: boolean) {
      this.expanded = value
      if (value) this.unreadErrors = 0
    },

    setHeight(height: number) {
      this.panelHeight = Math.max(120, Math.min(height, 640))
    },

    toggleFullscreen() {
      this.fullscreen = !this.fullscreen
      if (this.fullscreen) this.expanded = true
    },

    stop() {
      if (this.off) this.off()
      this.off = null
      this.started = false
    },
  },
})
