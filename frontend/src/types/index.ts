export type LogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'

export interface LogEntry {
  time: string
  level: LogLevel
  source: string
  message: string
  fields?: Record<string, string>
}

export type FieldType = 'string' | 'bool' | 'int' | 'float' | 'enum' | 'path'

export interface ConfigField {
  key: string
  label: string
  type: FieldType
  value: string | number | boolean
  default: string | number | boolean
  description: string
  options?: string[]
  min?: number
  max?: number
  required?: boolean
}

export interface ConfigGroup {
  key: string
  label: string
  description: string
  fields: ConfigField[]
}

export interface ConfigValues {
  [group: string]: { [field: string]: string | number | boolean }
}

export type TaskLane = 'workbench' | 'oomusic'
export type TaskStatus = 'queued' | 'running' | 'done' | 'failed' | 'cancelled' | 'removed'

export interface LaneState {
  lane: TaskLane
  running: string
  queued: number
}

/** 一个独立任务的完整生命周期记录；items 是最近若干条的预览。 */
export interface TaskEntry {
  task_id: string
  kind: string
  lane: TaskLane
  status: TaskStatus
  stage: string
  enqueued_at: number
  started_at: number
  finished_at: number
  elapsed_ms: number
  current_item: string
  current: number
  total: number
  success: number
  skipped: number
  failed: number
  cancelled: boolean
  error: string
  items: TaskItemResult[]
  item_count: number
  can_retry: boolean
}

export interface TaskBoard {
  lanes: LaneState[]
  tasks: TaskEntry[]
}

export interface TaskProgress {
  task_id: string
  kind: string
  stage: string
  status: 'running' | 'done' | ''
  current_item: string
  current: number
  total: number
  success: number
  skipped: number
  failed: number
  done: boolean
  cancelled: boolean
  error: string
  lane?: TaskLane
  enqueued_at?: number
  started_at?: number
  elapsed_ms?: number
  can_retry?: boolean
}

export interface QueuedTask {
  task_id: string
  kind: string
  total: number
  enqueued_at: number
  lane?: TaskLane
  status?: string
}

export interface TaskQueueSnapshot {
  current: TaskProgress | null
  queued: QueuedTask[]
  lanes?: LaneState[]
  tasks?: TaskEntry[]
}

export interface TaskItemResult {
  name: string
  status: 'success' | 'skipped' | 'failed'
  message: string
  elapsed_ms?: number
}

export interface TaskItemEvent {
  task_id: string
  item: TaskItemResult
  lane?: TaskLane
}

export interface TaskDone {
  task_id: string
  kind: string
  success: number
  skipped: number
  failed: number
  cancelled: boolean
  error: string
  items: TaskItemResult[]
  elapsed_ms: number
  lane?: TaskLane
  status?: string
  started_at?: number
  finished_at?: number
}

export interface DownloadRequest {
  kind: 'song' | 'album' | 'playlist'
  resource_id: string
  quality: string
  indexes: number[]
}

export interface SearchSong {
  id: string
  name: string
  artists: string
  album: string
  pic: string
  duration: number
}

export interface QualityOption {
  quality: string
  kbps: string
  size: string
  url: string
  ext: string
}

export interface OOSong {
  song_title: string
  singer_name: string
  song_mid: string
  pay: string
}

export interface OODetail {
  title: string
  artist: string
  album: string
  song_mid: string
  song_id: string
  duration_str: string
  vip: string
  language: string
  genre: string
  public_time: string
  quality: string
  kbps: string
  size_str: string
  file_extension: string
  cover_url: string
  lyric_lrc: string
  lyric_text: string
  quality_options: QualityOption[]
}

export interface OODownloadRequest {
  keyword: string
  song_mid: string
  quality: string
}

export interface CacheRegionStat {
  region: string
  label: string
  count: number
}

export interface CacheEntry {
  key: string
  preview: string
  bytes: number
}

export interface CacheDetail {
  formatted: string
  raw: string
}

export type LibraryFixKind = 'lyrics' | 'cover' | 'artists' | 'metadata'

export interface LibraryFixRequest {
  directory: string
  kind: LibraryFixKind
  only_missing: boolean
  force: boolean
  recursive: boolean
  continue_on_error: boolean
}

export interface FolderResult {
  folder: string
  success: boolean
  message: string
  elapsed_ms: number
}

export interface AppInfo {
  version: string
  config_path: string
  log_dir: string
  download_dir: string
  platform: string
}

export interface TaskRecord {
  id: number
  kind: string
  resource: string
  quality: string
  success: number
  skipped: number
  failed: number
  created_at: number
}

export const QUALITY_LEVELS = [
  { value: 'standard', label: '标准', hint: '128 kbps · 体积最小' },
  { value: 'exhigh', label: '极高', hint: '320 kbps · 均衡之选' },
  { value: 'lossless', label: '无损', hint: 'FLAC / WAV · CD 音质' },
  { value: 'hires', label: 'Hi-Res', hint: '高于 CD 的采样率' },
  { value: 'sky', label: '沉浸环绕', hint: '空间音频' },
  { value: 'jyeffect', label: '高清环绕', hint: '增强立体声' },
  { value: 'jymaster', label: '超清母带', hint: '录音室母带' },
] as const

export const OO_QUALITY_LEVELS = [
  { value: 'SQ', label: 'SQ 无损', hint: 'FLAC · 约 855 kbps' },
  { value: 'HQ', label: 'HQ 高品质', hint: 'M4A · 约 193 kbps' },
  { value: 'standard', label: '标准', hint: 'M4A · 约 97 kbps' },
  { value: 'low', label: '低品质', hint: 'M4A · 约 48 kbps' },
] as const

export const LIBRARY_TOOLS: { kind: LibraryFixKind; title: string; description: string }[] = [
  {
    kind: 'lyrics',
    title: '歌词修复',
    description: '按专辑 ID 拉取完整歌词并嵌入标签，补写 CMUSIC_ID；已处理的专辑自动跳过。',
  },
  {
    kind: 'cover',
    title: '封面重下',
    description: '检查 cover.jpg 分辨率，低于阈值的重新从专辑接口获取高清封面。',
  },
  {
    kind: 'artists',
    title: '艺术家拆分',
    description: '把含 / 的艺术家标签拆为多值，并按序号格式重建文件名。',
  },
  {
    kind: 'metadata',
    title: '元数据补全',
    description: '仅对缺失标题、艺术家或专辑的文件，从文件名与专辑信息回填标签与封面。',
  },
]
