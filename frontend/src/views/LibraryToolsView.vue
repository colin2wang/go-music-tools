<script setup lang="ts">
import { computed, ref, watch, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Wrench, FileText, Image, Users, Database, Play, Square, FolderTree,
  ChevronDown, ChevronRight, AlertCircle, Clock3, CheckCircle2, XCircle,
  Loader2, ShieldAlert, FolderOpen,
} from 'lucide-vue-next'
import { bindings } from '@/api/bindings'
import { useConfigStore } from '@/stores/config'
import { useTaskStore } from '@/stores/task'
import FolderPicker from '@/components/common/FolderPicker.vue'
import TaskProgressCard from '@/components/common/TaskProgressCard.vue'
import { LIBRARY_TOOLS, type LibraryFixKind, type TaskItemResult } from '@/types'

type OptionKey = 'only_missing' | 'force' | 'recursive' | 'continue_on_error'

const { t } = useI18n()

const toolIcons: Record<LibraryFixKind, Component> = {
  lyrics: FileText, cover: Image, artists: Users, metadata: Database,
}
const toolRiskKey: Record<LibraryFixKind, string> = {
  lyrics: 'library.toolLyricsRisk',
  cover: 'library.toolCoverRisk',
  artists: 'library.toolArtistsRisk',
  metadata: 'library.toolMetadataRisk',
}
const optionFields: { key: OptionKey; labelKey: string; hintKey: string }[] = [
  { key: 'only_missing', labelKey: 'library.optOnlyMissing', hintKey: 'library.optOnlyMissingHint' },
  { key: 'force', labelKey: 'library.optForce', hintKey: 'library.optForceHint' },
  { key: 'recursive', labelKey: 'library.optRecursive', hintKey: 'library.optRecursiveHint' },
  { key: 'continue_on_error', labelKey: 'library.optContinue', hintKey: 'library.optContinueHint' },
]

const config = useConfigStore()
const tasks = useTaskStore()
const directory = ref('')
const kind = ref<LibraryFixKind>('lyrics')
const folders = ref<string[]>([])
const options = ref<Record<OptionKey, boolean>>({
  only_missing: true, force: false, recursive: false, continue_on_error: true,
})
const starting = ref(false)
const notice = ref<{ ok: boolean; text: string } | null>(null)
const expanded = ref<string | null>(null)

const currentTool = computed(() => LIBRARY_TOOLS.find((item) => item.kind === kind.value) ?? LIBRARY_TOOLS[0])
const rows = computed(() => tasks.items)
const previewRows = computed(() => folders.value.map((path) => path.split(/[\\/]/).pop() ?? path))

