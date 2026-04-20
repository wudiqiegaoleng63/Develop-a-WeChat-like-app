<template>
  <div class="chat-wrap">
    <div class="chat-window">
      <el-container class="chat-window-container">
        <el-aside class="aside-container">
          <div class="nav-container">
            <div class="nav-item" @click="goToSession">
              <el-icon><ChatDotRound /></el-icon>
            </div>
            <div class="nav-item" @click="goToContact">
              <el-icon><User /></el-icon>
            </div>
            <div class="nav-item" @click="goToOwnInfo">
              <el-icon><HomeFilled /></el-icon>
            </div>
            <el-dropdown placement="top" trigger="click">
              <div class="nav-item">
                <el-icon><Setting /></el-icon>
              </div>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item v-if="isAdmin" @click="goToManager">进入管理</el-dropdown-item>
                  <el-dropdown-item @click="logout">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
          <div class="contact-info-container">
            <img :src="contactInfo.avatar" class="contact-avatar" />
            <div class="contact-detail">
              <span class="contact-name">{{ contactInfo.nickname }}</span>
              <span class="contact-id">{{ contactInfo.contact_id }}</span>
            </div>
          </div>
        </el-aside>
        <el-main class="main-container">
          <div class="chat-header">
            <span>{{ contactInfo.nickname }}</span>
            <div class="chat-actions">
              <el-button v-if="!isGroup" :disabled="!ableToStartCall" type="primary" size="small" @click="startCall">
                <el-icon><VideoCamera /></el-icon>
              </el-button>
              <el-button v-if="!isGroup && ableToReceiveOrRejectCall" type="success" size="small" @click="receiveCall">接听</el-button>
              <el-button v-if="!isGroup && ableToReceiveOrRejectCall" type="danger" size="small" @click="rejectCall">拒绝</el-button>
              <el-button v-if="!isGroup && (localVideo || remoteVideo)" type="danger" size="small" @click="sendEndCall">挂断</el-button>
            </div>
          </div>
          <div class="video-container" v-if="localVideo || remoteVideo">
            <video id="localVideo" ref="localVideoRef" autoplay muted playsinline></video>
            <video id="remoteVideo" ref="remoteVideoRef" autoplay playsinline></video>
          </div>
          <el-scrollbar ref="scrollbarRef" class="message-list-container">
            <div v-for="msg in messageList" :key="msg.id || msg.created_at" class="message-item">
              <div :class="msg.send_id === userInfo.uuid ? 'message-self' : 'message-other'">
                <img :src="msg.send_avatar" class="message-avatar" />
                <div class="message-content">
                  <span class="message-name">{{ msg.send_name }}</span>
                  <div v-if="msg.type === 1" class="message-text">{{ msg.content }}</div>
                  <div v-if="msg.type === 2" class="message-file">
                    <el-button size="small" @click="downloadFile(msg)">{{ msg.file_name }}</el-button>
                  </div>
                </div>
              </div>
            </div>
          </el-scrollbar>
          <div class="input-container">
            <el-input v-model="inputContent" placeholder="输入消息" @keyup.enter="sendMessage">
              <template #append>
                <el-upload
                  :action="uploadUrl"
                  :show-file-list="false"
                  :on-success="handleFileSuccess"
                >
                  <el-button>
                    <el-icon><Folder /></el-icon>
                  </el-button>
                </el-upload>
              </template>
            </el-input>
            <el-button type="primary" @click="sendMessage">发送</el-button>
          </div>
        </el-main>
      </el-container>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useStore } from 'vuex'
import { useRouter, useRoute } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElNotification } from 'element-plus'
import { ChatDotRound, User, HomeFilled, Setting, VideoCamera, Folder } from '@element-plus/icons-vue'

const store = useStore()
const router = useRouter()
const route = useRoute()

const userInfo = computed(() => store.state.userInfo)
const isAdmin = computed(() => userInfo.value.uuid && userInfo.value.uuid.substring(0, 4) === '1000')

const data = reactive({
  sessionId: route.params.id,
  contactInfo: {},
  messageList: [],
  inputContent: '',
  isGroup: false,

  // WebRTC状态
  ableToStartCall: true,
  ableToReceiveOrRejectCall: false,
  rtcPeerConn: null,
  localStream: null,
  remoteStream: null,
  localVideo: null,
  remoteVideo: null
})

