<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { FolderOpen, Settings, Square, Sparkles, Radio, ChevronRight, CheckCircle2, SkipForward, XCircle } from 'lucide-vue-next'
import { useTaskStore } from '@/stores/task'
import { useConfigStore } from '@/stores/config'
import { bindings } from '@/api/bindings'
import LanguageSwitcher from './LanguageSwitcher.vue'

const route = useRoute()
const router = useRouter()
const { t, te } = useI18n()
const tasks = useTaskStore()
const config = useConfigStore()

const title = computed(() => {
  const name = route.name as string | undefined
  if (name && te(`router.${name}`)) return t(`router.${name}`)
  return t('layout.workspace')
})

// 顶栏汇总两条通道：多于一个运行中任务时显示通道数，否则显示该任务的阶段。
const statusText = computed(() => {
  if (!tasks.running) return t('layout.idle')
  const list = tasks.activeTasks
  if (list.length > 1) return t('layout.multiRunning', { n: list.length })
  return list[0]?.stage || t('layout.running')
})
const statusTone = computed(() => (tasks.running ? 'active' : 'idle'))
const summary = computed(() => tasks.summary)

async function openDownloads() {
  await bindings.openDownloadDir()
}
</script>

<template>
  <header class="fixed left-0 right-0 top-0 z-30 flex h-16 items-center gap-4 border-b border-[var(--border-soft)] bg-[rgba(11,16,32,0.9)] px-5 backdrop-blur-xl">
    <div class="flex min-w-0 items-center gap-2">
      <Sparkles :size="16" class="shrink-0 text-[var(--brand-300)]" />
      <span class="truncate text-[15px] font-semibold">{{ title }}</span>
      <ChevronRight :size="14" class="hidden text-[var(--text-muted)] sm:block" />
      <span class="hidden truncate text-[12px] text-[var(--text-muted)] sm:block">{{ t('layout.console') }}</span>
    </div>

    <div class="ml-auto flex items-center gap-3">
      <div
        v-if="summary.count > 0"
        class="hidden items-center gap-2.5 rounded-full border border-[var(--border-soft)] bg-[rgba(148,163,184,0.05)] px-3 py-1.5 text-[11.5px] md:flex"
        :title="t('task.donutTitle')"
      >
        <span class="font-mono text-[var(--brand-300)]">{{ summary.current }}/{{ summary.total }}</span>
        <span class="flex items-center gap-1 text-[var(--ok)]"><CheckCircle2 :size="12" /> {{ summary.success }}</span>
        <span class="flex items-center gap-1 text-[var(--warn)]"><SkipForward :size="12" /> {{ summary.skipped }}</span>
        <span class="flex items-center gap-1 text-[var(--err)]"><XCircle :size="12" /> {{ summary.failed }}</span>
        <span class="font-mono text-[var(--text-muted)]">{{ summary.percent }}%</span>
      </div>

      <div
        class="hidden items-center gap-2 rounded-full border px-3 py-1.5 text-[12px] md:flex"
        :class="statusTone === 'active'
          ? 'border-[rgba(34,211,238,0.35)] bg-[rgba(34,211,238,0.08)] text-[var(--brand-300)]'
          : 'border-[var(--border-soft)] bg-[rgba(148,163,184,0.05)] text-[var(--text-secondary)]'"
      >
        <Radio :size="13" :class="statusTone === 'active' ? 'animate-pulse-soft' : ''" />
        <span class="max-w-[180px] truncate font-medium">{{ statusText }}</span>
      </div>

      <button
        type="button"
        class="ghost-button flex h-9 items-center gap-2 px-3 text-[12px]"
        :disabled="!tasks.running"
        @click="tasks.cancelAll()"
      >
        <Square :size="13" /> {{ t('layout.cancelAll') }}
      </button>
      <button type="button" class="ghost-button flex h-9 items-center gap-2 px-3 text-[12px]" @click="openDownloads">
        <FolderOpen :size="14" /> {{ t('layout.downloadDir') }}
      </button>
      <button
        type="button"
        class="ghost-button flex h-9 w-9 items-center justify-center"
        :title="t('router.settings')"
        @click="router.push('/settings')"
      >
        <Settings :size="15" />
      </button>
      <div class="hidden items-center gap-2 border-l border-[var(--border-soft)] pl-3 text-[11px] text-[var(--text-muted)] lg:flex">
        <span class="h-1.5 w-1.5 rounded-full bg-[var(--ok)] shadow-[0_0_8px_#22C55E]" />
        {{ config.appInfo?.platform ?? t('layout.desktop') }}
      </div>
      <LanguageSwitcher />
    </div>
  </header>
</template>
