<script setup lang="ts">
import { Languages } from 'lucide-vue-next'
import { i18n, setLocale, SUPPORTED_LOCALES, LOCALE_LABELS, type Locale } from '@/i18n'
import { computed } from 'vue'

const current = computed(() => i18n.global.locale.value as Locale)

function onChange(event: Event) {
  setLocale((event.target as HTMLSelectElement).value as Locale)
}
</script>

<template>
  <label class="flex items-center gap-1.5 text-[11px] text-[var(--text-muted)]">
    <Languages :size="13" />
    <select
      class="field-input h-7 w-[86px] px-1.5 py-0 text-[11px]"
      :value="current"
      @change="onChange"
    >
      <option v-for="locale in SUPPORTED_LOCALES" :key="locale" :value="locale">
        {{ LOCALE_LABELS[locale] }}
      </option>
    </select>
  </label>
</template>
