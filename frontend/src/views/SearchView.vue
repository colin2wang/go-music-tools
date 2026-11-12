<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Search, Loader2, Music2, Clock3, Download, X, ListPlus, Disc3 } from 'lucide-vue-next'
import { bindings } from '@/api/bindings'
import { useConfigStore } from '@/stores/config'
import { useTaskStore } from '@/stores/task'
import QualitySelect from '@/components/common/QualitySelect.vue'
import TaskProgressCard from '@/components/common/TaskProgressCard.vue'
import type { SearchSong } from '@/types'

const { t } = useI18n()
const config = useConfigStore()
const tasks = useTaskStore()
const keyword = ref('')
const history = ref<string[]>([])
const results = ref<SearchSong[]>([])
const selected = ref<string[]>([])
const active = ref<SearchSong | null>(null)
const quality = ref('lossless')
const searching = ref(false)
const notice = ref('')

const selectedCount = computed(() => selected.value.length)

function flash(text: string) {
  notice.value = text
  window.setTimeout(() => { notice.value = '' }, 3000)
}

async function search() {
  const value = keyword.value.trim()
  if (!value) return
  searching.value = true
  results.value = (await bindings.searchSongs(value)) ?? []
  selected.value = []
  if (!history.value.includes(value)) history.value.unshift(value)
  history.value = history.value.slice(0, 8)
  flash(t('search.found', { n: results.value.length }))
  searching.value = false
}

function toggle(id: string) {
  const index = selected.value.indexOf(id)
  if (index >= 0) selected.value.splice(index, 1)
  else selected.value.push(id)
}

function toggleAll() {
  if (selected.value.length === results.value.length) selected.value = []
  else selected.value = results.value.map((item) => item.id)
}