watch(
  () => config.appInfo?.download_dir,
  (value) => { if (!directory.value && typeof value === 'string') directory.value = value },
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

function statusMeta(status: string): { label: string; color: string; icon: Component } {
  if (status === 'failed') return { label: t('common.failed'), color: 'text-[var(--err)] bg-[rgba(239,68,68,0.12)]', icon: XCircle }
  if (status === 'skipped') return { label: t('common.skipped'), color: 'text-[var(--warn)] bg-[rgba(245,158,11,0.12)]', icon: AlertCircle }
  return { label: t('common.done'), color: 'text-[var(--ok)] bg-[rgba(34,197,94,0.12)]', icon: CheckCircle2 }
}

function elapsed(row: TaskItemResult): string {
  const ms = row.elapsed_ms ?? 0
  if (!ms) return '—'
  return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s`
}

function toggle(name: string) {
  expanded.value = expanded.value === name ? null : name
}

async function start() {
  if (!directory.value.trim()) { flash(false, t('library.needRoot')); return }
  if (!folders.value.length) { flash(false, t('library.noSubdirs')); return }
  starting.value = true
  expanded.value = null
  tasks.beginOptimistic(`library:${kind.value}`, folders.value.length)
  const id = await bindings.runLibraryFix({
    directory: directory.value.trim(), kind: kind.value, ...options.value,
  }).catch(fail)
  starting.value = false
  if (!id) { tasks.running = false; return }
  flash(true, t('library.started', { tool: currentTool.value.title, n: folders.value.length }))
  await tasks.refresh()
}

async function openDirectory() {
  if (!directory.value.trim()) return
  await bindings.openPath(directory.value.trim()).catch(fail)
}
</script>

<template>
  <div class="mx-auto max-w-[1320px] pb-8">
    <header class="mb-5 flex flex-wrap items-end gap-3">
      <div>
        <div class="mb-1 flex items-center gap-2 text-[11px] font-semibold uppercase tracking-widest text-[var(--brand-300)]">
          <Wrench :size="13" /> Library maintenance
        </div>
        <h1>{{ t('library.title') }}</h1>
        <p class="muted mt-1.5 text-[12.5px]">{{ t('library.subtitle') }}</p>
      </div>
      <button type="button" class="ghost-button ml-auto flex h-9 items-center gap-2 px-3 text-[12px]" :disabled="!directory" @click="openDirectory">
        <FolderOpen :size="14" /> {{ t('library.openDir') }}
      </button>
    </header>

    <FolderPicker v-model="directory" class="mb-4" @folders="folders = $event" />

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1fr)_320px]">
      <div class="space-y-4">
        <section class="glass-panel p-4">
          <div class="mb-3 flex items-center gap-2 text-[13px] font-semibold">
            <Wrench :size="14" class="text-[var(--brand-300)]" /> {{ t('library.chooseTool') }}
            <span class="ml-auto font-mono text-[10.5px] font-normal text-[var(--text-muted)]">{{ t('library.flows', { n: 4 }) }}</span>
          </div>
          <div class="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
            <button v-for="tool in LIBRARY_TOOLS" :key="tool.kind" type="button"
              class="flex items-start gap-3 rounded-xl border p-3 text-left transition-all duration-150"
              :class="kind === tool.kind
                ? 'border-[var(--border-strong)] bg-[rgba(108,92,231,0.13)] shadow-glow'
                : 'border-[var(--border-soft)] bg-[rgba(11,16,32,0.4)] hover:border-[rgba(148,163,184,0.3)]'"
              @click="kind = tool.kind">
              <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl"
                :class="kind === tool.kind ? 'text-white' : 'bg-[rgba(148,163,184,0.1)] text-[var(--text-secondary)]'"
                :style="kind === tool.kind ? { background: 'var(--gradient-brand)' } : {}">
                <component :is="toolIcons[tool.kind]" :size="17" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="text-[12.5px] font-semibold">{{ tool.title }}</div>
                <div class="mt-1 text-[11px] leading-relaxed text-[var(--text-muted)]">{{ tool.description }}</div>
                <div class="mt-2 flex items-start gap-1 text-[10.5px] leading-relaxed text-[var(--warn)]">
                  <ShieldAlert :size="11" class="mt-0.5 shrink-0" /> {{ t(toolRiskKey[tool.kind]) }}
                </div>
              </div>
            </button>
          </div>
        </section>

        <section class="glass-panel p-4">
          <div class="mb-3 flex items-center gap-2 text-[13px] font-semibold">
            <FolderTree :size="14" class="text-[var(--brand-300)]" /> {{ t('library.progress') }}
            <span class="ml-auto font-mono text-[10.5px] font-normal text-[var(--text-muted)]">
              {{ t('library.processed', { a: rows.length, b: folders.length }) }}
            </span>
          </div>

          <div v-if="!folders.length && !rows.length" class="flex flex-col items-center gap-2 py-10 text-center">
            <FolderTree :size="26" class="text-[var(--text-muted)]" />
            <p class="text-[12px] text-[var(--text-secondary)]">{{ t('library.previewHint') }}</p>
          </div>

          <div v-else class="max-h-[300px] overflow-y-auto rounded-xl border border-[var(--border-soft)]">
            <div class="grid grid-cols-[minmax(0,1fr)_64px_64px_28px] gap-2 border-b border-[var(--border-soft)] bg-[rgba(11,16,32,0.4)] px-3 py-2 text-[10.5px] text-[var(--text-muted)]">
              <span>{{ t('library.colDir') }}</span><span>{{ t('library.colStatus') }}</span><span>{{ t('library.colElapsed') }}</span><span />
            </div>
            <template v-if="rows.length">
              <template v-for="(row, index) in rows" :key="index + row.name">
                <button type="button" class="grid w-full grid-cols-[minmax(0,1fr)_64px_64px_28px] items-center gap-2 border-b border-[var(--border-soft)] px-3 py-2 text-left text-[11.5px] transition-colors hover:bg-[rgba(108,92,231,0.07)]" @click="toggle(row.name)">
                  <span class="truncate" :title="row.name">{{ row.name }}</span>
                  <span class="flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[10px]" :class="statusMeta(row.status).color">
                    <component :is="statusMeta(row.status).icon" :size="10" /> {{ statusMeta(row.status).label }}
                  </span>
                  <span class="flex items-center gap-1 font-mono text-[10.5px] text-[var(--text-muted)]"><Clock3 :size="10" /> {{ elapsed(row) }}</span>
                  <component :is="expanded === row.name ? ChevronDown : ChevronRight" :size="12" class="text-[var(--text-muted)]" />
                </button>
                <div v-if="expanded === row.name" class="border-b border-[var(--border-soft)] bg-[rgba(11,16,32,0.55)] px-3 py-2 font-mono text-[10.5px] leading-relaxed text-[var(--text-secondary)]">
                  {{ row.message || t('library.noNote') }}
                </div>
              </template>
            </template>
            <template v-else>
              <div v-for="(name, index) in previewRows" :key="index + name"
                class="grid grid-cols-[minmax(0,1fr)_64px_64px_28px] items-center gap-2 border-b border-[var(--border-soft)] px-3 py-2 text-[11.5px] last:border-b-0">
                <span class="truncate" :title="name">{{ name }}</span>
                <span class="rounded-md bg-[rgba(148,163,184,0.1)] px-1.5 py-0.5 text-center text-[10px] text-[var(--text-muted)]">{{ t('common.pending') }}</span>
                <span class="font-mono text-[10.5px] text-[var(--text-muted)]">—</span>
                <span />
              </div>
            </template>
          </div>
        </section>
      </div>

      <div class="space-y-4">
        <section class="glass-panel p-4">
          <div class="mb-3 text-[13px] font-semibold">{{ t('library.params') }}</div>
          <div class="space-y-2.5">
            <label v-for="field in optionFields" :key="field.key"
              class="flex cursor-pointer items-start gap-2.5 rounded-xl border border-[var(--border-soft)] bg-[rgba(11,16,32,0.4)] px-3 py-2.5 transition-colors hover:border-[var(--border-strong)]">
              <input v-model="options[field.key]" type="checkbox" class="mt-0.5 h-3.5 w-3.5 cursor-pointer accent-[#6C5CE7]" />
              <span class="min-w-0">
                <span class="block text-[12px] font-medium">{{ t(field.labelKey) }}</span>
                <span class="mt-0.5 block text-[10.5px] leading-relaxed text-[var(--text-muted)]">{{ t(field.hintKey) }}</span>
              </span>
            </label>
          </div>
          <div class="mt-4 space-y-2 border-t border-[var(--border-soft)] pt-4">
            <button type="button" class="brand-button flex h-10 w-full items-center justify-center gap-2 text-[12.5px]"
              :disabled="starting || tasks.running || !folders.length" @click="start">
              <Loader2 v-if="starting" :size="15" class="animate-spin" /><Play v-else :size="15" />
              {{ t('library.run', { tool: currentTool.title }) }}
            </button>
            <button type="button" class="ghost-button flex h-9 w-full items-center justify-center gap-2 text-[11.5px] text-[var(--err)]"
              :disabled="!tasks.running" @click="tasks.cancel()">
              <Square :size="12" /> {{ t('library.cancelTask') }}
            </button>
            <p v-if="notice" class="text-[11px] leading-relaxed" :class="notice.ok ? 'text-[var(--ok)]' : 'text-[var(--err)]'">{{ notice.text }}</p>
            <p v-else class="text-[11px] leading-relaxed text-[var(--text-muted)]">
              {{ t('library.hint') }}
            </p>
          </div>
        </section>

        <section class="glass-panel p-4">
          <div class="mb-2 text-[12.5px] font-semibold">{{ t('library.scope') }}</div>
          <div class="grid grid-cols-2 gap-2 text-[11px]">
            <div class="rounded-xl border border-[var(--border-soft)] p-2.5">
              <div class="text-[10.5px] text-[var(--text-muted)]">{{ t('library.dirs') }}</div>
              <div class="mt-0.5 text-[18px] font-semibold">{{ folders.length }}</div>
            </div>
            <div class="rounded-xl border border-[var(--border-soft)] p-2.5">
              <div class="text-[10.5px] text-[var(--text-muted)]">{{ t('library.failedDirs') }}</div>
              <div class="mt-0.5 text-[18px] font-semibold text-[var(--err)]">{{ tasks.progress?.failed ?? 0 }}</div>
            </div>
          </div>
        </section>
      </div>
    </div>

    <div class="mt-4"><TaskProgressCard /></div>
  </div>
</template>
