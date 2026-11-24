import { Call } from '@wailsio/runtime'

import type {
  AppInfo,
  CacheEntry,
  CacheDetail,
  CacheRegionStat,
  ConfigGroup,
  ConfigValues,
  DownloadRequest,
  LibraryFixRequest,
  LogEntry,
  OODetail,
  OODownloadRequest,
  OOSong,
  QualityOption,
  TaskBoard,
  TaskDone,
  TaskQueueSnapshot,
  TaskProgress,
  TaskRecord,
} from '@/types'

/** 统一调用 Wails 绑定方法，并归一化错误。 */
export async function call<T>(method: string, ...args: unknown[]): Promise<T> {
  try {
    return (await Call.ByName(`main.App.${method}`, ...args)) as T
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
  setLogLevel: (level: string) => call<void>('SetLogLevel', level),
  exportLogs: () => call<string>('ExportLogs'),
  openLogDir: () => call<void>('OpenLogDir'),

  // 系统
  getAppInfo: () => call<AppInfo>('GetAppInfo'),
  getTaskStatus: () => call<TaskProgress>('GetTaskStatus'),
  getTaskBoard: () => call<TaskBoard>('GetTaskBoard'),
  getTaskQueue: () => call<TaskQueueSnapshot>('GetTaskQueue'),
  removeQueuedTask: (taskID: string) => call<boolean>('RemoveQueuedTask', taskID),
  clearTaskQueue: () => call<number>('ClearTaskQueue'),
  getTaskHistory: (limit: number) => call<TaskRecord[]>('GetTaskHistory', limit),
  // CancelTask 现在取消两条通道里全部运行中的任务。
  cancelTask: () => call<boolean>('CancelTask'),
  cancelTaskById: (taskID: string) => call<boolean>('CancelTaskByID', taskID),
  retryFailedItems: (taskID: string) => call<string>('RetryFailedItems', taskID),
  openDownloadDir: () => call<void>('OpenDownloadDir'),
  openPath: (path: string) => call<void>('OpenPath', path),
  selectDirectory: (title: string) => call<string>('SelectDirectory', title),
  listSubFolders: (dir: string) => call<string[]>('ListSubFolders', dir),

  // 下载工作台
  getLastPreference: () => call<LastPreference>('GetLastDownloadConfig'),
  saveLastPreference: (p: LastPreference) => call<void>('SaveLastDownloadConfig', p),
  download: (req: DownloadRequest) => call<string>('Download', req),
  parseIndexes: (raw: string) => call<number[]>('ParseIndexes', raw),
  extractResourceID: (raw: string) => call<string>('ExtractResourceID', raw),

  // OO Music
  ooSearch: (keyword: string) => call<OOSong[]>('OOSearch', keyword),
  ooDetail: (keyword: string, songMid: string) => call<OODetail>('OODetail', keyword, songMid),
  ooQualities: (keyword: string, songMid: string) => call<QualityOption[]>('OOQualities', keyword, songMid),
  ooDownload: (req: OODownloadRequest) => call<string>('OODownload', req),
  ooBatchDownload: (keyword: string, quality: string) => call<string>('OOBatchDownload', keyword, quality),

  // 缓存检视
  getCacheStats: () => call<CacheRegionStat[]>('GetCacheStats'),
  listCache: (region: string) => call<CacheEntry[]>('ListCache', region),
  getCacheDetail: (region: string, key: string) => call<CacheDetail>('GetCacheDetail', region, key),
  clearCache: (region: string, key: string) => call<number>('ClearCache', region, key),

  // 音乐库工具
  runLibraryFix: (req: LibraryFixRequest) => call<string>('RunLibraryFix', req),
}

export type Bindings = typeof bindings
