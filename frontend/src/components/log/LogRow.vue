<script setup lang="ts">
import type { LogEntry } from '@/types'

defineProps<{ entry: LogEntry }>()

const tone: Record<string, string> = {
  DEBUG: 'text-[var(--text-muted)] bg-[rgba(148,163,184,0.1)]',
  INFO: 'text-[var(--info)] bg-[rgba(56,189,248,0.1)]',
  WARN: 'text-[var(--warn)] bg-[rgba(245,158,11,0.12)]',
  ERROR: 'text-[var(--err)] bg-[rgba(239,68,68,0.14)]',
}

function fieldsText(fields?: Record<string, string>): string {
  if (!fields) return ''
  return Object.entries(fields)
    .map(([k, v]) => `${k}=${v}`)
    .join('  ')
}
</script>

<template>
  <div class="flex h-6 animate-log-in items-center gap-2 whitespace-nowrap px-3 font-mono text-[11.5px] leading-6 hover:bg-[rgba(148,163,184,0.05)]">
    <span class="w-[92px] shrink-0 text-[var(--text-muted)]">{{ entry.time.slice(11) }}</span>
    <span class="w-[52px] shrink-0 rounded text-center text-[10px] font-semibold" :class="tone[entry.level] ?? tone.INFO">
      {{ entry.level }}
    </span>
    <span v-if="entry.source" class="w-[150px] shrink-0 truncate text-[var(--brand-400)]" :title="entry.source">{{ entry.source }}</span>
    <span class="min-w-0 flex-1 truncate" :class="entry.level === 'ERROR' ? 'text-[var(--err)]' : 'text-[var(--text-primary)]'" :title="entry.message">
      {{ entry.message }}
    </span>
    <span v-if="entry.fields && Object.keys(entry.fields).length" class="hidden shrink-0 truncate text-[var(--text-muted)] xl:inline" :title="fieldsText(entry.fields)">
      {{ fieldsText(entry.fields) }}
    </span>
  </div>
</template>
