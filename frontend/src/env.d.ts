/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<Record<string, unknown>, Record<string, unknown>, unknown>
  export default component
}

interface WailsRuntime {
  EventsOn(eventName: string, callback: (...data: unknown[]) => void): () => void
  EventsOff(eventName: string): void
  EventsEmit(eventName: string, ...data: unknown[]): void
  Environment(): Promise<{ platform: string; arch: string }>
}

interface Window {
  runtime: WailsRuntime
  go: Record<string, Record<string, Record<string, (...args: unknown[]) => Promise<unknown>>>>
}
