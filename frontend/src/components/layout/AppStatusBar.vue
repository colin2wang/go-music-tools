<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { FileText, Database, AlertCircle, ChevronUp } from 'lucide-vue-next'
import { useConfigStore } from '@/stores/config'
import { useLogStore } from '@/stores/logs'
import { useTaskStore } from '@/stores/task'

const { t } = useI18n()
const config = useConfigStore()
const logs = useLogStore()
const tasks = useTaskStore()

const elapsed = computed(() => {
  if (!tasks.done) return ''
  const ms = tasks.done.elapsed_ms
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
})

function focusLogs() {
  logs.setExpanded(true)
  if (logs.errorCount > 0) logs.setLevel('ERROR')
}
</script>

<template>
  <footer class="fixed bottom-0 left-0 right-0 z-30 flex h-[30px] items-center gap-4 border-t border-[var(--border-soft)] bg-[rgba(7,11,22,0.94)] px-4 text-[11px] text-[var(--text-muted)] backdrop-blur-xl">
    <button type="button" class="flex min-w-0 items-center gap-1.5 hover:text-[var(--text-primary)]" :title="t('layout.openConfigFile')" @click="config.openFile()">
      <FileText :size="12" />
      <span class="truncate">{{ config.appInfo?.config_path ?? 'config.yml' }}</span>
    </button>
    <button type="button" class="hidden items-center gap-1.5 hover:text-[var(--text-primary)] md:flex" :title="t('layout.openLogDir')" @click="logs.openDir()">
      <Database :size="12" />
      <span class="truncate">{{ config.appInfo?.log_dir ?? 'logs/' }}</span>
    </button>
    <span v-if="elapsed" class="hidden text-[var(--text-secondary)] lg:inline">{{ t('layout.lastTask', { t: elapsed }) }}</span>

    <div class="ml-auto flex items-center gap-3">
      <button
        type="button"
        class="flex items-center gap-1.5 rounded px-2 py-0.5 transition-colors"
        :class="logs.errorCount > 0 ? 'bg-[rgba(239,68,68,0.12)] text-[var(--err)]' : 'hover:text-[var(--text-primary)]'"
        @click="focusLogs"
      >
        <AlertCircle :size="12" />
        {{ t('layout.errors', { n: logs.errorCount }) }}
        <ChevronUp v-if="!logs.expanded" :size="11" />
      </button>
      <span class="hidden sm:inline">v{{ config.appInfo?.version ?? '1.0.0' }}</span>
    </div>
  </footer>
</template>
