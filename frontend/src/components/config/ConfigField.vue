<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { FolderOpen, RotateCcw, AlertCircle } from 'lucide-vue-next'
import type { ConfigField as Field } from '@/types'

const { t, te } = useI18n()

const props = defineProps<{
  field: Field
  groupKey: string
  value: string | number | boolean
  error?: string
}>()

const emit = defineEmits<{
  (e: 'update', value: string | number | boolean): void
  (e: 'browse'): void
}>()

// 语言包有对应键时使用翻译，否则回退后端 schema 自带的中文文案。
const label = computed(() => {
  const key = `schema.field.${props.groupKey}.${props.field.key}`
  return te(key) ? t(key) : props.field.label
})
const description = computed(() => {
  const key = `schema.desc.${props.groupKey}.${props.field.key}`
  return te(key) ? t(key) : props.field.description
})

const changed = computed(() => String(props.value) !== String(props.field.default))
const isPath = computed(() => props.field.type === 'path')

function onInput(event: Event) {
  const raw = (event.target as HTMLInputElement).value
  if (props.field.type === 'int' || props.field.type === 'float') {
    const parsed = Number(raw)
    emit('update', Number.isFinite(parsed) ? parsed : 0)
    return
  }
  emit('update', raw)
}

function toggle() {
  emit('update', !props.value)
}

function reset() {
  emit('update', props.field.default)
}
</script>

<template>
  <div
    class="group rounded-xl border p-3.5 transition-all duration-150"
    :class="error
      ? 'border-[rgba(239,68,68,0.5)] bg-[rgba(239,68,68,0.05)]'
      : changed
        ? 'border-[rgba(139,124,246,0.35)] bg-[rgba(108,92,231,0.06)]'
        : 'border-[var(--border-soft)] bg-[rgba(11,16,32,0.45)] hover:border-[rgba(148,163,184,0.28)]'"
  >
    <div class="mb-1 flex items-center gap-2">
      <label class="text-[12.5px] font-semibold text-[var(--text-primary)]">{{ label }}</label>
      <span class="rounded bg-[rgba(148,163,184,0.1)] px-1.5 py-0.5 font-mono text-[10px] text-[var(--text-muted)]">
        {{ field.type }}
      </span>
      <span v-if="field.required" class="text-[11px] text-[var(--brand-300)]">{{ t('common.required') }}</span>
      <button
        v-if="changed"
        type="button"
        class="ml-auto flex items-center gap-1 text-[10.5px] text-[var(--text-muted)] transition-colors hover:text-[var(--brand-300)]"
        :title="t('common.resetField')"
        @click="reset"
      >
        <RotateCcw :size="11" /> {{ t('common.defaultWith', { v: field.default }) }}
      </button>
    </div>

    <div class="mt-2">
      <div v-if="field.type === 'bool'" class="flex items-center gap-3">
        <button
          type="button"
          role="switch"
          :aria-checked="Boolean(value)"
          class="relative h-6 w-11 shrink-0 rounded-full border transition-all duration-200"
          :class="value ? 'border-transparent' : 'border-[var(--border-soft)] bg-[rgba(148,163,184,0.14)]'"
          :style="value ? { background: 'var(--gradient-brand)' } : {}"
          @click="toggle"
        >
          <span
            class="absolute top-[3px] h-4 w-4 rounded-full bg-white shadow transition-all duration-200"
            :style="{ left: value ? '24px' : '4px' }"
          />
        </button>
        <span class="text-[12px] text-[var(--text-secondary)]">{{ value ? t('common.enabled') : t('common.disabled') }}</span>
      </div>

      <select
        v-else-if="field.type === 'enum'"
        class="field-input h-9 text-[12.5px]"
        :value="String(value)"
        @change="emit('update', ($event.target as HTMLSelectElement).value)"
      >
        <option v-for="option in field.options ?? []" :key="option" :value="option">{{ option }}</option>
      </select>

      <input
        v-else-if="field.type === 'int' || field.type === 'float'"
        type="number"
        class="field-input h-9 text-[12.5px]"
        :value="value"
        :min="field.min ?? undefined"
        :max="field.max ?? undefined"
        :step="field.type === 'float' ? '0.1' : '1'"
        @input="onInput"
      />

      <div v-else class="flex gap-2">
        <input
          type="text"
          class="field-input h-9 text-[12.5px]"
          :value="String(value)"
          :placeholder="description"
          @input="onInput"
        />
        <button
          v-if="isPath"
          type="button"
          class="ghost-button flex h-9 shrink-0 items-center gap-1.5 px-3 text-[11.5px]"
          :title="t('settings.browseDir')"
          @click="emit('browse')"
        >
          <FolderOpen :size="14" /> {{ t('common.browse') }}
        </button>
      </div>
    </div>

    <p class="mt-2 text-[11px] leading-relaxed text-[var(--text-muted)]">{{ description }}</p>
    <p v-if="error" class="mt-1.5 flex items-center gap-1 text-[11px] text-[var(--err)]">
      <AlertCircle :size="12" /> {{ error }}
    </p>
  </div>
</template>
