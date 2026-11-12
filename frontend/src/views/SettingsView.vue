<script setup lang="ts">
import { computed } from 'vue'
import { SlidersHorizontal, ShieldCheck, Database, HardDrive, Info } from 'lucide-vue-next'
import { useConfigStore } from '@/stores/config'
import ConfigForm from '@/components/config/ConfigForm.vue'

const config = useConfigStore()

const anchors = computed(() => config.groups.map((g) => ({ key: g.key, label: g.label, count: g.fields.length })))

const summary = computed(() => {
  const v = config.values
  return [
    { label: 'API 根地址', value: String(v.api?.base_url ?? '—'), icon: ShieldCheck },
    { label: '下载目录', value: String(v.download?.default_dir ?? '—'), icon: HardDrive },
    { label: '默认音质', value: String(v.download?.default_quality ?? '—'), icon: SlidersHorizontal },
    { label: '日志级别', value: String(v.logging?.level ?? '—'), icon: Info },
    { label: '数据库', value: config.appInfo?.database_path?.split(/[\\/]/).pop() ?? 'app.db', icon: Database },
  ]
})

function scrollTo(key: string) {
  const el = document.getElementById(`cfg-${key}`)
  if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
</script>

<template>
  <div class="mx-auto max-w-[1400px] pb-8">
    <header class="mb-5 flex flex-wrap items-end gap-3">
      <div>
        <div class="mb-1 flex items-center gap-2 text-[11px] font-semibold uppercase tracking-widest text-[var(--brand-300)]">
          <SlidersHorizontal :size="13" /> Configuration
        </div>
        <h1>设置</h1>
        <p class="muted mt-1.5 text-[12.5px]">
          全部选项保存在 config.yaml，表单由后端 schema 动态生成，保存后立即生效且无需重启。
        </p>
      </div>
      <div class="ml-auto hidden rounded-xl border border-[var(--border-soft)] bg-[rgba(34,211,238,0.05)] px-4 py-2.5 text-[11.5px] text-[var(--text-secondary)] md:block">
        共 {{ config.groups.length }} 个分组 · {{ config.groups.reduce((n, g) => n + g.fields.length, 0) }} 个生效字段
      </div>
    </header>

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-[168px_minmax(0,1fr)]">
      <aside class="hidden xl:block">
        <div class="glass-panel sticky top-1 p-3">
          <div class="mb-2 px-2 text-[11px] font-semibold uppercase tracking-wider text-[var(--text-muted)]">分组导航</div>
          <button
            v-for="item in anchors"
            :key="item.key"
            type="button"
            class="flex w-full items-center justify-between rounded-lg px-2.5 py-2 text-left text-[12px] text-[var(--text-secondary)] transition-all hover:bg-[rgba(108,92,231,0.12)] hover:text-[var(--text-primary)]"
            @click="scrollTo(item.key)"
          >
            <span>{{ item.label }}</span>
            <span class="font-mono text-[10px] text-[var(--text-muted)]">{{ item.count }}</span>
          </button>

          <div class="mt-4 border-t border-[var(--border-soft)] pt-3">
            <div class="mb-2 px-2 text-[11px] font-semibold uppercase tracking-wider text-[var(--text-muted)]">当前生效</div>
            <div v-for="row in summary" :key="row.label" class="mb-2 px-2">
              <div class="flex items-center gap-1.5 text-[10.5px] text-[var(--text-muted)]">
                <component :is="row.icon" :size="11" /> {{ row.label }}
              </div>
              <div class="mt-0.5 truncate font-mono text-[11px] text-[var(--text-primary)]" :title="row.value">{{ row.value }}</div>
            </div>
          </div>
        </div>
      </aside>

      <ConfigForm />
    </div>
  </div>
</template>
