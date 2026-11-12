<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Search, Loader2, Music2, Download, Clock3, Disc3, AlertCircle, RefreshCw,
  ListMusic, Zap, CircleDollarSign, FileAudio, RotateCcw, Square,
} from 'lucide-vue-next'
import { bindings } from '@/api/bindings'
import { useConfigStore } from '@/stores/config'
import { useTaskStore } from '@/stores/task'
import QualitySelect from '@/components/common/QualitySelect.vue'
import TaskProgressCard from '@/components/common/TaskProgressCard.vue'
import { OO_QUALITY_LEVELS, type OODetail, type OOSong, type QualityOption } from '@/types'

const { t } = useI18n()
const config = useConfigStore()
const tasks = useTaskStore()
const keyword = ref('')
const searchedKeyword = ref('')
const results = ref<OOSong[]>([])
const activeMid = ref('')
const detail = ref<OODetail | null>(null)
const quality = ref('SQ')
const qualityTouched = ref(false)
const searching = ref(false)
const loadingDetail = ref(false)
const starting = ref(false)
const notice = ref<{ ok: boolean; text: string } | null>(null)
const elapsedMS = ref(0)
const searched = ref<string[]>([])
const cacheHit = ref(false)
const lyricOpen = ref(false)

const active = computed(() => results.value.find((song) => song.song_mid === activeMid.value) ?? null)
const available = computed(() =>
  new Map((detail.value?.quality_options ?? []).map((item): [string, QualityOption] => [item.quality, item])),
)
const firstAvailable = computed(() => detail.value?.quality_options?.[0]?.quality ?? '')
const qualityRows = computed(() => OO_QUALITY_LEVELS.map((level) => ({
  ...level, option: available.value.get(level.value) ?? null, picked: quality.value === level.value,
})))
const lyricLines = computed(() =>
  (detail.value?.lyric_text || detail.value?.lyric_lrc || '').split('\n').filter(Boolean).slice(0, lyricOpen.value ? 40 : 5),
)
const failedItems = computed(() => tasks.items.filter((item) => item.status === 'failed'))

watch(
  () => config.values.advanced?.oo_music_default_quality,
  (value) => { if (!qualityTouched.value && typeof value === 'string' && value) quality.value = value },
  { immediate: true },
)

function flash(ok: boolean, text: string) {
  notice.value = { ok, text }
  window.setTimeout(() => { if (notice.value?.text === text) notice.value = null }, 3400)
}

function fail(err: unknown): null {
  console.error(err)
  flash(false, err instanceof Error ? err.message : String(err))
  return null
}

function isPaid(song: OOSong): boolean {
  return !!song.pay && song.pay !== '0' && song.pay.toLowerCase() !== 'false'
}

async function search() {
  const value = keyword.value.trim()
  if (!value || searching.value) return
  searching.value = true
  const started = performance.now()
  cacheHit.value = searched.value.includes(value)
  const list = await bindings.ooSearch(value).catch(fail)
  elapsedMS.value = Math.round(performance.now() - started)
  searching.value = false
  if (!list) return
  results.value = list
  searchedKeyword.value = value
  if (!cacheHit.value) searched.value.push(value)
  activeMid.value = ''
  detail.value = null
  flash(true, t('oo.found', { n: list.length, hit: cacheHit.value ? t('oo.cacheHit') : '' }))
  if (list.length) await selectSong(list[0])
}

async function selectSong(song: OOSong) {
  activeMid.value = song.song_mid
  detail.value = null
  loadingDetail.value = true
  const info = await bindings.ooDetail(song.singer_name, song.song_mid).catch(fail)
  loadingDetail.value = false
  if (!info) return
  detail.value = info
  if (!qualityTouched.value && info.quality) quality.value = info.quality
}

async function refreshQualities() {
  if (!active.value) return
  loadingDetail.value = true
  const list = await bindings.ooQualities(active.value.singer_name, active.value.song_mid).catch(fail)
  loadingDetail.value = false
  if (!list || !detail.value) return
  detail.value = { ...detail.value, quality_options: list }
  flash(true, t('oo.detected', { n: list.length }))
}

async function downloadOne(song: OOSong | null = active.value) {
  if (!song) { flash(false, '请先选择一首歌曲'); return }
  starting.value = true
  const id = await bindings.ooDownload({
    keyword: song.singer_name, song_mid: song.song_mid, quality: quality.value,
  }).catch(fail)
  starting.value = false
  if (!id) return
  flash(true, t('oo.startedOne', { title: song.song_title }))
  await tasks.refresh()
}

