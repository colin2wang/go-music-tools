import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/download' },
  {
    path: '/download',
    name: 'download',
    component: () => import('@/views/DownloadView.vue'),
    meta: { title: '下载工作台', icon: 'download' },
  },
  {
    path: '/search',
    name: 'search',
    component: () => import('@/views/SearchView.vue'),
    meta: { title: '音乐搜索', icon: 'search' },
  },
  {
    path: '/oomusic',
    name: 'oomusic',
    component: () => import('@/views/OOMusicView.vue'),
    meta: { title: 'OO Music', icon: 'music' },
  },
  {
    path: '/library',
    name: 'library',
    component: () => import('@/views/LibraryToolsView.vue'),
    meta: { title: '音乐库工具', icon: 'folder' },
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('@/views/SettingsView.vue'),
    meta: { title: '设置', icon: 'settings' },
  },
]

export default createRouter({
  history: createWebHashHistory(),
  routes,
})
