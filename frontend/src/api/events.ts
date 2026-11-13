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

/** 订阅 Wails 事件，返回取消订阅函数。浏览器预览时静默降级。 */
export function on<T>(event: string, handler: Handler<T>): () => void {
  const rt = (window as unknown as { runtime?: {
    EventsOn: (name: string, cb: (...data: unknown[]) => void) => unknown
    EventsOff: (name: string) => void
  } }).runtime
  if (!rt || typeof rt.EventsOn !== 'function') {
    return () => {}
  }
  const off = rt.EventsOn(event, (...data: unknown[]) => {
    handler(data[0] as T)
  })
  return () => {
    if (typeof off === 'function') {
      ;(off as () => void)()
    } else {
      rt.EventsOff(event)
    }
  }
}

export function emit<T>(event: string, payload: T): void {
  const rt = (window as unknown as { runtime?: { EventsEmit: (name: string, ...data: unknown[]) => void } }).runtime
  if (rt && typeof rt.EventsEmit === 'function') {
    rt.EventsEmit(event, payload)
  }
}
