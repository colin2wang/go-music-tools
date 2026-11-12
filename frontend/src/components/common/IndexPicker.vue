<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ListChecks, Info } from 'lucide-vue-next'
import { bindings } from '@/api/bindings'

const props = defineProps<{ modelValue: string; disabled?: boolean }>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: number[]): void
}>()

const raw = ref(props.modelValue)
const parsed = ref<number[]>([])
let timer: number | undefined

watch(() => props.modelValue, (value) => {
  if (value !== raw.value) raw.value = value
})

async function parse() {
  const result = await bindings.parseIndexes(raw.value)
  parsed.value = result ?? []
  emit('change', parsed.value)
}

function scheduleParse() {
  if (timer) window.clearTimeout(timer)
  timer = window.setTimeout(() => { void parse() }, 220)
}

function onInput(event: Event) {
  raw.value = (event.target as HTMLInputElement).value
  emit('update:modelValue', raw.value)
  scheduleParse()
}

onMounted(() => { void parse() })
onBeforeUnmount(() => { if (timer) window.clearTimeout(timer) })
</script>

<template>
  <div>
    <label class="mb-1.5 flex items-center gap-1.5 text-[12px] font-medium text-[var(--text-secondary)]">
      <ListChecks :size="13" /> 指定曲目编号
    </label>
    <input
      type="text"
      class="field-input h-10 font-mono text-[12.5px]"
      placeholder="留空下载全部，例如 1 3 5-9 或 1,2,3"
      :value="raw"
      :disabled="disabled"
      @input="onInput"
    />
    <div class="mt-1.5 flex flex-wrap items-center gap-2 text-[11px]">
      <span v-if="parsed.length" class="rounded-full bg-[rgba(108,92,231,0.14)] px-2 py-0.5 text-[var(--brand-400)]">
        共 {{ parsed.length }} 首
      </span>
      <span v-else class="flex items-center gap-1 text-[var(--text-muted)]"><Info :size="11" /> 全部曲目</span>
      <span v-if="parsed.length" class="truncate font-mono text-[var(--text-muted)]">
        {{ parsed.slice(0, 20).join(', ') }}{{ parsed.length > 20 ? ' …' : '' }}
      </span>
    </div>
  </div>
</template>
