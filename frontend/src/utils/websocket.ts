import type {
  StatusData,
  HostMetrics,
  WorkspaceItem,
  ConversationMeta,
  SnapshotSummary,
  SnapshotProgressTask
} from '../types/api'
import { resolveApiPrefix } from './http'

export type WSEventMap = {
  status: StatusData
  usage: { cpu: string; memory: string; host?: HostMetrics }
  workspace: { items: WorkspaceItem[] }
  conversations: ConversationMeta[]
  snapshot: SnapshotSummary
  snapshot_progress: SnapshotProgressTask
  log: string
  pong: { server_time: number }
  connectionChange: boolean
  reconnected: void
}

type EventCallback<K extends keyof WSEventMap> = (data: WSEventMap[K]) => void

class WSManager {
  private ws: WebSocket | null = null
  private isConnected = false
  private openedOnce = false
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private retryCount = 0
  private listeners: { [K in keyof WSEventMap]?: Set<EventCallback<K>> } = {}

  private resolveWsUrl(): string {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${window.location.host}${resolveApiPrefix()}/ws`
  }

  public connect(): void {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return
    }
    try {
      this.ws = new WebSocket(this.resolveWsUrl())
    } catch {
      this.handleClose()
      return
    }

    this.ws.onopen = () => {
      this.isConnected = true
      this.retryCount = 0
      this.emit('connectionChange', true)
      if (this.openedOnce) {
        this.emit('reconnected', undefined as unknown as void)
      }
      this.openedOnce = true
      this.startHeartbeat()
    }

    this.ws.onmessage = (e: MessageEvent) => {
      if (typeof e.data !== 'string') return
      try {
        const env = JSON.parse(e.data) as { type: keyof WSEventMap; data: unknown }
        if (env && env.type) {
          this.emit(env.type, env.data as never)
        }
      } catch {
        // ignore invalid json
      }
    }

    this.ws.onclose = () => this.handleClose()
    this.ws.onerror = () => this.ws?.close()
  }

  private handleClose(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
    this.ws = null
    if (this.isConnected) {
      this.isConnected = false
      this.emit('connectionChange', false)
    }
    if (!this.reconnectTimer) {
      const delay = Math.min(1000 * Math.pow(1.5, this.retryCount), 10000)
      this.retryCount++
      this.reconnectTimer = setTimeout(() => {
        this.reconnectTimer = null
        this.connect()
      }, delay)
    }
  }

  private startHeartbeat(): void {
    if (this.heartbeatTimer) clearInterval(this.heartbeatTimer)
    this.heartbeatTimer = setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: 'ping' }))
      }
    }, 12000)
  }

  public on<K extends keyof WSEventMap>(event: K, callback: EventCallback<K>): () => void {
    if (!this.listeners[event]) {
      this.listeners[event] = new Set() as never
    }
    const set = this.listeners[event] as Set<EventCallback<K>>
    set.add(callback)
    return () => set.delete(callback)
  }

  private emit<K extends keyof WSEventMap>(event: K, data: WSEventMap[K]): void {
    const set = this.listeners[event] as Set<EventCallback<K>> | undefined
    set?.forEach((cb) => {
      try {
        cb(data)
      } catch {
        // isolate listener error
      }
    })
  }
}

export const wsClient = new WSManager()
