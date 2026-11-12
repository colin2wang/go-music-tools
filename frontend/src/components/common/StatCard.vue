<script setup lang="ts">
import type { Component } from 'vue'

const props = defineProps<{
  label: string
  value: string | number
  tone?: 'brand' | 'ok' | 'warn' | 'err' | 'info'
  hint?: string
  icon?: Component
}>()

const tones: Record<string, string> = {
  brand: 'text-[var(--brand-300)] bg-[rgba(108,92,231,0.14)]',
  ok: 'text-[var(--ok)] bg-[rgba(34,197,94,0.12)]',
  warn: 'text-[var(--warn)] bg-[rgba(245,158,11,0.12)]',
  err: 'text-[var(--err)] bg-[rgba(239,68,68,0.12)]',
  info: 'text-[var(--info)] bg-[rgba(56,189,248,0.12)]',
}
</script>

<template>
  <div class="glass-panel flex items-center gap-3 p-3.5">
    <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl" :class="tones[props.tone ?? 'brand']">
      <component :is="props.icon" v-if="props.icon" :size="16" />
      <span v-else class="text-[13px] font-semibold">{{ String(props.value).slice(0, 2) }}</span>
    </div>
    <div class="min-w-0">
      <div class="text-[11px] text-[var(--text-muted)]">{{ props.label }}</div>
      <div class="truncate text-[19px] font-semibold leading-tight">{{ props.value }}</div>
    </div>
    <span v-if="props.hint" class="ml-auto hidden truncate text-[11px] text-[var(--text-muted)] xl:inline">{{ props.hint }}</span>
  </div>
</template>
