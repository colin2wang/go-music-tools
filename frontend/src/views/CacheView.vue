<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Database, Loader2, RefreshCw, Trash2, Eraser, Braces } from 'lucide-vue-next'
import { bindings } from '@/api/bindings'
import JsonDetailDialog from '@/components/JsonDetailDialog.vue'
import type { CacheDetail, CacheEntry, CacheRegionStat } from '@/types'

const { t } = useI18n()
const stats = ref<CacheRegionStat[]>([])
const activeRegion = ref('')
const entries = ref<CacheEntry[]>([])
const loading = ref(false)
const notice = ref<{ ok: boolean; text: string } | null>(null)
const detailDialog = ref<{ title: string; content: string } | null>(null)
const detailLoading = ref(false)

const activeLabel = computed(() => regionLabel(activeRegion.value))

const REGION_KEYS: Record<string, string> = {
  prefs: 'cache.regionPrefs',
  music_info: 'cache.regionInfo',
  music_url: 'cache.regionUrl',
  netease_song: 'cache.regionNetEase',
  oo_detail: 'cache.regionOODetail',
  oo_search: 'cache.regionOO',
}

function regionLabel(region: string): string {
  const key = REGION_KEYS[region]
  return key ? t(key) : region
}

function flash(ok: boolean, text: string) {
  notice.value = { ok, text }
  window.setTimeout(() => { if (notice.value?.text === text) notice.value = null }, 3000)
}

async function refreshStats() {
  stats.value = (await bindings.getCacheStats()) ?? []
  if (!activeRegion.value && stats.value.length) await selectRegion(stats.value[0].region)
}

async function selectRegion(region: string) {
  activeRegion.value = region
  loading.value = true
  entries.value = (await bindings.listCache(region)) ?? []
  loading.value = false
}

async function removeEntry(key: string) {
  await bindings.clearCache(activeRegion.value, key)
  flash(true, t('cache.deleted', { key: key.slice(0, 40) }))
  await selectRegion(activeRegion.value)
  await refreshStats()
}

async function clearRegion() {
  if (!activeRegion.value) return
  await bindings.clearCache(activeRegion.value, '')
  flash(true, t('cache.cleared', { region: activeLabel.value }))
  entries.value = []
  await refreshStats()
}

async function showDetail(item: CacheEntry) {
  detailLoading.value = true
  detailDialog.value = { title: item.key, content: '' }
  try {
    const detail: CacheDetail = await bindings.getCacheDetail(activeRegion.value, item.key)
    // OO 详情缓存优先展示原始响应；其余缓存区展示格式化 JSON。
    const content = activeRegion.value === 'oo_detail'
      ? (detail.raw || detail.formatted)
      : detail.formatted
    detailDialog.value = { title: `${t('cache.jsonTitle')} · ${item.key}`, content }
  } catch (e) {
    detailDialog.value = { title: item.key, content: String(e) }
  } finally {
    detailLoading.value = false
  }
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  return `${(n / 1024).toFixed(1)} KB`
}

onMounted(refreshStats)
</script>