const localVideoRef = ref(null)
const remoteVideoRef = ref(null)
const scrollbarRef = ref(null)

const uploadUrl = computed(() => store.state.backendUrl + '/message/uploadFile')

// ICE配置（局域网为空，公网需要STUN/TURN服务器）
const ICE_CFG = {}

onMounted(() => {
  parseSessionId()
  loadContactInfo()
  loadMessageList()

  // 监听WebSocket消息
  if (store.state.socket) {
    store.state.socket.onmessage = handleWebSocketMessage
  }

  // 监听自定义WebSocket事件
  window.addEventListener('ws-message', handleCustomMessage)
})

onBeforeUnmount(() => {
  closeLocalMediaStream()
  closeRtcPeerConnection()
  window.removeEventListener('ws-message', handleCustomMessage)
})

const parseSessionId = () => {
  const parts = data.sessionId.split('_')
  if (parts.length === 2) {
    const contactId = parts[1]
    if (contactId.startsWith('G')) {
      data.isGroup = true
    }
    data.contactInfo.contact_id = contactId
  }
}

const loadContactInfo = async () => {
  try {
    if (data.isGroup) {
      const response = await axios.get(store.state.backendUrl + '/group/getGroupInfo', {
        params: { uuid: data.contactInfo.contact_id }
      })
      if (response.data.code === 200) {
        data.contactInfo = response.data.data
        if (data.contactInfo.avatar && !data.contactInfo.avatar.startsWith('http')) {
          data.contactInfo.avatar = store.state.backendUrl + data.contactInfo.avatar
        }
        data.contactInfo.nickname = data.contactInfo.name
      }
    } else {
      const response = await axios.get(store.state.backendUrl + '/contact/getContactInfo', {
        params: {
          user_uuid: userInfo.value.uuid,
          contact_uuid: data.contactInfo.contact_id
        }
      })
      if (response.data.code === 200) {
        data.contactInfo = response.data.data
        if (data.contactInfo.avatar && !data.contactInfo.avatar.startsWith('http')) {
          data.contactInfo.avatar = store.state.backendUrl + data.contactInfo.avatar
        }
      }
    }
  } catch (error) {
    console.error('加载联系人信息失败:', error)
  }
}

const loadMessageList = async () => {
  try {
    let response
    if (data.isGroup) {
      response = await axios.get(store.state.backendUrl + '/message/getGroupMessageList', {
        params: {
          user_uuid: userInfo.value.uuid,
          group_uuid: data.contactInfo.contact_id
        }
      })
    } else {
      response = await axios.get(store.state.backendUrl + '/message/getMessageList', {
        params: {
          user_uuid: userInfo.value.uuid,
          contact_uuid: data.contactInfo.contact_id
        }
      })
    }
    if (response.data.code === 200) {
      data.messageList = response.data.data || []
      data.messageList.forEach(msg => {
        if (msg.send_avatar && !msg.send_avatar.startsWith('http')) {
          msg.send_avatar = store.state.backendUrl + msg.send_avatar
        }
      })
      scrollToBottom()
    }
  } catch (error) {
    console.error('加载消息列表失败:', error)
  }
}

const handleWebSocketMessage = (jsonMessage) => {
  const message = JSON.parse(jsonMessage.data)
  handleIncomingMessage(message)
}

const handleCustomMessage = (event) => {
  handleIncomingMessage(event.detail)
}

const handleIncomingMessage = (message) => {
  if (message.type !== 3) {
    // 普通消息
    if (
      (message.receive_id[0] === 'G' && message.receive_id === data.contactInfo.contact_id) ||
      (message.receive_id[0] === 'U' && message.receive_id === userInfo.value.uuid) ||
      message.send_id === userInfo.value.uuid
    ) {
      if (message.send_avatar && !message.send_avatar.startsWith('http')) {
        message.send_avatar = store.state.backendUrl + message.send_avatar
      }
      data.messageList.push(message)
      scrollToBottom()
    }
  } else {
    // 音视频消息
    const avData = JSON.parse(message.av_data)
    if (avData.messageId === 'PEER_LEAVE') {
      receiveEndCall()
    } else if (avData.messageId === 'PROXY') {
      if (avData.type === 'start_call') {
        ElNotification({
          title: '通话请求',
          message: `收到来自${message.send_name}的通话请求`,
          type: 'warning'
        })
        data.ableToReceiveOrRejectCall = true
        data.ableToStartCall = false
      } else if (avData.type === 'receive_call') {
        createOffer()
      } else if (avData.type === 'reject_call') {
        endCall()
      } else if (avData.type === 'sdp') {
        if (avData.messageData.sdp.type === 'offer') {
          handleOfferSdp(avData.messageData.sdp)
        } else if (avData.messageData.sdp.type === 'answer') {
          handleAnswerSdp(avData.messageData.sdp)
        }
      } else if (avData.type === 'candidate') {
        handleCandidate(avData.messageData.candidate)
      }
    }
  }
}