async function downloadAll() {
  if (!searchedKeyword.value) { flash(false, t('oo.needKeyword')); return }
  starting.value = true
  const id = await bindings.ooBatchDownload(searchedKeyword.value, quality.value).catch(fail)
  starting.value = false
  if (!id) return
  flash(true, t('oo.startedAll', { n: results.value.length }))
  await tasks.refresh()
}

async function retry(name: string) {
  const song = results.value.find((item) => `${item.singer_name} - ${item.song_title}` === name)
  if (!song) { flash(false, t('oo.retryMissing')); return }
  await selectSong(song)
  await downloadOne(song)
}

function pickQuality(value: string) {
  qualityTouched.value = true
  quality.value = value
  if (!available.value.has(value) && firstAvailable.value) {
    flash(false, t('oo.fallbackNotice', { q: value, fb: firstAvailable.value }))
  }
}
</script>

<template>
  <div class="mx-auto max-w-[1320px] pb-8">
    <header class="mb-5 flex flex-wrap items-end gap-3">
      <div>
        <div class="mb-1 flex items-center gap-2 text-[11px] font-semibold uppercase tracking-widest text-[var(--brand-300)]">
          <Zap :size="13" /> OO music console
        </div>
        <h1>{{ t('router.oomusic') }}</h1>
        <p class="muted mt-1.5 text-[12.5px]">{{ t('oo.subtitle') }}</p>
      </div>
      <span class="ml-auto rounded-full border border-[var(--border-soft)] px-3 py-1.5 text-[11px] text-[var(--text-secondary)]">
        {{ t('oo.outputDir', { dir: config.values.advanced?.oo_music_download_dir ?? 'downloads/[OOMusic]' }) }}
      </span>
    </header>

    <section class="glass-panel mb-4 p-4">
      <div class="flex flex-wrap gap-2">
        <div class="relative min-w-[260px] flex-1">
          <Search :size="15" class="absolute left-3 top-3 text-[var(--text-muted)]" />
          <input v-model="keyword" type="text" class="field-input h-10 pl-9 text-[13px]"
            :placeholder="t('oo.placeholder')" @keyup.enter="search" />
        </div>
        <button type="button" class="brand-button flex h-10 items-center gap-2 px-5 text-[12.5px]"
          :disabled="searching" @click="search">
          <Loader2 v-if="searching" :size="15" class="animate-spin" /><Search v-else :size="15" /> {{ t('oo.search') }}
        </button>
      </div>
      <div class="mt-2.5 flex flex-wrap items-center gap-3 text-[11px] text-[var(--text-muted)]">
        <span class="flex items-center gap-1"><ListMusic :size="12" /> {{ t('search.count', { n: results.length }) }}</span>
        <span class="flex items-center gap-1"><Clock3 :size="12" /> {{ t('oo.elapsed', { n: elapsedMS }) }}</span>
        <span v-if="cacheHit" class="rounded-md bg-[rgba(34,197,94,0.12)] px-2 py-0.5 text-[var(--ok)]">{{ t('oo.cacheHit') }}</span>
        <span v-if="notice" :class="notice.ok ? 'text-[var(--ok)]' : 'text-[var(--err)]'">{{ notice.text }}</span>
      </div>
    </section>

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,300px)_minmax(0,1fr)_280px]">
      <section class="glass-panel overflow-hidden">
        <div class="flex items-center gap-2 border-b border-[var(--border-soft)] px-4 py-3 text-[12.5px] font-semibold">
          <Music2 :size="14" class="text-[var(--brand-300)]" /> {{ t('search.results') }}
          <span class="ml-auto font-mono text-[10.5px] font-normal text-[var(--text-muted)]">{{ searchedKeyword || t('oo.notSearched') }}</span>
        </div>
        <div v-if="!results.length" class="flex flex-col items-center gap-2 px-4 py-14 text-center">
          <Disc3 :size="28" class="text-[var(--text-muted)]" />
          <p class="text-[12px] text-[var(--text-secondary)]">{{ t('oo.emptyTitle') }}</p>
          <p class="text-[11px] leading-relaxed text-[var(--text-muted)]">{{ t('oo.emptyHint') }}</p>
        </div>
        <div v-else class="max-h-[520px] space-y-1.5 overflow-y-auto p-2.5">
          <button v-for="(song, index) in results" :key="song.song_mid" type="button"
            class="flex w-full items-center gap-2.5 rounded-xl border px-3 py-2.5 text-left transition-all duration-150"
            :class="activeMid === song.song_mid
              ? 'border-[var(--border-strong)] bg-[rgba(108,92,231,0.15)] shadow-glow'
              : 'border-transparent bg-[rgba(11,16,32,0.4)] hover:border-[var(--border-soft)]'"
            @click="selectSong(song)">
            <span class="w-5 shrink-0 font-mono text-[10.5px] text-[var(--text-muted)]">{{ String(index + 1).padStart(2, '0') }}</span>
            <div class="min-w-0 flex-1">
              <div class="truncate text-[12.5px] font-medium">{{ song.song_title || t('common.unnamedSong') }}</div>
              <div class="truncate text-[11px] text-[var(--text-muted)]">{{ song.singer_name || t('common.unknownArtist') }}</div>
            </div>
            <span v-if="isPaid(song)" class="flex shrink-0 items-center gap-1 rounded-md bg-[rgba(245,158,11,0.14)] px-1.5 py-0.5 text-[10px] text-[var(--warn)]">
              <CircleDollarSign :size="10" /> {{ t('oo.paid') }}
            </span>
          </button>
        </div>
      </section>

      <section class="glass-panel p-4">
        <div class="mb-3 flex items-center gap-2 text-[13px] font-semibold">
          <Disc3 :size="15" class="text-[var(--brand-300)]" /> {{ t('oo.detailTitle') }}
          <Loader2 v-if="loadingDetail" :size="13" class="ml-auto animate-spin text-[var(--text-muted)]" />
          <button v-if="active" type="button" class="ghost-button ml-auto flex h-7 items-center gap-1 px-2 text-[11px]" @click="refreshQualities">
            <RefreshCw :size="11" /> {{ t('oo.recheck') }}
          </button>
        </div>

        <template v-if="detail">
          <div class="flex gap-3">
            <img v-if="detail.cover_url" :src="detail.cover_url" alt="cover"
              class="h-24 w-24 shrink-0 rounded-xl object-cover" />
            <div v-else class="flex h-24 w-24 shrink-0 items-center justify-center rounded-xl bg-[rgba(148,163,184,0.08)] text-[var(--text-muted)]">
              <Music2 :size="24" />
            </div>
            <div class="min-w-0 flex-1 space-y-1 text-[11.5px]">
              <div class="truncate text-[15px] font-semibold" :title="detail.title">{{ detail.title || t('common.unnamedSong') }}</div>
              <div class="truncate text-[var(--text-secondary)]">{{ detail.artist || t('common.unknownArtist') }} · {{ detail.album || t('common.unknownAlbum') }}</div>
              <div class="flex flex-wrap gap-x-3 gap-y-1 pt-1 font-mono text-[10.5px] text-[var(--text-muted)]">
                <span class="flex items-center gap-1"><Clock3 :size="11" /> {{ detail.duration_str || '—' }}</span>
                <span class="flex items-center gap-1"><FileAudio :size="11" /> {{ detail.file_extension || '.m4a' }}</span>
                <span v-if="detail.vip && detail.vip !== '0'">VIP {{ detail.vip }}</span>
                <span v-if="detail.genre">{{ detail.genre }}</span>
                <span v-if="detail.language">{{ detail.language }}</span>
                <span v-if="detail.public_time">{{ detail.public_time }}</span>
              </div>
              <div class="truncate font-mono text-[10.5px] text-[var(--text-muted)]" :title="detail.song_mid">MID {{ detail.song_mid }}</div>
            </div>
          </div>

          <div class="mt-4 overflow-hidden rounded-xl border border-[var(--border-soft)]">
            <div class="grid grid-cols-[1fr_70px_80px_50px] gap-2 border-b border-[var(--border-soft)] bg-[rgba(11,16,32,0.4)] px-3 py-2 text-[10.5px] text-[var(--text-muted)]">
              <span>{{ t('oo.colQuality') }}</span><span>{{ t('oo.colKbps') }}</span><span>{{ t('oo.colSize') }}</span><span>{{ t('oo.colExt') }}</span>
            </div>
            <button v-for="row in qualityRows" :key="row.value" type="button"
              class="grid w-full grid-cols-[1fr_70px_80px_50px] gap-2 border-b border-[var(--border-soft)] px-3 py-2 text-left text-[11.5px] transition-colors last:border-b-0"
              :class="row.picked ? 'bg-[rgba(108,92,231,0.14)]' : 'hover:bg-[rgba(148,163,184,0.06)]'"
              :disabled="!row.option" @click="pickQuality(row.value)">
              <span :class="row.option ? 'font-medium' : 'text-[var(--text-muted)]'">
                {{ row.label }}
                <span v-if="!row.option" class="ml-1 text-[10px]">{{ t('oo.unavailable', { fallback: firstAvailable || t('common.none') }) }}</span>
              </span>
              <span class="text-[var(--text-secondary)]">{{ row.option?.kbps ? row.option.kbps + ' k' : '—' }}</span>
              <span class="text-[var(--text-secondary)]">{{ row.option?.size || '—' }}</span>
              <span class="font-mono text-[var(--text-muted)]">{{ row.option?.ext || '—' }}</span>
            </button>
          </div>

          <div v-if="lyricLines.length" class="mt-3 rounded-xl border border-[var(--border-soft)] px-3 py-2.5">
            <button type="button" class="flex w-full items-center gap-1.5 text-[11.5px] font-medium text-[var(--text-secondary)]" @click="lyricOpen = !lyricOpen">
              <FileAudio :size="12" /> {{ t('oo.lyricPreview') }}
              <span class="ml-auto text-[10.5px] text-[var(--brand-400)]">{{ lyricOpen ? t('oo.showLess') : t('oo.showMore') }}</span>
            </button>
            <div class="mt-1.5 space-y-0.5 font-mono text-[10.5px] text-[var(--text-muted)]">
              <div v-for="(line, index) in lyricLines" :key="index" class="truncate">{{ line }}</div>
            </div>
          </div>
        </template>
        <div v-else class="flex flex-col items-center gap-2 py-16 text-center">
          <AlertCircle :size="24" class="text-[var(--text-muted)]" />
          <p class="text-[12px] text-[var(--text-secondary)]">{{ loadingDetail ? t('oo.loadingDetail') : t('oo.pickHint') }}</p>
        </div>
      </section>

      <div class="space-y-4">
        <section class="glass-panel space-y-3 p-4">
          <div class="text-[13px] font-semibold">{{ t('oo.console') }}</div>
          <QualitySelect v-model="quality" source="oo" :label="t('oo.targetQuality')" @update:modelValue="qualityTouched = true" />
          <button type="button" class="brand-button flex h-10 w-full items-center justify-center gap-2 text-[12.5px]"
            :disabled="!active || starting || tasks.running" @click="downloadOne()">
            <Download :size="15" /> {{ t('oo.downloadOne') }}
          </button>
          <button type="button" class="ghost-button flex h-10 w-full items-center justify-center gap-2 text-[12.5px]"
            :disabled="!results.length || starting || tasks.running" @click="downloadAll">
            <ListMusic :size="15" /> {{ t('oo.downloadAll', { n: results.length }) }}
          </button>
          <button type="button" class="ghost-button flex h-9 w-full items-center justify-center gap-2 text-[11.5px] text-[var(--err)]"
            :disabled="!tasks.running" @click="tasks.cancel()">
            <Square :size="12" /> {{ t('oo.cancelCurrent') }}
          </button>
          <p class="text-[11px] leading-relaxed text-[var(--text-muted)]">
            {{ t('oo.skipHint') }}
          </p>
          <div class="grid grid-cols-3 gap-2 border-t border-[var(--border-soft)] pt-3 text-center">
            <div><div class="text-[16px] font-semibold text-[var(--ok)]">{{ tasks.progress?.success ?? 0 }}</div><div class="text-[10px] text-[var(--text-muted)]">{{ t('common.success') }}</div></div>
            <div><div class="text-[16px] font-semibold text-[var(--warn)]">{{ tasks.progress?.skipped ?? 0 }}</div><div class="text-[10px] text-[var(--text-muted)]">{{ t('common.skipped') }}</div></div>
            <div><div class="text-[16px] font-semibold text-[var(--err)]">{{ tasks.progress?.failed ?? 0 }}</div><div class="text-[10px] text-[var(--text-muted)]">{{ t('common.failed') }}</div></div>
          </div>
        </section>

        <section v-if="failedItems.length" class="glass-panel p-4">
          <div class="mb-2 flex items-center gap-2 text-[12px] font-semibold text-[var(--err)]">
            <AlertCircle :size="13" /> {{ t('oo.failedItems', { n: failedItems.length }) }}
          </div>
          <div class="max-h-[160px] space-y-1.5 overflow-y-auto">
            <div v-for="(item, index) in failedItems" :key="index + item.name" class="flex items-center gap-2 text-[11px]">
              <span class="min-w-0 flex-1 truncate" :title="item.message">{{ item.name }}</span>
              <button type="button" class="ghost-button flex h-6 shrink-0 items-center gap-1 px-2 text-[10.5px]" @click="retry(item.name)">
                <RotateCcw :size="10" /> {{ t('oo.retry') }}
              </button>
            </div>
          </div>
        </section>
      </div>
    </div>

    <div class="mt-4"><TaskProgressCard /></div>
  </div>
</template>
