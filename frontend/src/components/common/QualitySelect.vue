<script setup lang="ts">
import { computed } from 'vue'
import { QUALITY_LEVELS, OO_QUALITY_LEVELS } from '@/types'

const props = defineProps<{
  modelValue: string
  source?: 'netease' | 'oo'
  label?: string
  disabled?: boolean
}>()
const emit = defineEmits<{ (e: 'update:modelValue', value: string): void }>()

const levels = computed(() => (props.source === 'oo' ? OO_QUALITY_LEVELS : QUALITY_LEVELS))
const hint = computed(() => levels.value.find((item) => item.value === props.modelValue)?.hint ?? '')
</script>

<template>
  <div>
    <label v-if="label" class="mb-1.5 block text-[12px] font-medium text-[var(--text-secondary)]">{{ label }}</label>
    <select
      class="field-input h-10 text-[13px]"
      :value="modelValue"
      :disabled="disabled"
      @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
    >
      <option v-for="level in levels" :key="level.value" :value="level.value">
        {{ level.label }} · {{ level.value }}
      </option>
    </select>
    <p class="mt-1.5 text-[11px] text-[var(--text-muted)]">{{ hint }}</p>
  </div>
</template>
