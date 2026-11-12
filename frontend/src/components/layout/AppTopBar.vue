<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { FolderOpen, Settings, Square, Sparkles, Radio, ChevronRight } from 'lucide-vue-next'
import { useTaskStore } from '@/stores/task'
import { useConfigStore } from '@/stores/config'
import { bindings } from '@/api/bindings'

const route = useRoute()
const router = useRouter()
const tasks = useTaskStore()
const config = useConfigStore()

const title = computed(() => (route.meta.title as string) ?? '工作台')
const statusText = computed(() => {
  const p = tasks.progress
  if (!p) return '空闲'
  if (p.cancelled) return '已取消'
  if (p.done) return p.error ? '执行失败' : '已完成'
  return p.stage || '运行中'
})
const statusTone = computed(() => {
  const p = tasks.progress
  if (!p || p.done) return 'idle'
  return 'active'
})

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
      <span class="hidden truncate text-[12px] text-[var(--text-muted)] sm:block">Music Tools 控制台</span>
    </div>

    <div class="ml-auto flex items-center gap-3">
      <div
        class="hidden items-center gap-2 rounded-full border px-3 py-1.5 text-[12px] md:flex"
        :class="statusTone === 'active'
          ? 'border-[rgba(34,211,238,0.35)] bg-[rgba(34,211,238,0.08)] text-[var(--brand-300)]'
          : 'border-[var(--border-soft)] bg-[rgba(148,163,184,0.05)] text-[var(--text-secondary)]'"
      >
        <Radio :size="13" :class="statusTone === 'active' ? 'animate-pulse-soft' : ''" />
        <span class="font-medium">{{ statusText }}</span>
        <span v-if="tasks.progress && tasks.progress.total > 0" class="text-[var(--text-muted)]">
          {{ tasks.progress.current }}/{{ tasks.progress.total }}
        </span>
      </div>

      <button
        type="button"
        class="ghost-button flex h-9 items-center gap-2 px-3 text-[12px]"
        :disabled="!tasks.running"
        @click="tasks.cancel()"
      >
        <Square :size="13" /> 取消任务
      </button>
      <button type="button" class="ghost-button flex h-9 items-center gap-2 px-3 text-[12px]" @click="openDownloads">
        <FolderOpen :size="14" /> 下载目录
      </button>
      <button
        type="button"
        class="ghost-button flex h-9 w-9 items-center justify-center"
        title="设置"
        @click="router.push('/settings')"
      >
        <Settings :size="15" />
      </button>
      <div class="hidden items-center gap-2 border-l border-[var(--border-soft)] pl-3 text-[11px] text-[var(--text-muted)] lg:flex">
        <span class="h-1.5 w-1.5 rounded-full bg-[var(--ok)] shadow-[0_0_8px_#22C55E]" />
        {{ config.appInfo?.platform ?? '桌面端' }}
      </div>
    </div>
  </header>
</template>
