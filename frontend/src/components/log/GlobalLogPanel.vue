<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Terminal, Wifi } from 'lucide-vue-next'
import { useLogStore } from '@/stores/logs'
import LogToolbar from './LogToolbar.vue'
import LogRow from './LogRow.vue'

const props = defineProps<{ left: number; height: number }>()
const logs = useLogStore()
const ROW_HEIGHT = 24
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
      title="拖动调整日志窗口高度"
      @mousedown="beginDrag"
    />

    <div class="flex h-[38px] shrink-0 items-center gap-3 border-b border-[var(--border-soft)] px-3">
      <div class="flex items-center gap-2">
        <Terminal :size="15" class="text-[var(--brand-400)]" />
        <span class="text-[12px] font-semibold">全局日志</span>
        <span class="rounded-full bg-[rgba(108,92,231,0.16)] px-2 py-0.5 text-[10px] text-[var(--brand-400)]">
          {{ logs.filtered.length }} 行
        </span>
        <span v-if="logs.errorCount" class="rounded-full bg-[rgba(239,68,68,0.14)] px-2 py-0.5 text-[10px] text-[var(--err)]">
          {{ logs.errorCount }} 错误
        </span>
      </div>
      <div class="ml-auto flex items-center gap-3">
        <span class="hidden items-center gap-1 text-[10px] text-[var(--text-muted)] sm:flex">
          <Wifi :size="11" /> {{ logs.rate }} 行/秒
        </span>
        <LogToolbar v-if="logs.expanded" />
        <button
          v-else
          type="button"
          class="ghost-button flex h-7 items-center gap-1 px-2 text-[11px]"
          @click="logs.setExpanded(true)"
        >
          展开日志
        </button>
      </div>
    </div>

    <div
      v-if="logs.expanded"
      ref="listRef"
      class="min-h-0 flex-1 overflow-y-auto overflow-x-hidden py-1"
      @scroll="onScroll"
    >
      <div :style="{ height: topSpace + 'px' }" />
      <LogRow v-for="(entry, index) in visible" :key="startIndex + index + entry.time + entry.message" :entry="entry" />
      <div :style="{ height: bottomSpace + 'px' }" />
      <div v-if="filtered.length === 0" class="flex h-full items-center justify-center text-[12px] text-[var(--text-muted)]">
        暂无匹配日志。下载或修补任务启动后，日志会实时出现在这里。
      </div>
    </div>
  </section>
</template>