const sendMessage = async () => {
  if (!data.inputContent) return

  const message = {
    session_id: data.sessionId,
    type: 1,
    content: data.inputContent,
    url: '',
    send_id: userInfo.value.uuid,
    send_name: userInfo.value.nickname,
    send_avatar: userInfo.value.avatar.replace(store.state.backendUrl, ''),
    receive_id: data.contactInfo.contact_id,
    file_size: '',
    file_name: '',
    file_type: '',
    av_data: ''
  }

  if (store.state.socket && store.state.socket.readyState === WebSocket.OPEN) {
    store.state.socket.send(JSON.stringify(message))
  }

  data.messageList.push({
    ...message,
    send_avatar: userInfo.value.avatar
  })
  data.inputContent = ''
  scrollToBottom()
}

const handleFileSuccess = (response) => {
  if (response.code === 200) {
    const message = {
      session_id: data.sessionId,
      type: 2,
      content: '',
      url: response.data.url,
      send_id: userInfo.value.uuid,
      send_name: userInfo.value.nickname,
      send_avatar: userInfo.value.avatar.replace(store.state.backendUrl, ''),
      receive_id: data.contactInfo.contact_id,
      file_size: response.data.size,
      file_name: response.data.name,
      file_type: response.data.type,
      av_data: ''
    }

    if (store.state.socket && store.state.socket.readyState === WebSocket.OPEN) {
      store.state.socket.send(JSON.stringify(message))
    }

    data.messageList.push({
      ...message,
      send_avatar: userInfo.value.avatar
    })
    scrollToBottom()
  }
}

const downloadFile = (msg) => {
  const url = store.state.backendUrl + msg.url
  const a = document.createElement('a')
  a.href = url
  a.download = msg.file_name
  a.click()
}

const scrollToBottom = () => {
  nextTick(() => {
    if (scrollbarRef.value) {
      scrollbarRef.value.setScrollTop(scrollbarRef.value.wrap.scrollHeight)
    }
  })
}

// WebRTC相关函数
const startCall = async () => {
  if (!data.ableToStartCall) {
    ElMessage.warning('当前无法发起通话')
    return
  }

  try {
    data.localStream = await navigator.mediaDevices.getUserMedia({ audio: true, video: true })
    if (localVideoRef.value) {
      localVideoRef.value.srcObject = data.localStream
      data.localVideo = localVideoRef.value
      localVideoRef.value.style.display = 'block'
    }

    data.rtcPeerConn = new RTCPeerConnection(ICE_CFG)
    data.localStream.getTracks().forEach(track => {
      data.rtcPeerConn.addTrack(track, data.localStream)
    })

    data.rtcPeerConn.ontrack = (event) => {
      data.remoteStream = event.streams[0]
      if (remoteVideoRef.value) {
        remoteVideoRef.value.srcObject = data.remoteStream
        data.remoteVideo = remoteVideoRef.value
        remoteVideoRef.value.style.display = 'block'
      }
    }

    data.rtcPeerConn.onicecandidate = (event) => {
      if (event.candidate) {
        const proxyCandidateMessage = {
          messageId: 'PROXY',
          type: 'candidate',
          messageData: { candidate: event.candidate }
        }
        sendRtcMessage(proxyCandidateMessage)
      }
    }

    const proxyStartCallMessage = { messageId: 'PROXY', type: 'start_call' }
    sendRtcMessage(proxyStartCallMessage)

    data.ableToStartCall = false
    data.ableToReceiveOrRejectCall = false
  } catch (error) {
    ElMessage.error('无法获取媒体设备: ' + error.message)
  }
}

