import { useEffect } from 'react'
import { wsService } from '../services/websocket'
import { useChatStore } from '../stores/useChatStore'
import { useAuthStore } from '../stores/useAuthStore'
import { useAVStore } from '../stores/useAVStore'

export function useWebSocket() {
  const addIncomingMessage = useChatStore(state => state.addIncomingMessage)
  const userInfo = useAuthStore(state => state.userInfo)

  useEffect(() => {
    if (!userInfo) return

    const unsubscribe = wsService.onMessage((message) => {
      if (message.type === 3) {
        // AV/WebRTC signaling message
        const handleAVMessage = useAVStore.getState().handleAVMessage
        handleAVMessage(message.av_data || '', message.send_name)
      } else {
        addIncomingMessage(message, userInfo.uuid)
      }
    })
    return unsubscribe
  }, [addIncomingMessage, userInfo])
}
