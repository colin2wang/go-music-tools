<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { FolderOpen, Folder, RefreshCw } from 'lucide-vue-next'
import { bindings } from '@/api/bindings'

const { t } = useI18n()
const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'folders', value: string[]): void
}>()

const folders = ref<string[]>([])
const loading = ref(false)
const notice = ref('')

async function refresh() {
  if (!props.modelValue) {
    folders.value = []
    emit('folders', [])
    return
  }
  loading.value = true
  const list = await bindings.listSubFolders(props.modelValue)
  folders.value = list ?? []
  emit('folders', folders.value)
  loading.value = false
}

async function browse() {
  const path = await bindings.selectDirectory(t('library.rootLabel'))
  if (path) {
    emit('update:modelValue', path)
    notice.value = t('library.dirUpdated')
    window.setTimeout(() => { notice.value = '' }, 2200)
    await refresh()
  }
}

watch(() => props.modelValue, () => { void refresh() })
onMounted(() => { void refresh() })
</script>

<template>
  <div class="glass-panel p-4">
    <label class="mb-1.5 block text-[12px] font-medium text-[var(--text-secondary)]">{{ t('library.rootLabel') }}</label>
    <div class="flex flex-wrap gap-2">
      <input
        type="text"
        class="field-input h-10 min-w-[240px] flex-1 font-mono text-[12px]"
        :placeholder="t('library.rootPlaceholder')"
        :value="modelValue"
        @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
      />
      <button type="button" class="ghost-button flex h-10 items-center gap-2 px-3 text-[12px]" @click="browse">
        <FolderOpen :size="15" /> {{ t('library.browse') }}
      </button>
      <button type="button" class="ghost-button flex h-10 w-10 items-center justify-center" :title="t('library.refreshSub')" @click="refresh">
        <RefreshCw :size="14" :class="loading ? 'animate-spin' : ''" />
      </button>
    </div>

    <div class="mt-3 flex flex-wrap items-center gap-2">
      <span class="flex items-center gap-1.5 text-[11.5px] text-[var(--text-secondary)]">
        <Folder :size="13" /> {{ t('library.detected', { n: folders.length }) }}
      </span>
      <span v-if="notice" class="text-[11px] text-[var(--ok)]">{{ notice }}</span>
      <div v-if="folders.length" class="flex w-full flex-wrap gap-1.5">
        <span
          v-for="folder in folders.slice(0, 12)"
          :key="folder"
          class="max-w-[220px] truncate rounded-md border border-[var(--border-soft)] bg-[rgba(148,163,184,0.06)] px-2 py-1 text-[10.5px] text-[var(--text-muted)]"
          :title="folder"
        >
          {{ folder.split(/[\\/]/).pop() }}
        </span>
        <span v-if="folders.length > 12" class="px-2 py-1 text-[10.5px] text-[var(--text-muted)]">
          {{ t('library.more', { n: folders.length - 12 }) }}
        </span>
      </div>
    </div>
  </div>
</template>
