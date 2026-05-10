import React, { useRef, useState, useEffect, useCallback } from 'react'
import { useAuthStore } from '../../stores/useAuthStore'
import { useChatStore } from '../../stores/useChatStore'
import { useAVStore } from '../../stores/useAVStore'
import { showToast } from '../../utils/toast'
import { normalizeAvatarUrl } from '../../utils/avatar'

interface AVCallModalProps {
  visible: boolean
  onClose: () => void
}

export default function AVCallModal({ visible, onClose }: AVCallModalProps) {
  const userInfo = useAuthStore(state => state.userInfo)
  const contactInfo = useChatStore(state => state.contactInfo)
  const activeSessionId = useChatStore(state => state.activeSessionId)
  const {
    callStatus, ableToReceiveOrReject, ableToStartCall, remoteSdp,
    setCallStatus, setAbleToReceiveOrReject, setAbleToStartCall, setRemoteSdp,
    resetModalState, resetAllState, sendAVMessage,
  } = useAVStore()

  const [hasLocalStream, setHasLocalStream] = useState(false)
  const [hasRemoteStream, setHasRemoteStream] = useState(false)

  const localVideoRef = useRef<HTMLVideoElement>(null)
  const remoteVideoRef = useRef<HTMLVideoElement>(null)
  const peerConnectionRef = useRef<RTCPeerConnection | null>(null)
  const localStreamRef = useRef<MediaStream | null>(null)
  const remoteStreamRef = useRef<MediaStream | null>(null)

  const sessionId = activeSessionId || ''

  const cleanup = useCallback(() => {
    if (localStreamRef.current) {
      localStreamRef.current.getTracks().forEach(track => track.stop())
      localStreamRef.current = null
    }
    if (peerConnectionRef.current) {
      peerConnectionRef.current.close()
      peerConnectionRef.current = null
    }
    remoteStreamRef.current = null
    if (localVideoRef.current) localVideoRef.current.srcObject = null
    if (remoteVideoRef.current) remoteVideoRef.current.srcObject = null
    setHasLocalStream(false)
    setHasRemoteStream(false)
  }, [])

  // When modal closes, only cleanup media streams + reset modal state
  // Do NOT reset ableToReceiveOrReject / ableToStartCall
  useEffect(() => {
    if (!visible) {
      cleanup()
      resetModalState()
    }
  }, [visible, cleanup, resetModalState])

  const getUserInfoForAV = () => ({
    uuid: userInfo?.uuid || '',
    nickname: userInfo?.nickname || '',
    avatar: userInfo?.avatar || '',
  })

  const createRtcPeerConnection = () => {
    if (peerConnectionRef.current) return

    const pc = new RTCPeerConnection({})

    pc.onicecandidate = (event) => {
      if (event.candidate && contactInfo) {
        sendAVMessage(sessionId, getUserInfoForAV(), contactInfo.contact_id, {
          messageId: 'PROXY',
          type: 'candidate',
          messageData: { candidate: event.candidate },
        })
      }
    }

    pc.oniceconnectionstatechange = () => {
      console.log('[AV] ICE state:', pc.iceConnectionState)
    }

    pc.ontrack = (event) => {
      if (!remoteStreamRef.current) {
        remoteStreamRef.current = new MediaStream()
        if (remoteVideoRef.current) {
          remoteVideoRef.current.srcObject = remoteStreamRef.current
        }
        setHasRemoteStream(true)
      }
      remoteStreamRef.current.addTrack(event.track)
    }

    peerConnectionRef.current = pc
  }

  const closeRtcPeerConnection = () => {
    if (peerConnectionRef.current) {
      peerConnectionRef.current.close()
      peerConnectionRef.current = null
    }
  }

  const getLocalMediaStream = async (): Promise<MediaStream | null> => {
    if (localStreamRef.current) return localStreamRef.current
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true })
      localStreamRef.current = stream
      if (localVideoRef.current) {
        localVideoRef.current.srcObject = stream
        localVideoRef.current.muted = true
      }
      setHasLocalStream(true)
      return stream
    } catch {
      showToast('无法访问摄像头/麦克风', 'error')
      return null
    }
  }

  const closeLocalMediaStream = () => {
    if (localStreamRef.current) {
      localStreamRef.current.getTracks().forEach(track => track.stop())
      localStreamRef.current = null
    }
    if (localVideoRef.current) localVideoRef.current.srcObject = null
    setHasLocalStream(false)
  }

  const attachTracks = (stream: MediaStream) => {
    if (peerConnectionRef.current) {
      stream.getTracks().forEach(track => peerConnectionRef.current!.addTrack(track))
    }
  }

  const createOffer = () => {
    if (!peerConnectionRef.current || !contactInfo) return
    peerConnectionRef.current.createOffer({ offerToReceiveAudio: true, offerToReceiveVideo: true })
      .then(desc => {
        peerConnectionRef.current!.setLocalDescription(desc)
        sendAVMessage(sessionId, getUserInfoForAV(), contactInfo.contact_id, {
          messageId: 'PROXY',
          type: 'sdp',
          messageData: { sdp: desc },
        })
      })
      .catch(err => console.error('[AV] createOffer failed:', err))
  }

  const createAnswer = () => {
    if (!peerConnectionRef.current || !contactInfo) return
    peerConnectionRef.current.createAnswer()
      .then(desc => {
        peerConnectionRef.current!.setLocalDescription(desc)
        sendAVMessage(sessionId, getUserInfoForAV(), contactInfo.contact_id, {
          messageId: 'PROXY',
          type: 'sdp',
          messageData: { sdp: desc },
        })
      })
      .catch(err => console.error('[AV] createAnswer failed:', err))
  }

  // Handle WebRTC signaling state changes
  useEffect(() => {
    if (!visible || !contactInfo || !userInfo) return

    if (callStatus === 'create_offer') {
      createOffer()
      setCallStatus('calling')
    } else if (callStatus === 'handle_offer_sdp' && remoteSdp) {
      if (peerConnectionRef.current) {
        peerConnectionRef.current.setRemoteDescription(new RTCSessionDescription(remoteSdp as RTCSessionDescriptionInit))
          .then(() => createAnswer())
          .catch(err => console.error('[AV] setRemoteDescription(offer) failed:', err))
      }
      setRemoteSdp(null)
      setCallStatus('connected')
    } else if (callStatus === 'set_answer_sdp' && remoteSdp) {
      if (peerConnectionRef.current) {
        peerConnectionRef.current.setRemoteDescription(new RTCSessionDescription(remoteSdp as RTCSessionDescriptionInit))
          .catch(err => console.error('[AV] setRemoteDescription(answer) failed:', err))
      }
      setRemoteSdp(null)
    } else if (callStatus === 'add_ice_candidate' && remoteSdp) {
      if (peerConnectionRef.current) {
        peerConnectionRef.current.addIceCandidate(new RTCIceCandidate(remoteSdp as RTCIceCandidateInit))
          .catch(err => console.error('[AV] addIceCandidate failed:', err))
      }
      setRemoteSdp(null)
    } else if (callStatus === 'rejected') {
      showToast('对方拒绝通话', 'error')
      closeLocalMediaStream()
      closeRtcPeerConnection()
      resetAllState()
    } else if (callStatus === 'peer_hangup') {
      showToast('对方已挂断', 'info')
      closeLocalMediaStream()
      closeRtcPeerConnection()
      resetAllState()
    }
  }, [callStatus, visible, contactInfo, userInfo])

  // === Button handlers ===

  const handleStartCall = async () => {
    if (hasLocalStream) {
      showToast('已经在通话中，请勿重复发起', 'info')
      return
    }
    if (!ableToStartCall) {
      showToast('对方已经发起通话，请先接收通话或拒绝通话', 'info')
      return
    }

    createRtcPeerConnection()
    const stream = await getLocalMediaStream()
    if (!stream) return
    attachTracks(stream)

    sendAVMessage(sessionId, getUserInfoForAV(), contactInfo!.contact_id, {
      messageId: 'PROXY',
      type: 'start_call',
    })
    setCallStatus('calling')
  }

  const handleAnswer = async () => {
    if (!ableToReceiveOrReject) {
      showToast('对方没有发起通话或已在通话中，无法接收通话', 'info')
      return
    }

    createRtcPeerConnection()
    const stream = await getLocalMediaStream()
    if (!stream) return
    attachTracks(stream)

    sendAVMessage(sessionId, getUserInfoForAV(), contactInfo!.contact_id, {
      messageId: 'PROXY',
      type: 'receive_call',
    })
    setAbleToReceiveOrReject(false)
    setAbleToStartCall(true)
  }

  const handleReject = () => {
    if (!ableToReceiveOrReject) {
      showToast('对方没有发起通话或已在通话中，无法拒绝通话', 'info')
      return
    }
    sendAVMessage(sessionId, getUserInfoForAV(), contactInfo!.contact_id, {
      messageId: 'PROXY',
      type: 'reject_call',
    })
    setAbleToReceiveOrReject(false)
    setAbleToStartCall(true)
    cleanup()
    setCallStatus('idle')
  }

  const handleHangup = () => {
    if (!hasLocalStream && !hasRemoteStream) {
      showToast('尚未开始通话，无法挂断', 'info')
      return
    }

    sendAVMessage(sessionId, getUserInfoForAV(), contactInfo!.contact_id, {
      messageId: 'PEER_LEAVE',
    })

    closeLocalMediaStream()
    closeRtcPeerConnection()
    remoteStreamRef.current = null
    resetAllState()
  }

  const handleClose = () => {
    if (hasLocalStream || hasRemoteStream) {
      showToast('请先结束通话', 'info')
      return
    }
    cleanup()
    resetModalState()
    onClose()
  }

  // Early return AFTER all hooks
  if (!visible || !contactInfo || !userInfo) return null

  const isGroup = contactInfo.contact_id.startsWith('G')

  const statusLabels: Record<string, string> = {
    idle: '准备就绪',
    calling: '等待对方接听...',
    connected: '通话中',
    rejected: '对方已拒绝',
    peer_hangup: '对方已挂断',
    waiting_offer: '等待连接...',
  }

  const displayStatus = callStatus === 'create_offer' ? 'calling'
    : callStatus === 'handle_offer_sdp' ? 'connected'
    : callStatus === 'set_answer_sdp' ? 'connected'
    : callStatus === 'add_ice_candidate' ? 'connected'
    : callStatus === 'peer_hangup' ? 'idle'
    : callStatus

  return (
    <div className="av-modal-overlay" onClick={handleClose}>
      <div className="av-modal" onClick={e => e.stopPropagation()}>
        <div className="av-modal-header">
          <h3>{isGroup ? '👥 群组通话' : '📞 音视频通话'}</h3>
          <button className="av-close-btn" onClick={handleClose}>✕</button>
        </div>
        <div className="av-modal-body">
          <div className="av-video-box">
            <video ref={localVideoRef} autoPlay muted playsInline style={{ display: hasLocalStream ? 'block' : 'none' }} />
            {!hasLocalStream && (
              <>
                <img src={normalizeAvatarUrl(userInfo.avatar || '')} className="video-avatar" alt="我" />
                <div className="video-name">{userInfo.nickname}</div>
              </>
            )}
            <div className="av-video-label">本地</div>
          </div>
          <div className="av-video-box">
            <video ref={remoteVideoRef} autoPlay playsInline style={{ display: hasRemoteStream ? 'block' : 'none' }} />
            {!hasRemoteStream && (
              <div className="video-placeholder">
                <img src={normalizeAvatarUrl(contactInfo.contact_avatar)} className="video-avatar" alt={contactInfo.contact_name} style={{opacity:0.4}} />
                <div style={{color:'rgba(255,255,255,0.4)',marginTop:8}}>等待连接</div>
              </div>
            )}
            <div className="av-video-label">远程</div>
          </div>
        </div>
        <div className="av-status">
          <span className={`status-dot-inline ${displayStatus}`} />
          {statusLabels[displayStatus] || '准备就绪'}
        </div>
        <div className="av-modal-footer">
          <button className="av-btn av-btn-start" onClick={handleStartCall} disabled={!ableToStartCall || callStatus === 'calling' || callStatus === 'connected'}>
            <span>📞</span>
            <span className="av-btn-label">发起</span>
          </button>
          <button className="av-btn av-btn-answer" onClick={handleAnswer} disabled={!ableToReceiveOrReject}>
            <span>📱</span>
            <span className="av-btn-label">接听</span>
          </button>
          <button className="av-btn av-btn-reject" onClick={handleReject} disabled={!ableToReceiveOrReject}>
            <span>❌</span>
            <span className="av-btn-label">拒绝</span>
          </button>
          <button className="av-btn av-btn-hangup" onClick={handleHangup} disabled={!hasLocalStream && !hasRemoteStream}>
            <span>📵</span>
            <span className="av-btn-label">挂断</span>
          </button>
          <button className="av-btn av-btn-exit" onClick={handleClose}>
            <span>🚪</span>
            <span className="av-btn-label">退出</span>
          </button>
        </div>
      </div>
    </div>
  )
}
