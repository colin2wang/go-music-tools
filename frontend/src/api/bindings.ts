import type {
  AppInfo,
  ConfigGroup,
  ConfigValues,
  DownloadRequest,
  LibraryFixRequest,
  LogEntry,
  OODetail,
  OODownloadRequest,
  OOSong,
  QualityOption,
  SearchSong,
  TaskDone,
  TaskProgress,
  TaskRecord,
} from '@/types'

type Callable = (...args: unknown[]) => Promise<unknown>

function bridge(): Record<string, Callable> {
  const w = window as unknown as { go?: { main?: { App?: Record<string, Callable> } } }
  return w.go?.main?.App ?? {}
}

/** 统一调用 Wails 绑定方法，并归一化错误。 */
export async function call<T>(method: string, ...args: unknown[]): Promise<T> {
  const fn = bridge()[method]
  if (typeof fn !== 'function') {
    throw new Error(`绑定方法 ${method} 不可用，请从 Wails 窗口启动应用`)
  }
  try {
    return (await fn(...args)) as T
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err)
    throw new Error(message)
  }
}

export interface LastPreference {
  kind: 'song' | 'album' | 'playlist'
  resource_id: string
  quality: string
  indexes: number[]
}

export interface LibraryFixResult {
  task_id: string
}

export const bindings = {
  // 配置
  getConfigSchema: () => call<ConfigGroup[]>('GetConfigSchema'),
  getConfig: () => call<ConfigValues>('GetConfig'),
  validateConfig: (values: ConfigValues) => call<Record<string, string>>('ValidateConfig', values),
  saveConfig: (values: ConfigValues) => call<void>('SaveConfig', values),
  reloadConfig: () => call<void>('ReloadConfig'),
  resetConfig: () => call<void>('ResetConfig'),
  openConfigFile: () => call<void>('OpenConfigFile'),
  exportConfig: () => call<string>('ExportConfig'),
  importConfig: () => call<string>('ImportConfig'),

  // 全局日志
  getRecentLogs: () => call<LogEntry[]>('GetRecentLogs'),
  clearLogs: () => call<void>('ClearLogs'),
  exportLogs: () => call<string>('ExportLogs'),
  openLogDir: () => call<void>('OpenLogDir'),

  // 系统
  getAppInfo: () => call<AppInfo>('GetAppInfo'),
  getTaskStatus: () => call<TaskProgress>('GetTaskStatus'),
  getTaskHistory: (limit: number) => call<TaskRecord[]>('GetTaskHistory', limit),
  cancelTask: () => call<boolean>('CancelTask'),
  openDownloadDir: () => call<void>('OpenDownloadDir'),
  openPath: (path: string) => call<void>('OpenPath', path),
  selectDirectory: (title: string) => call<string>('SelectDirectory', title),
  listSubFolders: (dir: string) => call<string[]>('ListSubFolders', dir),

  // 下载工作台
  getLastPreference: () => call<LastPreference>('GetLastDownloadConfig'),
  saveLastPreference: (p: LastPreference) => call<void>('SaveLastDownloadConfig', p),
  download: (req: DownloadRequest) => call<string>('Download', req),
  downloadSongs: (ids: string[], quality: string) => call<string>('DownloadSongs', ids, quality),
  parseIndexes: (raw: string) => call<number[]>('ParseIndexes', raw),
  extractResourceID: (raw: string) => call<string>('ExtractResourceID', raw),
  searchSongs: (keyword: string) => call<SearchSong[]>('SearchSongs', keyword),

  // OO Music
  ooSearch: (keyword: string) => call<OOSong[]>('OOSearch', keyword),
  ooDetail: (keyword: string, songMid: string) => call<OODetail>('OODetail', keyword, songMid),
  ooQualities: (keyword: string, songMid: string) => call<QualityOption[]>('OOQualities', keyword, songMid),
  ooDownload: (req: OODownloadRequest) => call<string>('OODownload', req),
  ooBatchDownload: (keyword: string, quality: string) => call<string>('OOBatchDownload', keyword, quality),

  // 音乐库工具
  runLibraryFix: (req: LibraryFixRequest) => call<string>('RunLibraryFix', req),
}

export type Bindings = typeof bindings
