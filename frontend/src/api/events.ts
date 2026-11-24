import { Events } from '@wailsio/runtime'

export const EventNames = {
  LOG_BATCH: 'log:batch',
  TASK_PROGRESS: 'task:progress',
  TASK_ITEM: 'task:item',
  TASK_DONE: 'task:done',
  TASK_QUEUED: 'task:queued',
  CONFIG_CHANGED: 'config:changed',
  APP_READY: 'app:ready',
} as const

type Handler<T> = (payload: T) => void

/** 订阅 Wails 事件，返回取消订阅函数。浏览器预览时收不到事件但不会抛错。 */
export function on<T>(event: string, handler: Handler<T>): () => void {
  // v3 回调收到 WailsEvent 对象，payload 在 event.data 上（v2 是 (...data) 取 data[0]）。
  const off = Events.On(event, (e: { data?: unknown }) => handler(e?.data as T))
  return () => {
    if (typeof off === 'function') {
      off()
    }
  }
}

export function emit<T>(event: string, payload: T): void {
  void Events.Emit(event, payload)
}