function duration(seconds: number): string {
  if (!seconds) return '—'
  const ms = seconds > 10000 ? seconds : seconds * 1000
  const total = Math.floor(ms / 1000)
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`
}

async function downloadSelected() {
  if (!selected.value.length) {
    flash(t('search.needSelect'))
    return
  }
  await bindings.downloadSongs(selected.value, quality.value)
  flash(t('search.started', { n: selected.value.length }))
  await tasks.refresh()
}
</script>

<template>
  <div class="mx-auto max-w-[1320px] pb-8">
    <header class="mb-5">
      <div class="mb-1 flex items-center gap-2 text-[11px] font-semibold uppercase tracking-widest text-[var(--brand-300)]">
        <Search :size="13" /> Music search
      </div>
      <h1>{{ t('search.title') }}</h1>
      <p class="muted mt-1.5 text-[12.5px]">{{ t('search.subtitle') }}</p>
    </header>

    <section class="glass-panel mb-4 p-4">
      <div class="flex flex-wrap gap-2">
        <div class="relative min-w-[260px] flex-1">
          <Search :size="15" class="absolute left-3 top-3 text-[var(--text-muted)]" />
          <input
            v-model="keyword"
            type="text"
            class="field-input h-10 pl-9 text-[13px]"
            :placeholder="t('search.placeholder')"
            @keyup.enter="search"
          />
        </div>
        <button type="button" class="brand-button flex h-10 items-center gap-2 px-5 text-[12.5px]" :disabled="searching" @click="search">
          <Loader2 v-if="searching" :size="15" class="animate-spin" />
          <Search v-else :size="15" /> {{ t('search.btn') }}
        </button>
      </div>
      <div v-if="history.length" class="mt-3 flex flex-wrap items-center gap-1.5">
        <span class="mr-1 text-[11px] text-[var(--text-muted)]">{{ t('search.recent') }}</span>
        <button
          v-for="item in history"
          :key="item"
          type="button"
          class="rounded-md border border-[var(--border-soft)] px-2 py-1 text-[11px] text-[var(--text-secondary)] transition-colors hover:border-[var(--border-strong)] hover:text-[var(--text-primary)]"
          @click="keyword = item; void search()"
        >
          {{ item }}
        </button>
        <button type="button" class="ml-1 text-[11px] text-[var(--text-muted)] hover:text-[var(--err)]" @click="history = []">{{ t('search.clearHistory') }}</button>
      </div>
    </section>

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1fr)_320px]">
      <section class="glass-panel overflow-hidden">
        <div class="flex flex-wrap items-center gap-3 border-b border-[var(--border-soft)] px-4 py-3">
          <span class="text-[12.5px] font-semibold">{{ t('search.results') }}</span>
          <span class="rounded-full bg-[rgba(108,92,231,0.14)] px-2 py-0.5 text-[10.5px] text-[var(--brand-400)]">{{ t('search.count', { n: results.length }) }}</span>
          <span v-if="notice" class="text-[11px] text-[var(--ok)]">{{ notice }}</span>
          <button
            type="button"
            class="ghost-button ml-auto flex h-8 items-center gap-1.5 px-3 text-[11.5px]"
            :disabled="!results.length"
            @click="toggleAll"
          >
            <ListPlus :size="13" /> {{ selectedCount === results.length && results.length ? t('search.deselectAll') : t('search.selectAll') }}
          </button>
        </div>

        <div v-if="!results.length" class="flex flex-col items-center gap-2 py-14 text-center">
          <Music2 :size="28" class="text-[var(--text-muted)]" />
          <p class="text-[12.5px] text-[var(--text-secondary)]">{{ t('search.emptyTitle') }}</p>
          <p class="text-[11px] text-[var(--text-muted)]">{{ t('search.emptyHint') }}</p>
        </div>

        <div v-else class="max-h-[420px] overflow-y-auto">
          <table class="w-full text-left text-[11.5px]">
            <thead class="sticky top-0 bg-[rgba(19,26,46,0.96)] text-[var(--text-muted)]">
              <tr class="border-b border-[var(--border-soft)]">
                <th class="w-10 px-4 py-2.5 font-medium">{{ t('search.colSelect') }}</th>
                <th class="py-2.5 font-medium">{{ t('search.colSong') }}</th>
                <th class="py-2.5 font-medium">{{ t('search.colArtist') }}</th>
                <th class="py-2.5 font-medium">{{ t('search.colAlbum') }}</th>
                <th class="py-2.5 font-medium">{{ t('search.colDuration') }}</th>
                <th class="w-16 py-2.5 font-medium">{{ t('search.colDetail') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="song in results"
                :key="song.id"
                class="cursor-pointer border-b border-[var(--border-soft)] transition-colors last:border-b-0 hover:bg-[rgba(108,92,231,0.07)]"
                :class="active?.id === song.id ? 'bg-[rgba(108,92,231,0.1)]' : ''"
                @click="active = song"
              >
                <td class="px-4 py-2.5">
                  <input type="checkbox" class="h-3.5 w-3.5 cursor-pointer accent-[#6C5CE7]" :checked="selected.includes(song.id)" @click.stop="toggle(song.id)" />
                </td>
                <td class="max-w-[220px] truncate py-2.5 font-medium" :title="song.name">{{ song.name || t('common.unnamedSong') }}</td>
                <td class="max-w-[150px] truncate py-2.5 text-[var(--text-secondary)]" :title="song.artists">{{ song.artists || t('common.none') }}</td>
                <td class="max-w-[160px] truncate py-2.5 text-[var(--text-muted)]" :title="song.album">{{ song.album || t('common.none') }}</td>
                <td class="py-2.5 font-mono text-[var(--text-muted)]">{{ duration(song.duration) }}</td>
                <td class="py-2.5">
                  <button type="button" class="text-[var(--brand-400)] hover:text-[var(--brand-300)]" @click.stop="active = song">{{ t('common.view') }}</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <div class="space-y-4">
        <section class="glass-panel p-4">
          <div class="mb-3 flex items-center gap-2 text-[12.5px] font-semibold">
            <Disc3 :size="14" class="text-[var(--brand-300)]" /> {{ t('search.detail') }}
            <button v-if="active" type="button" class="ml-auto text-[var(--text-muted)] hover:text-[var(--text-primary)]" @click="active = null"><X :size="14" /></button>
          </div>
          <template v-if="active">
            <img v-if="active.pic" :src="active.pic" alt="专辑封面" class="mb-3 h-36 w-full rounded-xl object-cover" />
            <div v-else class="mb-3 flex h-36 w-full items-center justify-center rounded-xl bg-[rgba(148,163,184,0.08)] text-[var(--text-muted)]">
              <Music2 :size="26" />
            </div>
            <div class="space-y-1.5 text-[11.5px]">
              <div class="text-[14px] font-semibold">{{ active.name }}</div>
              <div class="text-[var(--text-secondary)]">{{ active.artists || t('common.unknownArtist') }}</div>
              <div class="text-[var(--text-muted)]">{{ active.album || t('common.unknownAlbum') }}</div>
              <div class="flex items-center gap-1 font-mono text-[var(--text-muted)]"><Clock3 :size="11" /> {{ duration(active.duration) }} · ID {{ active.id }}</div>
            </div>
          </template>
          <p v-else class="py-6 text-center text-[11.5px] text-[var(--text-muted)]">{{ t('search.detailHint') }}</p>
        </section>

        <section class="glass-panel space-y-3 p-4">
          <QualitySelect v-model="quality" :label="t('search.qualityLabel')" />
          <button type="button" class="brand-button flex h-10 w-full items-center justify-center gap-2 text-[12.5px]" :disabled="!selectedCount || tasks.running" @click="downloadSelected">
            <Download :size="15" /> {{ t('search.downloadSelected', { n: selectedCount }) }}
          </button>
          <p class="text-[11px] leading-relaxed text-[var(--text-muted)]">
            {{ t('search.hint', { q: config.values.download?.default_quality ?? 'lossless' }) }}
          </p>
        </section>
      </div>
    </div>

    <div class="mt-4"><TaskProgressCard /></div>
  </div>
</template>
