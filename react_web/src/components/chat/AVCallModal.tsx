import React, { useRef, useState, useEffect, useCallback } from 'react'
import { useAuthStore } from '../../stores/useAuthStore'
import { useChatStore } from '../../stores/useChatStore'
import { showToast } from '../../utils/toast'

interface AVCallModalProps {
  visible: boolean
  onClose: () => void
}

type CallStatus = 'idle' | 'calling' | 'connected' | 'rejected'

export default function AVCallModal({ visible, onClose }: AVCallModalProps) {
  const userInfo = useAuthStore(state => state.userInfo)
  const contactInfo = useChatStore(state => state.contactInfo)
  const [callStatus, setCallStatus] = useState<CallStatus>('idle')
  const [hasLocalStream, setHasLocalStream] = useState(false)

  const localVideoRef = useRef<HTMLVideoElement>(null)
  const remoteVideoRef = useRef<HTMLVideoElement>(null)
  const peerConnectionRef = useRef<RTCPeerConnection | null>(null)
  const localStreamRef = useRef<MediaStream | null>(null)
  const rejectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const cleanup = useCallback(() => {
    if (localStreamRef.current) {
      localStreamRef.current.getTracks().forEach(track => track.stop())
      localStreamRef.current = null
    }
    if (peerConnectionRef.current) {
      peerConnectionRef.current.close()
      peerConnectionRef.current = null
    }
    if (localVideoRef.current) localVideoRef.current.srcObject = null
    if (remoteVideoRef.current) remoteVideoRef.current.srcObject = null
    if (rejectTimerRef.current) {
      clearTimeout(rejectTimerRef.current)
      rejectTimerRef.current = null
    }
    setHasLocalStream(false)
  }, [])

  // Reset state when modal opens/closes
  useEffect(() => {
    if (visible) {
      setCallStatus('idle')
      setHasLocalStream(false)
    } else {
      cleanup()
      setCallStatus('idle')
    }
  }, [visible, cleanup])

  if (!visible || !contactInfo || !userInfo) return null

  const isGroup = contactInfo.contact_id.startsWith('G')

  const handleStartCall = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true })
      localStreamRef.current = stream
      if (localVideoRef.current) {
        localVideoRef.current.srcObject = stream
      }
      setHasLocalStream(true)
      setCallStatus('calling')
      // TODO: Send start_call signaling via WebSocket
    } catch (err) {
      showToast('无法访问摄像头/麦克风', 'error')
    }
  }

  const handleAnswer = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true })
      localStreamRef.current = stream
      if (localVideoRef.current) {
        localVideoRef.current.srcObject = stream
      }
      setHasLocalStream(true)
      setCallStatus('connected')
      // TODO: Send receive_call signaling via WebSocket
    } catch (err) {
      showToast('无法访问摄像头/麦克风', 'error')
    }
  }

  const handleReject = () => {
    setCallStatus('rejected')
    // TODO: Send reject_call signaling via WebSocket
    rejectTimerRef.current = setTimeout(() => {
      rejectTimerRef.current = null
      cleanup()
      onClose()
    }, 1500)
  }

  const handleHangup = () => {
    // TODO: Send PEER_LEAVE signaling via WebSocket
    cleanup()
    onClose()
  }

  const handleClose = () => {
    if (callStatus === 'calling' || callStatus === 'connected') {
      showToast('请先结束通话', 'info')
      return
    }
    cleanup()
    onClose()
  }

  const statusText: Record<CallStatus, string> = {
    idle: '准备就绪',
    calling: '等待对方接听...',
    connected: '通话中',
    rejected: '对方已拒绝',
  }

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
                <img src={userInfo.avatar} className="video-avatar" alt="我" />
                <div className="video-name">{userInfo.nickname}</div>
              </>
            )}
            <div className="av-video-label">本地</div>
          </div>
          <div className="av-video-box">
            <video ref={remoteVideoRef} autoPlay playsInline style={{ display: callStatus === 'connected' ? 'block' : 'none' }} />
            {callStatus !== 'connected' && (
              <div className="video-placeholder">
                <img src={contactInfo.contact_avatar} className="video-avatar" alt={contactInfo.contact_name} style={{opacity:0.4}} />
                <div style={{color:'rgba(255,255,255,0.4)',marginTop:8}}>等待连接</div>
              </div>
            )}
            <div className="av-video-label">远程</div>
          </div>
        </div>
        <div className="av-status">
          <span className={`status-dot-inline ${callStatus}`} />
          {statusText[callStatus]}
        </div>
        <div className="av-modal-footer">
          <button className="av-btn av-btn-start" onClick={handleStartCall} disabled={callStatus !== 'idle'}>
            <span>📞</span>
            <span className="av-btn-label">发起</span>
          </button>
          <button className="av-btn av-btn-answer" onClick={handleAnswer} disabled={callStatus !== 'calling'}>
            <span>📱</span>
            <span className="av-btn-label">接听</span>
          </button>
          <button className="av-btn av-btn-reject" onClick={handleReject} disabled={callStatus !== 'calling'}>
            <span>❌</span>
            <span className="av-btn-label">拒绝</span>
          </button>
          <button className="av-btn av-btn-hangup" onClick={handleHangup} disabled={callStatus === 'idle'}>
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
