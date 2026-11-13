<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Loader2, X } from 'lucide-vue-next'

const props = defineProps<{
  title: string
  content: string
  loading?: boolean
}>()

const emit = defineEmits<{ close: [] }>()

const { t } = useI18n()
const tab = ref<'table' | 'raw'>('table')

// 把缓存值解析成 key/value 表格行；字符串中的换行符原样保留，
// 嵌套结构展开为缩进 JSON。仅顶层对象可成表，否则只有 RAW 可看。
const rows = computed<Array<{ k: string; v: string }> | null>(() => {
  let parsed: unknown
  try {
    parsed = JSON.parse(props.content)
  } catch {
    return null
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null
  return Object.entries(parsed).map(([k, v]) => ({
    k,
    v: typeof v === 'string' ? v : JSON.stringify(v, null, 2),
  }))
})

// 内容切换（换条目）时回到表格视图；无表格可显示时停在 RAW。
watch(() => props.content, () => {
  tab.value = 'table'
})
watch(rows, (value) => {
  if (!value && tab.value === 'table') tab.value = 'raw'
})
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(5,8,18,0.72)] p-6" @click.self="emit('close')">
    <div class="glass-panel flex max-h-[80vh] w-full max-w-[860px] flex-col overflow-hidden">
      <div class="flex items-center gap-3 border-b border-[var(--border-soft)] px-4 py-2.5">
        <span class="truncate font-mono text-[12px] font-semibold" :title="title">{{ title }}</span>
        <div class="ml-auto flex items-center gap-2">
          <div v-if="rows" class="flex overflow-hidden rounded-md border border-[var(--border-soft)] text-[11px]">
            <button
              type="button"
              class="px-2.5 py-1 transition-colors"
              :class="tab === 'table' ? 'bg-[rgba(108,92,231,0.2)] text-[var(--brand-300)]' : 'text-[var(--text-muted)] hover:text-[var(--text-secondary)]'"
              @click="tab = 'table'"
            >{{ t('cache.tabTable') }}</button>
            <button
              type="button"
              class="px-2.5 py-1 transition-colors"
              :class="tab === 'raw' ? 'bg-[rgba(108,92,231,0.2)] text-[var(--brand-300)]' : 'text-[var(--text-muted)] hover:text-[var(--text-secondary)]'"
              @click="tab = 'raw'"
            >RAW</button>
          </div>
          <button type="button" class="text-[var(--text-secondary)] hover:opacity-80" @click="emit('close')">
            <X :size="16" />
          </button>
        </div>
      </div>
      <div class="overflow-auto px-4 py-3">
        <template v-if="loading || !content">
          <div v-if="loading || content === ''" class="flex items-center justify-center gap-2 py-10 text-[12px] text-[var(--text-muted)]">
            <Loader2 :size="15" class="animate-spin" /> {{ t('common.loading') }}
          </div>
          <p v-else class="py-6 text-center text-[12px] text-[var(--text-muted)]">{{ t('cache.emptyDetail') }}</p>
        </template>
        <table v-else-if="tab === 'table' && rows" class="w-full table-fixed border-collapse text-left text-[11.5px]">
          <tbody>
            <tr v-for="row in rows" :key="row.k" class="border-b border-[var(--border-soft)] last:border-b-0">
              <th class="w-40 shrink-0 break-all px-1 py-2 align-top font-mono font-medium text-[var(--text-muted)]">{{ row.k }}</th>
              <td class="whitespace-pre-wrap break-words px-1 py-2 font-mono leading-relaxed text-[var(--text-secondary)]">{{ row.v }}</td>
            </tr>
          </tbody>
        </table>
        <pre v-else class="whitespace-pre-wrap break-all font-mono text-[11.5px] leading-relaxed text-[var(--text-secondary)]">{{ content }}</pre>
      </div>
    </div>
  </div>
</template>
