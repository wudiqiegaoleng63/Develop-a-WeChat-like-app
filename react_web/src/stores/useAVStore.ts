import { create } from 'zustand'
import { wsService } from '../services/websocket'
import { MessageType } from '../types/message'
import type { ChatMessageRequest } from '../types/message'
import { showToast } from '../utils/toast'

export type CallStatus =
  | 'idle'
  | 'calling'
  | 'connected'
  | 'rejected'
  | 'peer_hangup'
  | 'create_offer'
  | 'handle_offer_sdp'
  | 'set_answer_sdp'
  | 'add_ice_candidate'

interface AVState {
  // Call signaling state — persists across modal open/close
  ableToReceiveOrReject: boolean
  ableToStartCall: boolean
  // Modal-local state — reset when modal closes
  callStatus: CallStatus
  remoteSdp: RTCSessionDescriptionInit | RTCIceCandidateInit | null

  setCallStatus: (status: CallStatus) => void
  setAbleToReceiveOrReject: (v: boolean) => void
  setAbleToStartCall: (v: boolean) => void
  setRemoteSdp: (sdp: RTCSessionDescriptionInit | RTCIceCandidateInit | null) => void
  resetModalState: () => void
  resetAllState: () => void
  handleAVMessage: (avDataStr: string, sendName: string) => void
  sendAVMessage: (sessionId: string, userInfo: { uuid: string; nickname: string; avatar: string }, contactId: string, avPayload: object) => void
}

export const useAVStore = create<AVState>((set, get) => ({
  callStatus: 'idle',
  ableToReceiveOrReject: false,
  ableToStartCall: true,
  remoteSdp: null,

  setCallStatus: (status) => set({ callStatus: status }),
  setAbleToReceiveOrReject: (v) => set({ ableToReceiveOrReject: v }),
  setAbleToStartCall: (v) => set({ ableToStartCall: v }),
  setRemoteSdp: (sdp) => set({ remoteSdp: sdp }),

  // Reset modal-local state only (callStatus, remoteSdp)
  // Does NOT reset ableToReceiveOrReject / ableToStartCall
  resetModalState: () => set({
    callStatus: 'idle',
    remoteSdp: null,
  }),

  // Full reset — called when a call actually ends (reject/hangup/peer_hangup)
  resetAllState: () => set({
    callStatus: 'idle',
    ableToReceiveOrReject: false,
    ableToStartCall: true,
    remoteSdp: null,
  }),

  handleAVMessage: (avDataStr, sendName) => {
    try {
      const avData = JSON.parse(avDataStr)

      if (avData.messageId === 'CURRENT_PEERS') {
        console.log('[AV] CURRENT_PEERS:', avData.messageData?.curContactList)
      } else if (avData.messageId === 'PEER_JOIN') {
        console.log('[AV] PEER_JOIN:', avData.messagecontactId)
      } else if (avData.messageId === 'PEER_LEAVE') {
        console.log('[AV] PEER_LEAVE')
        set({ callStatus: 'peer_hangup' })
      } else if (avData.messageId === 'PROXY') {
        if (avData.type === 'start_call') {
          console.log('[AV] start_call from', sendName)
          set({ ableToReceiveOrReject: true, ableToStartCall: false })
          showToast(`收到来自${sendName}的通话请求，请前往聊天室查看`, 'info')
        } else if (avData.type === 'receive_call') {
          console.log('[AV] receive_call — will create offer')
          set({ callStatus: 'create_offer' })
        } else if (avData.type === 'reject_call') {
          console.log('[AV] reject_call')
          set({ callStatus: 'rejected' })
        } else if (avData.type === 'sdp') {
          const sdp = avData.messageData?.sdp
          if (sdp?.type === 'offer') {
            console.log('[AV] offer SDP received')
            set({ callStatus: 'handle_offer_sdp', remoteSdp: sdp })
          } else if (sdp?.type === 'answer') {
            console.log('[AV] answer SDP received')
            set({ callStatus: 'set_answer_sdp', remoteSdp: sdp })
          }
        } else if (avData.type === 'candidate') {
          console.log('[AV] ICE candidate received')
          set({ callStatus: 'add_ice_candidate', remoteSdp: avData.messageData?.candidate })
        }
      }
    } catch (e) {
      console.error('[AV] Parse error:', e)
    }
  },

  sendAVMessage: (sessionId, userInfo, contactId, avPayload) => {
    const request: ChatMessageRequest = {
      session_id: sessionId,
      type: MessageType.AV,
      content: '',
      url: '',
      send_id: userInfo.uuid,
      send_name: userInfo.nickname,
      send_avatar: userInfo.avatar,
      receive_id: contactId,
      file_size: '',
      file_name: '',
      file_type: '',
      av_data: JSON.stringify(avPayload),
    }
    wsService.send(request)
  },
}))
