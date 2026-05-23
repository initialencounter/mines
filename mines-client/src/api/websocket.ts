import type { GameRequest, GameResponse } from '@/types'

type MessageHandler = (data: GameResponse) => void

class WsClient {
  private ws: WebSocket | null = null
  private handlers: Set<MessageHandler> = new Set()
  private reconnectTimer: number | null = null
  private url: string = ''

  connect(url: string): void {
    this.url = url
    this.ws = new WebSocket(url)

    this.ws.onopen = () => {
      console.log('WebSocket connected')
    }

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as GameResponse
        this.handlers.forEach(h => h(data))
      }
      catch (e) {
        console.error('Failed to parse WS message:', e)
      }
    }

    this.ws.onclose = () => {
      console.log('WebSocket disconnected')
      // Auto-reconnect after 2s
      this.reconnectTimer = window.setTimeout(() => {
        if (this.url)
          this.connect(this.url)
      }, 2000)
    }

    this.ws.onerror = (e) => {
      console.error('WebSocket error:', e)
    }
  }

  onMessage(handler: MessageHandler): void {
    this.handlers.add(handler)
  }

  offMessage(handler: MessageHandler): void {
    this.handlers.delete(handler)
  }

  send(data: GameRequest): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data))
    }
    else {
      console.warn('WebSocket not connected')
    }
  }

  close(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.handlers.clear()
    this.url = ''
    this.ws?.close()
    this.ws = null
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }
}

export const wsClient = new WsClient()
