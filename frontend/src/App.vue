<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterView } from 'vue-router'
import { AlertCircle } from 'lucide-vue-next'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import AppTopBar from '@/components/layout/AppTopBar.vue'
import AppStatusBar from '@/components/layout/AppStatusBar.vue'
import GlobalLogPanel from '@/components/log/GlobalLogPanel.vue'
import { useConfigStore } from '@/stores/config'
import { useLogStore } from '@/stores/logs'
import { useTaskStore } from '@/stores/task'

const collapsed = ref(false)
const logs = useLogStore()
const tasks = useTaskStore()
const config = useConfigStore()
const initError = ref('')
const windowHeight = ref(window.innerHeight)

const sidebarWidth = computed(() => (collapsed.value ? 72 : 232))
const collapsedLogHeight = computed(() => (logs.expanded ? logs.panelHeight : 40))
const logPanelHeight = computed(() =>
  logs.fullscreen ? Math.max(220, windowHeight.value - 94) : collapsedLogHeight.value,
)

const mainStyle = computed(() => ({
  top: '64px',
  left: sidebarWidth.value + 'px',
  right: '0px',
  bottom: 30 + logPanelHeight.value + 'px',
  display: logs.fullscreen ? 'none' : 'block',
}))

function onResize() {
  windowHeight.value = window.innerHeight
}

onMounted(async () => {
  window.addEventListener('resize', onResize)
  logs.start()
  tasks.start()
  config.start()
  const results = await Promise.allSettled([
    config.refresh(),
    logs.refresh(),
    tasks.refresh(),
    tasks.refreshHistory(),
  ])
  const failed = results.filter((r) => r.status === 'rejected')
  if (failed.length > 0) {
    const reason = failed[0] as PromiseRejectedResult
    initError.value = String(reason.reason?.message ?? reason.reason)
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  logs.stop()
  tasks.stop()
  config.stop()
})
</script>

<template>
  <div class="h-full w-full overflow-hidden">
    <AppTopBar />
    <AppSidebar v-model:collapsed="collapsed" />
    <GlobalLogPanel :left="sidebarWidth" :height="logPanelHeight" />

    <main
      class="fixed z-10 overflow-y-auto overflow-x-hidden px-6 py-5 transition-[bottom] duration-150"
      :style="mainStyle"
    >
      <div
        v-if="initError"
        class="glass-panel mb-4 flex items-center gap-3 border-[rgba(239,68,68,0.35)] p-4 text-[13px] text-[var(--err)]"
      >
        <AlertCircle :size="16" />
        <span>Wails 绑定未就绪：{{ initError }}。请通过 wails dev 或构建后的应用打开界面。</span>
      </div>
      <RouterView v-slot="{ Component }">
        <Transition name="page" mode="out-in">
          <component :is="Component" class="page-enter" />
        </Transition>
      </RouterView>
    </main>

    <AppStatusBar />
  </div>
</template>

<style>
.page-enter-active,
.page-leave-active {
  transition: opacity 160ms ease, transform 160ms ease;
}
.page-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.page-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