const receiveCall = () => {
  if (!data.ableToReceiveOrRejectCall) {
    ElMessage.warning('对方没有发起通话或已在通话中')
    return
  }

  try {
    navigator.mediaDevices.getUserMedia({ audio: true, video: true }).then(stream => {
      data.localStream = stream
      if (localVideoRef.value) {
        localVideoRef.value.srcObject = data.localStream
        data.localVideo = localVideoRef.value
        localVideoRef.value.style.display = 'block'
      }

      data.rtcPeerConn = new RTCPeerConnection(ICE_CFG)
      data.localStream.getTracks().forEach(track => {
        data.rtcPeerConn.addTrack(track, data.localStream)
      })

      data.rtcPeerConn.ontrack = (event) => {
        data.remoteStream = event.streams[0]
        if (remoteVideoRef.value) {
          remoteVideoRef.value.srcObject = data.remoteStream
          data.remoteVideo = remoteVideoRef.value
          remoteVideoRef.value.style.display = 'block'
        }
      }

      data.rtcPeerConn.onicecandidate = (event) => {
        if (event.candidate) {
          const proxyCandidateMessage = {
            messageId: 'PROXY',
            type: 'candidate',
            messageData: { candidate: event.candidate }
          }
          sendRtcMessage(proxyCandidateMessage)
        }
      }

      const proxyReceiveCallMessage = { messageId: 'PROXY', type: 'receive_call' }
      sendRtcMessage(proxyReceiveCallMessage)

      data.ableToReceiveOrRejectCall = false
    })
  } catch (error) {
    ElMessage.error('无法获取媒体设备: ' + error.message)
  }
}

const rejectCall = () => {
  if (!data.ableToReceiveOrRejectCall) {
    ElMessage.warning('对方没有发起通话或已在通话中，无法拒绝通话')
    return
  }

  const proxyRejectCallMessage = { messageId: 'PROXY', type: 'reject_call' }
  sendRtcMessage(proxyRejectCallMessage)
  data.ableToReceiveOrRejectCall = false
}

const sendEndCall = () => {
  if (!data.localVideo && !data.remoteVideo) {
    ElMessage.warning('尚未开始通话，无法挂断')
    return
  }

  closeLocalMediaStream()
  closeRtcPeerConnection()
  data.remoteStream = null
  data.localStream = null
  data.localVideo = null
  data.remoteVideo = null
  data.ableToReceiveOrRejectCall = false
  data.ableToStartCall = true

  const proxyPeerLeaveMessage = { messageId: 'PEER_LEAVE' }
  sendRtcMessage(proxyPeerLeaveMessage)
}

const receiveEndCall = () => {
  closeLocalMediaStream()
  closeRtcPeerConnection()
  data.remoteStream = null
  data.localStream = null
  data.localVideo = null
  data.remoteVideo = null
  data.ableToReceiveOrRejectCall = false
  data.ableToStartCall = true
  ElMessage.warning('对方已挂断')
}

const endCall = () => {
  closeLocalMediaStream()
  closeRtcPeerConnection()
  data.remoteStream = null
  data.localStream = null
  data.localVideo = null
  data.remoteVideo = null
  data.ableToReceiveOrRejectCall = false
  data.ableToStartCall = true
  ElMessage.warning('对方拒绝通话')
}

const createOffer = () => {
  const offerOpts = { offerToReceiveAudio: true, offerToReceiveVideo: true }
  data.rtcPeerConn.createOffer(offerOpts).then(desc => {
    data.rtcPeerConn.setLocalDescription(desc)
    const proxySdpMessage = {
      messageId: 'PROXY',
      type: 'sdp',
      messageData: { sdp: desc }
    }
    sendRtcMessage(proxySdpMessage)
  }).catch(err => {
    console.error('createOffer失败:', err)
  })
}

const createAnswer = () => {
  data.rtcPeerConn.createAnswer().then(desc => {
    data.rtcPeerConn.setLocalDescription(desc)
    const proxySdpMessage = {
      messageId: 'PROXY',
      type: 'sdp',
      messageData: { sdp: desc }
    }
    sendRtcMessage(proxySdpMessage)
  }).catch(err => {
    console.error('createAnswer失败:', err)
  })
}