<template>
  <div class="mx-auto max-w-[1320px] pb-8">
    <header class="mb-5">
      <div class="mb-1 flex items-center gap-2 text-[11px] font-semibold uppercase tracking-widest text-[var(--brand-300)]">
        <Database :size="13" /> Cache inspector
      </div>
      <h1>{{ t('router.cache') }}</h1>
      <p class="muted mt-1.5 text-[12.5px]">
        {{ t('cache.subtitle') }}
      </p>
    </header>

    <div class="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
      <button
        v-for="stat in stats"
        :key="stat.region"
        type="button"
        class="glass-panel p-4 text-left transition-all duration-150"
        :class="activeRegion === stat.region ? 'border-[rgba(108,92,231,0.55)] bg-[rgba(108,92,231,0.12)]' : 'hover:border-[var(--border-strong)]'"
        @click="selectRegion(stat.region)"
      >
        <div class="text-[20px] font-semibold">{{ stat.count }}</div>
        <div class="mt-1 truncate text-[11.5px] text-[var(--text-secondary)]" :title="regionLabel(stat.region)">{{ regionLabel(stat.region) }}</div>
      </button>
    </div>

    <section class="glass-panel overflow-hidden">
      <div class="flex flex-wrap items-center gap-3 border-b border-[var(--border-soft)] px-4 py-3">
        <span class="text-[12.5px] font-semibold">{{ activeLabel }}</span>
        <span class="rounded-full bg-[rgba(108,92,231,0.14)] px-2 py-0.5 text-[10.5px] text-[var(--brand-400)]">{{ t('cache.count', { n: entries.length }) }}</span>
        <span v-if="notice" class="text-[11px]" :class="notice.ok ? 'text-[var(--ok)]' : 'text-[var(--err)]'">{{ notice.text }}</span>
        <div class="ml-auto flex items-center gap-2">
          <button type="button" class="ghost-button flex h-8 items-center gap-1.5 px-3 text-[11.5px]" @click="refreshStats(); activeRegion && selectRegion(activeRegion)">
            <RefreshCw :size="13" /> {{ t('common.refresh') }}
          </button>
          <button type="button" class="ghost-button flex h-8 items-center gap-1.5 px-3 text-[11.5px] text-[var(--err)]" :disabled="!entries.length" @click="clearRegion">
            <Eraser :size="13" /> {{ t('cache.clearAll') }}
          </button>
        </div>
      </div>

      <div v-if="loading" class="flex items-center justify-center gap-2 py-12 text-[12px] text-[var(--text-muted)]">
        <Loader2 :size="15" class="animate-spin" /> {{ t('common.loading') }}
      </div>

      <div v-else-if="!entries.length" class="flex flex-col items-center gap-2 py-12 text-center">
        <Database :size="26" class="text-[var(--text-muted)]" />
        <p class="text-[12.5px] text-[var(--text-secondary)]">{{ t('cache.emptyTitle') }}</p>
        <p class="text-[11px] text-[var(--text-muted)]">{{ t('cache.emptyHint') }}</p>
      </div>

      <table v-else class="w-full text-left text-[11.5px]">
        <thead class="bg-[rgba(19,26,46,0.96)] text-[var(--text-muted)]">
          <tr class="border-b border-[var(--border-soft)]">
            <th class="px-4 py-2.5 font-medium">{{ t('cache.colKey') }}</th>
            <th class="py-2.5 font-medium">{{ t('cache.colPreview') }}</th>
            <th class="w-24 py-2.5 font-medium">{{ t('cache.colSize') }}</th>
            <th class="w-36 py-2.5 font-medium">{{ t('cache.colAction') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in entries" :key="item.key" class="border-b border-[var(--border-soft)] last:border-b-0 hover:bg-[rgba(108,92,231,0.05)]">
            <td class="max-w-[280px] truncate px-4 py-2.5 font-mono" :title="item.key">{{ item.key }}</td>
            <td class="max-w-[420px] truncate py-2.5 text-[var(--text-secondary)]" :title="item.preview">{{ item.preview || '—' }}</td>
            <td class="py-2.5 font-mono text-[var(--text-muted)]">{{ formatBytes(item.bytes) }}</td>
            <td class="py-2.5">
              <div class="flex items-center gap-2.5">
                <button type="button" class="flex items-center gap-1 text-[var(--brand-300)] hover:opacity-80" @click="showDetail(item)">
                  <Braces :size="13" /> JSON
                </button>
                <button type="button" class="text-[var(--err)] hover:opacity-80" :title="t('cache.deleted', { key: '' })" @click="removeEntry(item.key)">
                  <Trash2 :size="13" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <Teleport to="body">
      <JsonDetailDialog
        v-if="detailDialog"
        :title="detailDialog.title"
        :content="detailDialog.content"
        :loading="detailLoading"
        @close="detailDialog = null"
      />
    </Teleport>
  </div>
</template>
