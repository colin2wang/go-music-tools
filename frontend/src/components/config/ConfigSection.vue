<script setup lang="ts">
import type { ConfigGroup } from '@/types'
import ConfigField from './ConfigField.vue'

const props = defineProps<{
  group: ConfigGroup
  values: Record<string, string | number | boolean>
  errors: Record<string, string>
}>()

const emit = defineEmits<{
  (e: 'update', key: string, value: string | number | boolean): void
  (e: 'browse', key: string): void
}>()

function errorFor(key: string): string {
  return props.errors[`${props.group.key}.${key}`] ?? ''
}
</script>

<template>
  <section :id="`cfg-${group.key}`" class="glass-panel scroll-mt-6 p-5">
    <div class="mb-4 flex items-start gap-3">
      <div class="mt-0.5 h-8 w-1 rounded-full" style="background: var(--gradient-brand)" />
      <div>
        <h2 class="section-title">{{ group.label }}</h2>
        <p class="mt-1 text-[12px] text-[var(--text-muted)]">{{ group.description }}</p>
      </div>
      <span class="ml-auto rounded-full border border-[var(--border-soft)] px-2.5 py-1 font-mono text-[10.5px] text-[var(--text-muted)]">
        {{ group.key }}
      </span>
    </div>

    <div class="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
      <ConfigField
        v-for="field in group.fields"
        :key="field.key"
        :field="field"
        :value="values[field.key] ?? field.default"
        :error="errorFor(field.key)"
        @update="(v) => emit('update', field.key, v)"
        @browse="emit('browse', field.key)"
      />
    </div>
  </section>
</template>
