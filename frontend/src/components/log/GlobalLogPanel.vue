<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal, Wifi, ListChecks, Square } from 'lucide-vue-next'
import { useLogStore } from '@/stores/logs'
import { useTaskStore } from '@/stores/task'
import LogToolbar from './LogToolbar.vue'
import LogRow from './LogRow.vue'
import TaskProgressCard from '@/components/common/TaskProgressCard.vue'

const { t } = useI18n()
const props = defineProps<{ left: number; height: number }>()
const logs = useLogStore()
const tasks = useTaskStore()
const ROW_HEIGHT = 24
const activeTab = ref<'tasks' | 'logs'>('logs')
const listRef = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const viewportHeight = ref(200)
let observer: ResizeObserver | null = null

const filtered = computed(() => logs.filtered)
const startIndex = computed(() => Math.max(0, Math.floor(scrollTop.value / ROW_HEIGHT) - 6))
const endIndex = computed(() => Math.min(
  filtered.value.length,
  startIndex.value + Math.ceil(viewportHeight.value / ROW_HEIGHT) + 14,
))
const visible = computed(() => filtered.value.slice(startIndex.value, endIndex.value))
const topSpace = computed(() => startIndex.value * ROW_HEIGHT)
const bottomSpace = computed(() => Math.max(0, (filtered.value.length - endIndex.value) * ROW_HEIGHT))

// 有任务在运行或排队时自动切到任务 Tab；用户手动选日志后以活跃任务状态为准。
watch(
  () => tasks.activeCount > 0,
  (active) => {
    if (active) activeTab.value = 'tasks'
  },
)

// 拖拽日志面板上边缘调整高度：面板底部固定，向上拖动使高度增加。
let dragging = false
let dragStartY = 0
let dragStartHeight = 0

function beginDrag(event: MouseEvent) {
  if (!logs.expanded || logs.fullscreen) return
  dragging = true
  dragStartY = event.clientY
  dragStartHeight = logs.panelHeight
  window.addEventListener('mousemove', onDrag)
  window.addEventListener('mouseup', endDrag)
}

function onDrag(event: MouseEvent) {
  if (!dragging) return
  logs.setHeight(dragStartHeight + (dragStartY - event.clientY))
}

function endDrag() {
  dragging = false
  window.removeEventListener('mousemove', onDrag)
  window.removeEventListener('mouseup', endDrag)
}

function onScroll(event: Event) {
  const el = event.target as HTMLElement
  scrollTop.value = el.scrollTop
  const distance = el.scrollHeight - el.scrollTop - el.clientHeight
  logs.autoScroll = distance < 40
}

watch(
  () => filtered.value.length,
  async () => {
    if (!logs.autoScroll) return
    await nextTick()
    const el = listRef.value
    if (el) {
      el.scrollTop = el.scrollHeight
      scrollTop.value = el.scrollTop
    }
  },
)

onMounted(() => {
  if (listRef.value && typeof ResizeObserver !== 'undefined') {
    observer = new ResizeObserver((entries) => {
      viewportHeight.value = entries[0]?.contentRect.height ?? 200
    })
    observer.observe(listRef.value)
  }
})

onBeforeUnmount(() => {
  observer?.disconnect()
  endDrag()
})
</script>

<template>
  <section
    class="fixed z-20 flex flex-col overflow-hidden border-t border-[var(--border-strong)] bg-[rgba(9,14,28,0.94)] backdrop-blur-xl transition-[height] duration-150"
    :style="{ bottom: '30px', left: props.left + 'px', right: '0px', height: props.height + 'px' }"
  >
    <div
      class="absolute left-0 right-0 top-0 h-1 cursor-ns-resize bg-transparent hover:bg-[rgba(139,124,246,0.5)]"
      :title="t('log.dragHint')"
      @mousedown="beginDrag"
    />

    <div class="flex h-[38px] shrink-0 items-center gap-3 border-b border-[var(--border-soft)] px-3">
      <div class="flex items-center gap-1">
        <button
          type="button"
          class="flex h-7 items-center gap-1.5 rounded-md px-2.5 text-[12px] font-medium transition-colors"
          :class="activeTab === 'tasks'
            ? 'bg-[rgba(108,92,231,0.18)] text-[var(--brand-300)]'
            : 'text-[var(--text-secondary)] hover:text-[var(--brand-300)]'"
          @click="activeTab = 'tasks'"
        >
          <ListChecks :size="13" /> {{ t('task.tabTitle') }}
          <span
            v-if="tasks.activeCount"
            class="rounded-full bg-[rgba(34,211,238,0.16)] px-1.5 py-0.5 text-[9.5px] text-[var(--brand-300)]"
          >
            {{ tasks.activeCount }}
          </span>
        </button>
        <button
          type="button"
          class="flex h-7 items-center gap-1.5 rounded-md px-2.5 text-[12px] font-medium transition-colors"
          :class="activeTab === 'logs'
            ? 'bg-[rgba(108,92,231,0.18)] text-[var(--brand-300)]'
            : 'text-[var(--text-secondary)] hover:text-[var(--brand-300)]'"
          @click="activeTab = 'logs'"
        >
          <Terminal :size="13" /> {{ t('log.tabTitle') }}
          <span class="rounded-full bg-[rgba(108,92,231,0.16)] px-2 py-0.5 text-[10px] text-[var(--brand-400)]">
            {{ t('log.lines', { n: logs.filtered.length }) }}
          </span>
          <span
            v-if="logs.errorCount"
            class="rounded-full bg-[rgba(239,68,68,0.14)] px-2 py-0.5 text-[10px] text-[var(--err)]"
          >
            {{ t('log.errorBadge', { n: logs.errorCount }) }}
          </span>
        </button>
      </div>

      <div class="ml-auto flex items-center gap-3">
        <span class="hidden items-center gap-1 text-[10px] text-[var(--text-muted)] sm:flex">
          <Wifi :size="11" /> {{ t('log.rate', { n: logs.rate }) }}
        </span>
        <template v-if="logs.expanded">
          <LogToolbar v-if="activeTab === 'logs'" />
          <button
            v-else
            type="button"
            class="ghost-button flex h-7 items-center gap-1.5 px-3 text-[11.5px]"
            :disabled="!tasks.running"
            @click="tasks.cancelAll()"
          >
            <Square :size="12" /> {{ t('task.cancelAll') }}
          </button>
        </template>
        <button
          v-else
          type="button"
          class="ghost-button flex h-7 items-center gap-1 px-2 text-[11px]"
          @click="logs.setExpanded(true)"
        >
          {{ t('log.expand') }}
        </button>
      </div>
    </div>

    <div v-if="logs.expanded" class="min-h-0 flex-1 overflow-hidden">
      <div v-show="activeTab === 'tasks'" class="h-full overflow-y-auto px-3 py-2.5">
        <TaskProgressCard />
      </div>
      <div
        v-show="activeTab === 'logs'"
        ref="listRef"
        class="h-full overflow-y-auto overflow-x-hidden py-1"
        @scroll="onScroll"
      >
        <div :style="{ height: topSpace + 'px' }" />
        <LogRow v-for="(entry, index) in visible" :key="startIndex + index + entry.time + entry.message" :entry="entry" />
        <div :style="{ height: bottomSpace + 'px' }" />
        <div v-if="filtered.length === 0" class="flex h-full items-center justify-center text-[12px] text-[var(--text-muted)]">
          {{ t('log.noMatch') }}
        </div>
      </div>
    </div>
  </section>
</template>
