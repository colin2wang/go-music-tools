<script setup lang="ts">
import { computed, ref } from 'vue'
import { Save, RotateCcw, RefreshCw, FileText, CheckCircle2, AlertCircle, Loader2, Download, Upload } from 'lucide-vue-next'
import { useConfigStore } from '@/stores/config'
import { bindings } from '@/api/bindings'
import ConfigSection from './ConfigSection.vue'

const config = useConfigStore()
const notice = ref<{ ok: boolean; text: string } | null>(null)

const sections = computed(() => config.groups)
const changedCount = computed(() => {
  let count = 0
  for (const group of config.groups) {
    for (const field of group.fields) {
      const current = config.values[group.key]?.[field.key]
      if (String(current ?? '') !== String(config.original[group.key]?.[field.key] ?? '')) count++
    }
  }
  return count
})

function flash(ok: boolean, text: string) {
  notice.value = { ok, text }
  window.setTimeout(() => { notice.value = null }, 3200)
}

function update(group: string, key: string, value: string | number | boolean) {
  config.setField(group, key, value)
  notice.value = null
}

async function browse(group: string, key: string) {
  const path = await bindings.selectDirectory(`选择${group}.${key} 目录`)
  if (path) config.setField(group, key, path)
}

async function save() {
  const ok = await config.save()
  flash(ok, ok ? '配置已保存并立即生效' : `有 ${config.errorCount} 个字段需要修正`)
}

async function reload() {
  await config.reload()
  flash(true, '已从 config.yaml 重新加载')
}

async function reset() {
  await config.reset()
  flash(true, '已恢复全部默认值')
}

function errorText(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

async function exportConfig() {
  const path = await bindings.exportConfig().catch((err: unknown) => {
    console.error(err)
    flash(false, errorText(err))
    return null
  })
  if (path) flash(true, `已导出到 ${path}`)
}

async function importConfig() {
  const path = await bindings.importConfig().catch((err: unknown) => {
    console.error(err)
    flash(false, errorText(err))
    return null
  })
  if (!path) return
  await config.refresh()
  flash(true, `已从 ${path} 导入并立即生效`)
}
</script>

<template>
  <div class="space-y-4">
    <div class="glass-panel sticky top-0 z-10 flex flex-wrap items-center gap-3 p-4">
      <div class="mr-auto">
        <div class="flex items-center gap-2 text-[13px] font-semibold">
          配置编辑
          <span
            v-if="config.dirty"
            class="rounded-full bg-[rgba(245,158,11,0.14)] px-2 py-0.5 text-[10.5px] font-normal text-[var(--warn)]"
          >
            {{ changedCount }} 项未保存
          </span>
          <span v-else class="rounded-full bg-[rgba(34,197,94,0.12)] px-2 py-0.5 text-[10.5px] font-normal text-[var(--ok)]">
            已是最新
          </span>
        </div>
        <p class="mt-1 font-mono text-[11px] text-[var(--text-muted)]">{{ config.appInfo?.config_path }}</p>
      </div>

      <span v-if="notice" class="flex items-center gap-1.5 text-[11.5px]" :class="notice.ok ? 'text-[var(--ok)]' : 'text-[var(--err)]'">
        <CheckCircle2 v-if="notice.ok" :size="13" />
        <AlertCircle v-else :size="13" />
        {{ notice.text }}
      </span>

      <button type="button" class="ghost-button flex h-9 items-center gap-1.5 px-3 text-[12px]" @click="reload">
        <RefreshCw :size="14" /> 重新加载
      </button>
      <button type="button" class="ghost-button flex h-9 items-center gap-1.5 px-3 text-[12px]" @click="reset">
        <RotateCcw :size="14" /> 恢复默认
      </button>
      <button type="button" class="ghost-button flex h-9 items-center gap-1.5 px-3 text-[12px]" @click="config.openFile()">
        <FileText :size="14" /> 打开文件
      </button>
      <button type="button" class="ghost-button flex h-9 items-center gap-1.5 px-3 text-[12px]" @click="exportConfig">
        <Download :size="14" /> 导出
      </button>
      <button type="button" class="ghost-button flex h-9 items-center gap-1.5 px-3 text-[12px]" @click="importConfig">
        <Upload :size="14" /> 导入
      </button>
      <button type="button" class="brand-button flex h-9 items-center gap-1.5 px-4 text-[12px]" :disabled="config.saving" @click="save">
        <Loader2 v-if="config.saving" :size="14" class="animate-spin" />
        <Save v-else :size="14" />
        保存配置
      </button>
    </div>

    <div
      v-if="config.errorCount > 0"
      class="glass-panel border-[rgba(239,68,68,0.4)] p-4 text-[12px] text-[var(--err)]"
    >
      <div class="mb-2 flex items-center gap-2 font-semibold"><AlertCircle :size="14" /> 字段校验未通过</div>
      <ul class="space-y-1">
        <li v-for="(message, path) in config.errors" :key="path" class="font-mono text-[11px]">
          {{ path }} — {{ message }}
        </li>
      </ul>
    </div>

    <ConfigSection
      v-for="group in sections"
      :key="group.key"
      :group="group"
      :values="config.values[group.key] ?? {}"
      :errors="config.errors"
      @update="(key, value) => update(group.key, key, value)"
      @browse="(key) => browse(group.key, key)"
    />
  </div>
</template>
