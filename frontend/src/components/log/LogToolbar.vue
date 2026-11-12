<script setup lang="ts">
import { ref } from 'vue'
import {
  Trash2, Download, FolderOpen, ArrowDownToLine, Maximize2, Minimize2,
  ChevronDown, ChevronUp, AlertCircle,
} from 'lucide-vue-next'
import { useLogStore } from '@/stores/logs'
import type { LogLevel } from '@/types'

const logs = useLogStore()
const notice = ref('')

const levels: { value: 'ALL' | LogLevel; label: string }[] = [
  { value: 'ALL', label: '全部' },
  { value: 'DEBUG', label: 'DEBUG' },
  { value: 'INFO', label: 'INFO' },
  { value: 'WARN', label: 'WARN' },
  { value: 'ERROR', label: 'ERROR' },
]

function flash(text: string) {
  notice.value = text
  window.setTimeout(() => { notice.value = '' }, 2600)
}

async function exportLogs() {
  const path = await logs.exportLogs()
  flash(path ? `已导出到 ${path.split(/[\\/]/).pop()}` : '已取消导出')
}

async function clear() {
  await logs.clear()
  flash('日志已清空')
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-2">
    <span v-if="notice" class="mr-1 text-[11px] text-[var(--ok)]">{{ notice }}</span>

    <select
      class="field-input h-7 w-[92px] px-2 py-0 text-[11px]"
      aria-label="日志等级过滤"
      :value="logs.levelFilter"
      @change="logs.setLevel(($event.target as HTMLSelectElement).value as 'ALL' | LogLevel)"
    >
      <option v-for="l in levels" :key="l.value" :value="l.value">{{ l.label }}</option>
    </select>

    <input
      type="text"
      class="field-input h-7 w-[150px] px-2 py-0 text-[11px]"
      placeholder="搜索日志关键字"
      :value="logs.keyword"
      @input="logs.keyword = ($event.target as HTMLInputElement).value"
    />

    <button
      type="button"
      class="ghost-button flex h-7 items-center gap-1 px-2 text-[11px]"
      :class="logs.autoScroll ? '!text-[var(--brand-300)]' : ''"
      :title="logs.autoScroll ? '自动滚动已开启' : '自动滚动已暂停'"
      @click="logs.autoScroll = !logs.autoScroll"
    >
      <ArrowDownToLine :size="13" /> 跟随
    </button>
    <button type="button" class="ghost-button flex h-7 items-center gap-1 px-2 text-[11px]" title="清空日志" @click="clear">
      <Trash2 :size="13" /> 清空
    </button>
    <button type="button" class="ghost-button flex h-7 items-center gap-1 px-2 text-[11px]" title="导出日志" @click="exportLogs">
      <Download :size="13" /> 导出
    </button>
    <button type="button" class="ghost-button flex h-7 items-center gap-1 px-2 text-[11px]" title="打开日志目录" @click="logs.openDir()">
      <FolderOpen :size="13" /> 目录
    </button>
    <button
      type="button"
      class="ghost-button flex h-7 w-7 items-center justify-center"
      :title="logs.fullscreen ? '退出全屏' : '全屏展开'"
      @click="logs.toggleFullscreen()"
    >
      <Minimize2 v-if="logs.fullscreen" :size="13" />
      <Maximize2 v-else :size="13" />
    </button>
    <button
      type="button"
      class="ghost-button flex h-7 w-7 items-center justify-center"
      :title="logs.expanded ? '折叠日志窗口' : '展开日志窗口'"
      @click="logs.setExpanded(!logs.expanded)"
    >
      <ChevronDown v-if="logs.expanded" :size="14" />
      <ChevronUp v-else :size="14" />
    </button>
    <span v-if="logs.unreadErrors > 0" class="flex items-center gap-1 rounded-full bg-[rgba(239,68,68,0.15)] px-2 py-0.5 text-[10px] text-[var(--err)]">
      <AlertCircle :size="11" /> {{ logs.unreadErrors }} 条新错误
    </span>
  </div>
</template>
