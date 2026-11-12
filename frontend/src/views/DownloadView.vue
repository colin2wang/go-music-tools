<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Download, Disc3, ListMusic, Music2, Link2, FolderOpen, Play, Loader2, History } from 'lucide-vue-next'
import { bindings } from '@/api/bindings'
import { useConfigStore } from '@/stores/config'
import { useTaskStore } from '@/stores/task'
import QualitySelect from '@/components/common/QualitySelect.vue'
import IndexPicker from '@/components/common/IndexPicker.vue'
import TaskProgressCard from '@/components/common/TaskProgressCard.vue'
import type { DownloadRequest } from '@/types'

type Kind = DownloadRequest['kind']

const modes: { kind: Kind; title: string; description: string; icon: typeof Disc3; badge: string }[] = [
  { kind: 'song', title: '单曲下载', description: '按歌曲 ID 下载一首，自动写入标签、歌词与封面', icon: Music2, badge: 'SONG' },
  { kind: 'album', title: '整张专辑', description: '生成专辑目录、说明文件与高清封面，按曲目号排序', icon: Disc3, badge: 'ALBUM' },
  { kind: 'playlist', title: '整个歌单', description: '按歌单顺序批量下载，生成歌单说明与曲目清单', icon: ListMusic, badge: 'PLAYLIST' },
]

const config = useConfigStore()
const tasks = useTaskStore()
const kind = ref<Kind>('song')
const resourceId = ref('')
const quality = ref('lossless')
const indexRaw = ref('')
const indexes = ref<number[]>([])
const starting = ref(false)
const notice = ref<{ ok: boolean; text: string } | null>(null)

const needsIndexes = computed(() => kind.value !== 'song')
const currentMode = computed(() => modes.find((item) => item.kind === kind.value) ?? modes[0])

watch(
  () => config.values.download?.default_quality,
  (value) => { if (typeof value === 'string' && value) quality.value = value },
  { immediate: true },
)

function flash(ok: boolean, text: string) {
  notice.value = { ok, text }
  window.setTimeout(() => { notice.value = null }, 3200)
}

onMounted(async () => {
  const pref = await bindings.getLastPreference()
  if (pref.kind) kind.value = pref.kind
  if (pref.resource_id) resourceId.value = pref.resource_id
  if (pref.quality) quality.value = pref.quality
  if (pref.indexes?.length) {
    indexes.value = pref.indexes
    indexRaw.value = pref.indexes.join(' ')
  }
  await tasks.refreshHistory()
})

async function extractID() {
  const raw = resourceId.value.trim()
  if (!raw || /^\d+$/.test(raw)) return
  const id = await bindings.extractResourceID(raw)
  if (id && id !== raw) {
    resourceId.value = id
    flash(true, `已从分享链接提取 ID：${id}`)
  }
}

async function start() {
  if (!resourceId.value.trim()) {
    flash(false, '请填写歌曲、专辑或歌单 ID')
    return
  }
  starting.value = true
  const request: DownloadRequest = {
    kind: kind.value,
    resource_id: resourceId.value.trim(),
    quality: quality.value,
    indexes: needsIndexes.value ? indexes.value : [],
  }
  await bindings.download(request)
  flash(true, '下载任务已启动，进度与日志见下方实时窗口')
  starting.value = false
}

async function openDir() {
  await bindings.openDownloadDir()
}

