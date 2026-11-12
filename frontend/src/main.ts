import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { i18n, registerLocale, type Locale } from './i18n'
import zhCN from './i18n/locales/zh-CN'
import en from './i18n/locales/en'
import ja from './i18n/locales/ja'
import './styles/theme.css'
import './styles/global.css'

registerLocale('zh-CN', zhCN as unknown as Record<string, unknown>)
registerLocale('en', en as unknown as Record<string, unknown>)
registerLocale('ja', ja as unknown as Record<string, unknown>)

createApp(App).use(createPinia()).use(router).use(i18n).mount('#app')

// 首帧即同步 <html lang>，保证字体与无障碍语义正确
document.documentElement.lang = i18n.global.locale.value as Locale
