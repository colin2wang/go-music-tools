import { createI18n } from 'vue-i18n'

export const LOCALE_KEY = 'app.locale'
export const SUPPORTED_LOCALES = ['zh-CN', 'en', 'ja'] as const
export type Locale = (typeof SUPPORTED_LOCALES)[number]

export const LOCALE_LABELS: Record<Locale, string> = {
  'zh-CN': '中文',
  en: 'English',
  ja: '日本語',
}

/** 读取持久化的语言偏好；无记录时跟随浏览器语言，兜底 zh-CN。 */
export function detectLocale(): Locale {
  const saved = localStorage.getItem(LOCALE_KEY) as Locale | null
  if (saved && (SUPPORTED_LOCALES as readonly string[]).includes(saved)) return saved
  const nav = navigator.language.toLowerCase()
  if (nav.startsWith('zh')) return 'zh-CN'
  if (nav.startsWith('ja')) return 'ja'
  if (nav.startsWith('en')) return 'en'
  return 'zh-CN'
}

export function setLocale(locale: Locale) {
  i18n.global.locale.value = locale
  localStorage.setItem(LOCALE_KEY, locale)
  document.documentElement.lang = locale
}

// 后端校验消息等运行期文本以 zh-CN 为兜底；缺失键回退到中文而不是裸键名。

export const i18n = createI18n({
  legacy: false,
  locale: detectLocale(),
  fallbackLocale: 'zh-CN',
  missingWarn: false,
  fallbackWarn: false,
  messages: {}, // 由 registerLocale 在 main.ts 中填充
})

/** 注册语言包（main.ts 中按需 import 后调用）。 */
export function registerLocale(locale: Locale, messages: Record<string, unknown>) {
  i18n.global.setLocaleMessage(locale, messages)
}
