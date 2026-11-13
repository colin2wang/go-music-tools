<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { CheckCircle2, SkipForward, XCircle, CircleDashed } from 'lucide-vue-next'
import type { TaskSummary } from '@/stores/task'

const props = defineProps<{ summary: TaskSummary }>()
const { t } = useI18n()
const hovered = ref(false)

const RADIUS = 12.5
const CIRCUMFERENCE = 2 * Math.PI * RADIUS

const segments = computed(() => {
  const s = props.summary
  const total = Math.max(1, s.total)
  const parts = [
    { key: 'success', value: s.success, color: 'var(--ok)' },
    { key: 'skipped', value: s.skipped, color: 'var(--warn)' },
    { key: 'failed', value: s.failed, color: 'var(--err)' },
    { key: 'remaining', value: s.remaining, color: 'rgba(148, 163, 184, 0.22)' },
  ]
  let start = 0
  return parts.map((part) => {
    const length = Math.max(0, Math.min(CIRCUMFERENCE, (part.value / total) * CIRCUMFERENCE))
    const segment = { ...part, length, offset: -start }
    start += length
    return segment
  })
})

const rows = computed(() => [
  { icon: CheckCircle2, label: t('task.donutSuccess'), value: props.summary.success, color: 'var(--ok)' },
  { icon: SkipForward, label: t('task.donutSkipped'), value: props.summary.skipped, color: 'var(--warn)' },
  { icon: XCircle, label: t('task.donutFailed'), value: props.summary.failed, color: 'var(--err)' },
  { icon: CircleDashed, label: t('task.donutRemaining'), value: props.summary.remaining, color: 'var(--text-muted)' },
])
</script>

<template>
  <div
    class="donut-wrap"
    @mouseenter="hovered = true"
    @mouseleave="hovered = false"
  >
    <div class="donut-halo" :class="summary.count > 0 ? 'halo-active' : ''">
      <svg viewBox="0 0 32 32" class="h-[30px] w-[30px]" role="img" :aria-label="t('task.donutTitle')">
        <circle cx="16" cy="16" :r="RADIUS" fill="none" stroke="rgba(148,163,184,0.14)" stroke-width="4" />
        <g transform="rotate(-90 16 16)">
          <circle
            v-for="segment in segments"
            :key="segment.key"
            cx="16"
            cy="16"
            :r="RADIUS"
            fill="none"
            :stroke="segment.color"
            stroke-width="4"
            stroke-linecap="butt"
            :stroke-dasharray="`${segment.length} ${CIRCUMFERENCE - segment.length}`"
            :stroke-dashoffset="segment.offset"
            class="donut-segment"
          />
        </g>
        <text
          x="16" y="16"
          text-anchor="middle" dominant-baseline="central"
          font-size="9" font-weight="700" font-family="ui-monospace, monospace"
          fill="var(--text-primary)"
        >{{ summary.percent }}</text>
      </svg>
    </div>

    <Transition name="pop">
      <div v-if="hovered" class="donut-popover">
        <div class="mb-2 flex items-center justify-between text-[10.5px] font-semibold text-[var(--text-primary)]">
          <span>{{ t('task.donutTitle') }}</span>
          <span class="font-mono text-[var(--brand-300)]">{{ summary.percent }}%</span>
        </div>
        <div class="space-y-1.5">
          <div v-for="row in rows" :key="row.label" class="flex items-center gap-2 text-[10px]">
            <component :is="row.icon" :size="11" :style="{ color: row.color }" />
            <span class="flex-1 text-[var(--text-secondary)]">{{ row.label }}</span>
            <span class="font-mono" :style="{ color: row.color }">{{ row.value }}</span>
          </div>
        </div>
        <div class="mt-2 border-t border-[var(--border-soft)] pt-2 text-[9.5px] text-[var(--text-muted)]">
          {{ t('task.donutLanes', { n: summary.count }) }}
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.donut-wrap { position: relative; display: flex; align-items: center; }
.donut-halo {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: 999px;
  transition: box-shadow 220ms ease, background 220ms ease;
}
.halo-active {
  background: radial-gradient(circle, rgba(108, 92, 231, 0.16), transparent 72%);
  box-shadow: 0 0 14px -4px rgba(34, 211, 238, 0.55);
}
.donut-segment { transition: stroke-dasharray 320ms cubic-bezier(0.22, 1, 0.36, 1), stroke-dashoffset 320ms cubic-bezier(0.22, 1, 0.36, 1); }

.donut-popover {
  position: absolute;
  top: calc(100% + 9px);
  right: -8px;
  z-index: 60;
  width: 186px;
  border: 1px solid rgba(139, 124, 246, 0.32);
  border-radius: 12px;
  background: rgba(19, 26, 46, 0.97);
  box-shadow: 0 18px 40px -16px rgba(0, 0, 0, 0.9), 0 0 20px -12px rgba(108, 92, 231, 0.9);
  backdrop-filter: blur(14px);
  padding: 10px 11px;
}
.donut-popover::before {
  content: '';
  position: absolute;
  top: -5px;
  right: 18px;
  width: 9px;
  height: 9px;
  transform: rotate(45deg);
  border-left: 1px solid rgba(139, 124, 246, 0.32);
  border-top: 1px solid rgba(139, 124, 246, 0.32);
  background: rgba(19, 26, 46, 0.97);
}
.pop-enter-active, .pop-leave-active { transition: opacity 160ms ease, transform 160ms ease; }
.pop-enter-from, .pop-leave-to { opacity: 0; transform: translateY(-5px) scale(0.97); }
</style>
