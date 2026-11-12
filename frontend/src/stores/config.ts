import { defineStore } from 'pinia'
import { bindings } from '@/api/bindings'
import { EventNames, on } from '@/api/events'
import type { AppInfo, ConfigGroup, ConfigValues } from '@/types'

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

export const useConfigStore = defineStore('config', {
  state: () => ({
    groups: [] as ConfigGroup[],
    values: {} as ConfigValues,
    original: {} as ConfigValues,
    errors: {} as Record<string, string>,
    appInfo: null as AppInfo | null,
    saving: false,
    loaded: false,
    started: false,
    off: null as null | (() => void),
  }),

  getters: {
    dirty(state): boolean {
      return JSON.stringify(state.values) !== JSON.stringify(state.original)
    },
    errorCount(state): number {
      return Object.keys(state.errors).length
    },
    fieldValue(state) {
      return (group: string, key: string): string | number | boolean => {
        return state.values[group]?.[key] ?? ''
      }
    },
  },

  actions: {
    start() {
      if (this.started) return
      this.off = on<ConfigValues>(EventNames.CONFIG_CHANGED, (values) => {
        if (!this.dirty) {
          this.values = clone(values)
          this.original = clone(values)
        }
      })
      this.started = true
    },

    async refresh() {
      const [groups, values, info] = await Promise.all([
        bindings.getConfigSchema(),
        bindings.getConfig(),
        bindings.getAppInfo(),
      ])
      this.groups = groups ?? []
      this.values = clone(values ?? {})
      this.original = clone(values ?? {})
      this.appInfo = info ?? null
      this.errors = {}
      this.loaded = true
    },

    setField(group: string, key: string, value: string | number | boolean) {
      if (!this.values[group]) this.values[group] = {}
      this.values[group][key] = value
      const path = `${group}.${key}`
      if (this.errors[path]) {
        const next = { ...this.errors }
        delete next[path]
        this.errors = next
      }
    },

    async validate(): Promise<boolean> {
      this.errors = (await bindings.validateConfig(this.values)) ?? {}
      return this.errorCount === 0
    },

    async save(): Promise<boolean> {
      this.saving = true
      const valid = await this.validate()
      if (!valid) {
        this.saving = false
        return false
      }
      await bindings.saveConfig(this.values)
      await this.refresh()
      this.saving = false
      return true
    },

    async reload() {
      await bindings.reloadConfig()
      await this.refresh()
    },

    async reset() {
      await bindings.resetConfig()
      await this.refresh()
    },

    async openFile() {
      await bindings.openConfigFile()
    },

    stop() {
      if (this.off) this.off()
      this.off = null
      this.started = false
    },
  },
})
