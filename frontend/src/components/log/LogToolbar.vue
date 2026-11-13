<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Trash2, Download, FolderOpen, ArrowDownToLine, Maximize2, Minimize2,
  ChevronDown, ChevronUp, AlertCircle,
} from 'lucide-vue-next'
import { useLogStore } from '@/stores/logs'
import type { LogLevel } from '@/types'

const { t } = useI18n()
const logs = useLogStore()
const notice = ref('')

const levels: { value: 'ALL' | LogLevel; label: string }[] = [
  { value: 'ALL', label: '全部' },
  { value: 'DEBUG', label: 'DEBUG' },
  { value: 'INFO', label: 'INFO' },
  { value: 'WARN', label: 'WARN' },
  { value: 'ERROR', label: 'ERROR' },
]

const runtimeLevels = ['debug', 'info', 'warn', 'error'] as const

function flash(text: string) {
  notice.value = text
  window.setTimeout(() => { notice.value = '' }, 2600)
}

async function exportLogs() {
  const path = await logs.exportLogs()
  flash(path ? t('log.exported', { name: path.split(/[\\/]/).pop() }) : t('log.exportCancelled'))
}

async function clear() {
  await logs.clear()
  flash(t('log.cleared'))
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-2">
    <span v-if="notice" class="mr-1 text-[11px] text-[var(--ok)]">{{ notice }}</span>

    <select
      class="field-input h-7 w-[92px] px-2 py-0 text-[11px]"
      :aria-label="t('log.levelFilter')"
      :value="logs.levelFilter"
      @change="logs.setLevel(($event.target as HTMLSelectElement).value as 'ALL' | LogLevel)"
    >
      <option v-for="l in levels" :key="l.value" :value="l.value">{{ l.value === 'ALL' ? t('common.all') : l.label }}</option>
    </select>

    <select
      class="field-input h-7 w-[92px] px-2 py-0 text-[11px]"
      :title="t('log.runtimeLevelHint')"
      :aria-label="t('log.runtimeLevel')"
      :value="logs.runtimeLevel"
      @change="logs.setRuntimeLevel(($event.target as HTMLSelectElement).value as 'debug' | 'info' | 'warn' | 'error')"
    >
      <option v-for="l in runtimeLevels" :key="l" :value="l">{{ l.toUpperCase() }}</option>
    </select>

    <input
      type="text"
      class="field-input h-7 w-[150px] px-2 py-0 text-[11px]"
      :placeholder="t('log.searchPlaceholder')"
      :value="logs.keyword"
      @input="logs.keyword = ($event.target as HTMLInputElement).value"
    />

    <button
      type="button"
      class="ghost-button flex h-7 items-center gap-1 px-2 text-[11px]"
      :class="logs.autoScroll ? '!text-[var(--brand-300)]' : ''"
      :title="logs.autoScroll ? t('log.followOn') : t('log.followOff')"
      @click="logs.autoScroll = !logs.autoScroll"
    >
      <ArrowDownToLine :size="13" /> {{ t('log.follow') }}
    </button>
    <button type="button" class="ghost-button flex h-7 items-center gap-1 px-2 text-[11px]" :title="t('log.clear')" @click="clear">
      <Trash2 :size="13" /> {{ t('log.clear') }}
    </button>
    <button type="button" class="ghost-button flex h-7 items-center gap-1 px-2 text-[11px]" :title="t('log.export')" @click="exportLogs">
      <Download :size="13" /> {{ t('log.export') }}
    </button>
    <button type="button" class="ghost-button flex h-7 items-center gap-1 px-2 text-[11px]" :title="t('layout.openLogDir')" @click="logs.openDir()">
      <FolderOpen :size="13" /> {{ t('log.dir') }}
    </button>
    <button
      type="button"
      class="ghost-button flex h-7 w-7 items-center justify-center"
      :title="logs.fullscreen ? t('log.exitFullscreen') : t('log.fullscreen')"
      @click="logs.toggleFullscreen()"
    >
      <Minimize2 v-if="logs.fullscreen" :size="13" />
      <Maximize2 v-else :size="13" />
    </button>
    <button
      type="button"
      class="ghost-button flex h-7 w-7 items-center justify-center"
      :title="logs.expanded ? t('log.collapse') : t('log.expandWindow')"
      @click="logs.setExpanded(!logs.expanded)"
    >
      <ChevronDown v-if="logs.expanded" :size="14" />
      <ChevronUp v-else :size="14" />
    </button>
    <span v-if="logs.unreadErrors > 0" class="flex items-center gap-1 rounded-full bg-[rgba(239,68,68,0.15)] px-2 py-0.5 text-[10px] text-[var(--err)]">
      <AlertCircle :size="11" /> {{ t('log.newErrors', { n: logs.unreadErrors }) }}
    </span>
  </div>
</template>