const handleOfferSdp = (sdp) => {
  data.rtcPeerConn.setRemoteDescription(new RTCSessionDescription(sdp)).then(() => {
    createAnswer()
  }).catch(err => {
    console.error('setRemoteDescription失败:', err)
  })
}

const handleAnswerSdp = (sdp) => {
  data.rtcPeerConn.setRemoteDescription(new RTCSessionDescription(sdp)).catch(err => {
    console.error('setRemoteDescription失败:', err)
  })
}

const handleCandidate = (candidate) => {
  data.rtcPeerConn.addIceCandidate(new RTCIceCandidate(candidate))
}

const sendRtcMessage = (avDataObj) => {
  const rtcMessage = {
    session_id: data.sessionId,
    type: 3,
    content: '',
    url: '',
    send_id: userInfo.value.uuid,
    send_name: userInfo.value.nickname,
    send_avatar: userInfo.value.avatar.replace(store.state.backendUrl, ''),
    receive_id: data.contactInfo.contact_id,
    file_size: '',
    file_name: '',
    file_type: '',
    av_data: JSON.stringify(avDataObj)
  }

  if (store.state.socket && store.state.socket.readyState === WebSocket.OPEN) {
    store.state.socket.send(JSON.stringify(rtcMessage))
  }
}

const closeLocalMediaStream = () => {
  if (data.localStream) {
    data.localStream.getTracks().forEach(track => track.stop())
    data.localStream = null
  }
}

const closeRtcPeerConnection = () => {
  if (data.rtcPeerConn) {
    data.rtcPeerConn.close()
    data.rtcPeerConn = null
  }
}

// 导航函数
const goToSession = () => {
  router.push('/sessionlist')
}

const goToContact = () => {
  router.push('/contactlist')
}

const goToOwnInfo = () => {
  router.push('/owninfo')
}

const goToManager = () => {
  router.push('/manager')
}

const logout = () => {
  store.commit('cleanUserInfo')
  router.push('/login')
}
</script>

<style scoped>
.chat-wrap {
  width: 100%;
  height: 100vh;
  background-color: #f5f5f5;
}

.chat-window {
  width: 100%;
  height: 100%;
}

.chat-window-container {
  height: 100%;
}

.aside-container {
  width: 200px;
  background-color: #2e2e2e;
}

.nav-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 15px 10px;
  background-color: #1e1e1e;
}

.nav-item {
  width: 40px;
  height: 40px;
  display: flex;
  justify-content: center;
  align-items: center;
  cursor: pointer;
  color: #fff;
  border-radius: 4px;
  margin-bottom: 10px;
}

.nav-item:hover {
  background-color: #3e3e3e;
}

.contact-info-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px;
}

.contact-avatar {
  width: 80px;
  height: 80px;
  border-radius: 4px;
  margin-bottom: 10px;
}

.contact-detail {
  text-align: center;
}

.contact-name {
  font-size: 14px;
  color: #fff;
  display: block;
}

.contact-id {
  font-size: 12px;
  color: #aaa;
  display: block;
}

.main-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px;
  background-color: #fff;
  border-bottom: 1px solid #eee;
}

.chat-actions {
  display: flex;
  gap: 10px;
}

.video-container {
  display: flex;
  gap: 10px;
  padding: 10px;
  background-color: #000;
}

.video-container video {
  width: 300px;
  height: 200px;
  background-color: #333;
}

#localVideo {
  display: none;
}

#remoteVideo {
  display: none;
}

.message-list-container {
  flex: 1;
  padding: 10px;
  background-color: #f5f5f5;
}

.message-item {
  margin-bottom: 15px;
}

.message-self {
  display: flex;
  justify-content: flex-end;
}

.message-other {
  display: flex;
  justify-content: flex-start;
}

.message-avatar {
  width: 40px;
  height: 40px;
  border-radius: 4px;
  margin: 0 10px;
}

.message-content {
  max-width: 300px;
}

.message-name {
  font-size: 12px;
  color: #666;
  display: block;
  margin-bottom: 5px;
}

.message-text {
  padding: 10px;
  background-color: #fff;
  border-radius: 4px;
  word-wrap: break-word;
}

.message-self .message-text {
  background-color: #95ec69;
}

.input-container {
  display: flex;
  padding: 15px;
  background-color: #fff;
  border-top: 1px solid #eee;
}

.input-container .el-input {
  flex: 1;
  margin-right: 10px;
}
</style>