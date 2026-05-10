import type { ChatMessage, ChatMessageRequest } from '../types/message'

type MessageHandler = (msg: ChatMessage) => void

class WebSocketService {
  private ws: WebSocket | null = null
  private messageHandlers: Set<MessageHandler> = new Set()
  private reconnectAttempts = 0
  private maxReconnectAttempts = 10
  private clientId: string = ''
  private wsBaseUrl: string = ''
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private isConnecting = false
  private intentionalClose = false

  connect(clientId: string, wsBaseUrl: string): void {
    // Prevent duplicate connections
    if (this.isConnecting || (this.ws && this.ws.readyState === WebSocket.OPEN && this.clientId === clientId)) {
      return
    }
    this.clientId = clientId
    this.wsBaseUrl = wsBaseUrl
    this.reconnectAttempts = 0
    this.intentionalClose = false
    this.doConnect()
  }

  private doConnect(): void {
    if (this.isConnecting) return
    this.isConnecting = true

    if (this.ws) {
      this.ws.onclose = null // Prevent reconnect on intentional close
      this.ws.close()
      this.ws = null
    }

    const url = `${this.wsBaseUrl}/user/wsLogin?client_id=${this.clientId}`
    console.log('[WS] Connecting to:', url)

    try {
      this.ws = new WebSocket(url)

      this.ws.onopen = () => {
        console.log('[WS] Connected')
        this.reconnectAttempts = 0
        this.isConnecting = false
      }

      this.ws.onmessage = (event) => {
        // Skip non-JSON messages (e.g. welcome text from server)
        if (typeof event.data === 'string' && !event.data.startsWith('{')) {
          console.log('[WS] Server:', event.data)
          return
        }
        try {
          const message = JSON.parse(event.data) as ChatMessage
          this.messageHandlers.forEach(handler => handler(message))
        } catch (e) {
          console.error('[WS] Parse error:', e)
        }
      }

      this.ws.onclose = (event) => {
        console.log('[WS] Disconnected, code:', event.code)
        this.isConnecting = false
        this.ws = null
        // Only reconnect if not an intentional close
        if (!this.intentionalClose) {
          this.attemptReconnect()
        }
      }

      this.ws.onerror = (err) => {
        console.error('[WS] Error:', err)
        this.isConnecting = false
      }
    } catch (e) {
      console.error('[WS] Connection failed:', e)
      this.isConnecting = false
      this.attemptReconnect()
    }
  }

  private attemptReconnect(): void {
    if (this.reconnectTimer) return
    if (this.intentionalClose) return
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log('[WS] Max reconnect attempts reached')
      return
    }

    this.reconnectAttempts++
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000)
    console.log(`[WS] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`)

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.doConnect()
    }, delay)
  }

  send(data: ChatMessageRequest): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data))
    } else {
      console.error('[WS] Not connected, cannot send message')
    }
  }

  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler)
    return () => {
      this.messageHandlers.delete(handler)
    }
  }

  disconnect(): void {
    this.intentionalClose = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.ws) {
      this.ws.onclose = null
      this.ws.close(1000)
      this.ws = null
    }
    this.messageHandlers.clear()
  }

  get connected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }
}

export const wsService = new WebSocketService()
