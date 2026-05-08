import { useEffect } from 'react'
import { wsService } from '../services/websocket'
import { useChatStore } from '../stores/useChatStore'
import { useAuthStore } from '../stores/useAuthStore'

export function useWebSocket() {
  const addIncomingMessage = useChatStore(state => state.addIncomingMessage)
  const userInfo = useAuthStore(state => state.userInfo)

  useEffect(() => {
    if (!userInfo) return

    const unsubscribe = wsService.onMessage((message) => {
      // Only handle non-AV messages (type 3 is AV/WebRTC signaling)
      if (message.type !== 3) {
        addIncomingMessage(message, userInfo.uuid)
      }
    })
    return unsubscribe
  }, [addIncomingMessage, userInfo])
}
