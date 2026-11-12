<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { SlidersHorizontal, ShieldCheck, HardDrive, Info } from 'lucide-vue-next'
import { useConfigStore } from '@/stores/config'
import ConfigForm from '@/components/config/ConfigForm.vue'

const { t } = useI18n()
const config = useConfigStore()

const anchors = computed(() => config.groups.map((g) => ({ key: g.key, label: g.label, count: g.fields.length })))

const summary = computed(() => {
  const v = config.values
  return [
    { label: t('settings.sumApi'), value: String(v.api?.base_url ?? '—'), icon: ShieldCheck },
    { label: t('settings.sumDownload'), value: String(v.download?.default_dir ?? '—'), icon: HardDrive },
    { label: t('settings.sumQuality'), value: String(v.download?.default_quality ?? '—'), icon: SlidersHorizontal },
    { label: t('settings.sumLog'), value: String(v.logging?.level ?? '—'), icon: Info },
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
        <h1>{{ t('router.settings') }}</h1>
        <p class="muted mt-1.5 text-[12.5px]">
          {{ t('settings.subtitle') }}
        </p>
      </div>
      <div class="ml-auto hidden rounded-xl border border-[var(--border-soft)] bg-[rgba(34,211,238,0.05)] px-4 py-2.5 text-[11.5px] text-[var(--text-secondary)] md:block">
        {{ t('settings.groupsSummary', { n: config.groups.length, m: config.groups.reduce((n, g) => n + g.fields.length, 0) }) }}
      </div>
    </header>

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-[168px_minmax(0,1fr)]">
      <aside class="hidden xl:block">
        <div class="glass-panel sticky top-1 p-3">
          <div class="mb-2 px-2 text-[11px] font-semibold uppercase tracking-wider text-[var(--text-muted)]">{{ t('settings.navGroups') }}</div>
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
            <div class="mb-2 px-2 text-[11px] font-semibold uppercase tracking-wider text-[var(--text-muted)]">{{ t('settings.activeNow') }}</div>
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
