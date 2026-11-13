<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  Download, Music2, FolderOpen, Settings, ChevronLeft, ChevronRight,
  Activity, Disc3, Database,
} from 'lucide-vue-next'

const props = defineProps<{ collapsed: boolean }>()
const emit = defineEmits<{ (e: 'update:collapsed', value: boolean): void }>()

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const items = computed(() => [
  { path: '/download', label: t('router.download'), hint: t('router.hintDownload'), icon: Download },
  { path: '/oomusic', label: t('router.oomusic'), hint: t('router.hintOO'), icon: Music2 },
  { path: '/library', label: t('router.library'), hint: t('router.hintLibrary'), icon: FolderOpen },
  { path: '/cache', label: t('router.cache'), hint: t('router.hintCache'), icon: Database },
  { path: '/settings', label: t('router.settings'), hint: t('router.hintSettings'), icon: Settings },
])

const active = computed(() => route.path)
function navigate(path: string) {
  void router.push(path)
}
</script>

<template>
  <aside
    class="fixed bottom-[30px] left-0 top-16 z-20 flex flex-col border-r border-[var(--border-soft)] bg-[rgba(11,16,32,0.86)] backdrop-blur-xl transition-all duration-200"
    :style="{ width: props.collapsed ? '72px' : '232px' }"
  >
    <div class="flex h-[72px] shrink-0 items-center gap-3 px-4">
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl" style="background: var(--gradient-brand)">
        <Disc3 :size="22" class="text-white" />
      </div>
      <div v-if="!props.collapsed" class="min-w-0 animate-fade-up">
        <div class="truncate text-[15px] font-semibold tracking-tight">Music Tools</div>
        <div class="truncate text-[11px] text-[var(--text-muted)]">音乐下载与整理工作台</div>
      </div>
    </div>

    <div class="mx-4 mb-3 h-px bg-[var(--border-soft)]" />

    <nav class="flex-1 space-y-1 overflow-y-auto px-3 pb-3">
      <button
        v-for="item in items"
        :key="item.path"
        type="button"
        class="group relative flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left transition-all duration-150"
        :class="active === item.path
          ? 'bg-[rgba(108,92,231,0.16)] text-[var(--text-primary)]'
          : 'text-[var(--text-secondary)] hover:bg-[rgba(148,163,184,0.07)] hover:text-[var(--text-primary)]'"
        :title="props.collapsed ? item.label : ''"
        @click="navigate(item.path)"
      >
        <span
          v-if="active === item.path"
          class="absolute left-0 top-2 bottom-2 w-[3px] rounded-r-full"
          style="background: var(--gradient-brand)"
        />
        <component :is="item.icon" :size="18" class="shrink-0" :class="active === item.path ? 'text-[var(--brand-400)]' : ''" />
        <span v-if="!props.collapsed" class="min-w-0 flex-1">
          <span class="block truncate text-[13px] font-medium">{{ item.label }}</span>
          <span class="block truncate text-[11px] text-[var(--text-muted)]">{{ item.hint }}</span>
        </span>
      </button>
    </nav>

    <div v-if="!props.collapsed" class="mx-3 mb-3 rounded-xl border border-[var(--border-soft)] bg-[rgba(34,211,238,0.05)] p-3">
      <div class="mb-1.5 flex items-center gap-2 text-[11px] font-semibold text-[var(--brand-300)]">
        <Activity :size="13" /> {{ t('sidebar.consoleTitle') }}
      </div>
      <p class="text-[11px] leading-relaxed text-[var(--text-muted)]">
        {{ t('sidebar.consoleHint') }}
      </p>
    </div>

    <button
      type="button"
      class="flex h-11 shrink-0 items-center justify-center gap-2 border-t border-[var(--border-soft)] text-[12px] text-[var(--text-muted)] transition-colors hover:text-[var(--text-primary)]"
      @click="emit('update:collapsed', !props.collapsed)"
    >
      <ChevronLeft v-if="!props.collapsed" :size="15" />
      <ChevronRight v-else :size="15" />
      <span v-if="!props.collapsed">{{ t('sidebar.collapse') }}</span>
    </button>
  </aside>
</template>