function formatTime(seconds: number): string {
  if (!seconds) return '—'
  const date = new Date(seconds * 1000)
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div class="mx-auto max-w-[1320px] pb-8">
    <header class="mb-5 flex flex-wrap items-end gap-3">
      <div>
        <div class="mb-1 flex items-center gap-2 text-[11px] font-semibold uppercase tracking-widest text-[var(--brand-300)]">
          <Download :size="13" /> Download workspace
        </div>
        <h1>下载工作台</h1>
        <p class="muted mt-1.5 text-[12.5px]">
          支持 ID 或分享链接、7 档音质与指定曲目；下载过程自动整理目录、封面、歌词和音频标签。
        </p>
      </div>
      <button type="button" class="ghost-button ml-auto flex h-9 items-center gap-2 px-3 text-[12px]" @click="openDir">
        <FolderOpen :size="14" /> 打开下载目录
      </button>
    </header>

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,420px)_minmax(0,1fr)]">
      <div class="space-y-4">
        <section class="glass-panel p-5">
          <div class="mb-3 text-[12px] font-semibold text-[var(--text-secondary)]">选择下载方式</div>
          <div class="space-y-2">
            <button
              v-for="mode in modes"
              :key="mode.kind"
              type="button"
              class="flex w-full items-center gap-3 rounded-xl border p-3 text-left transition-all duration-150"
              :class="kind === mode.kind
                ? 'border-[var(--border-strong)] bg-[rgba(108,92,231,0.13)] shadow-glow'
                : 'border-[var(--border-soft)] bg-[rgba(11,16,32,0.45)] hover:border-[rgba(148,163,184,0.3)]'"
              @click="kind = mode.kind"
            >
              <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl" :class="kind === mode.kind ? 'text-white' : 'bg-[rgba(148,163,184,0.1)] text-[var(--text-secondary)]'" :style="kind === mode.kind ? { background: 'var(--gradient-brand)' } : {}">
                <component :is="mode.icon" :size="19" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2 text-[13px] font-semibold">
                  {{ mode.title }}
                  <span class="font-mono text-[9.5px] text-[var(--text-muted)]">{{ mode.badge }}</span>
                </div>
                <div class="mt-0.5 text-[11px] leading-relaxed text-[var(--text-muted)]">{{ mode.description }}</div>
              </div>
            </button>
          </div>
        </section>

        <section class="glass-panel space-y-4 p-5">
          <div>
            <label class="mb-1.5 flex items-center gap-1.5 text-[12px] font-medium text-[var(--text-secondary)]">
              <Link2 :size="13" /> {{ currentMode.title }} ID 或分享链接
            </label>
            <input
              v-model="resourceId"
              type="text"
              class="field-input h-10 font-mono text-[12.5px]"
              placeholder="例如 3383993462 或 https://music.163.com/song?id=3383993462"
              @blur="extractID"
            />
          </div>
          <QualitySelect v-model="quality" label="音质" />
          <IndexPicker v-if="needsIndexes" v-model="indexRaw" @change="(v) => (indexes = v)" />

          <div class="flex items-center gap-3 border-t border-[var(--border-soft)] pt-4">
            <button type="button" class="brand-button flex h-10 flex-1 items-center justify-center gap-2 text-[13px]" :disabled="starting || tasks.running" @click="start">
              <Loader2 v-if="starting" :size="15" class="animate-spin" />
              <Play v-else :size="15" />
              开始下载
            </button>
            <span v-if="notice" class="max-w-[45%] truncate text-[11px]" :class="notice.ok ? 'text-[var(--ok)]' : 'text-[var(--err)]'">
              {{ notice.text }}
            </span>
          </div>
        </section>
      </div>

      <div class="space-y-4">
        <TaskProgressCard />

        <section class="glass-panel p-5">
          <div class="mb-3 flex items-center gap-2 text-[13px] font-semibold">
            <History :size="15" class="text-[var(--brand-300)]" /> 最近任务
          </div>
          <div v-if="!tasks.history.length" class="py-6 text-center text-[12px] text-[var(--text-muted)]">
            暂无历史记录。完成一次下载后，这里会显示成功、跳过与失败的数量。
          </div>
          <div v-else class="overflow-x-auto">
            <table class="w-full text-left text-[11.5px]">
              <thead class="text-[var(--text-muted)]">
                <tr class="border-b border-[var(--border-soft)]">
                  <th class="pb-2 font-medium">时间</th>
                  <th class="pb-2 font-medium">类型</th>
                  <th class="pb-2 font-medium">音质</th>
                  <th class="pb-2 font-medium">成功</th>
                  <th class="pb-2 font-medium">跳过</th>
                  <th class="pb-2 font-medium">失败</th>
                  <th class="pb-2 font-medium">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in tasks.history" :key="row.id" class="border-b border-[var(--border-soft)] last:border-b-0">
                  <td class="py-2 font-mono text-[var(--text-secondary)]">{{ formatTime(row.created_at) }}</td>
                  <td class="py-2">{{ row.kind }}</td>
                  <td class="py-2 text-[var(--text-secondary)]">{{ row.quality || '—' }}</td>
                  <td class="py-2 text-[var(--ok)]">{{ row.success }}</td>
                  <td class="py-2 text-[var(--warn)]">{{ row.skipped }}</td>
                  <td class="py-2 text-[var(--err)]">{{ row.failed }}</td>
                  <td class="py-2">
                    <button type="button" class="flex items-center gap-1 text-[11px] text-[var(--brand-400)] hover:text-[var(--brand-300)]" @click="openDir">
                      <FolderOpen :size="12" /> 打开所在目录
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>
