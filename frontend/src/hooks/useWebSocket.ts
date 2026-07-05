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
      console.log('[WS] Received message, type:', message.type, typeof message.type, 'av_data:', message.av_data)
      // Use loose comparison to handle both number 3 and string "3"
      if (message.type == 3) {
        console.log('[WS] AV message detected, av_data:', message.av_data)
        const handleAVMessage = useAVStore.getState().handleAVMessage
        handleAVMessage(message.av_data || '', message.send_name)
      } else {
        addIncomingMessage(message, userInfo.uuid)
      }
    })
    return unsubscribe
  }, [addIncomingMessage, userInfo])
}
