// Backend message types
export enum MessageType {
  TEXT = 0,
  VOICE = 1,
  FILE = 2,
  AV = 3,
}

// Matches GetMessageListRespond / GetGroupMessageListRespond
// Backend does NOT return uuid or session_id in message list
export interface ChatMessage {
  send_id: string
  send_name: string
  send_avatar: string
  receive_id: string
  type: MessageType
  content: string
  url: string
  file_size: string
  file_name: string
  file_type: string
  created_at: string
  // Client-only fields for optimistic updates
  uuid?: string
  session_id?: string
  status?: number
  // AV signaling data
  av_data?: string
}

export interface ChatMessageRequest {
  session_id: string
  type: number
  content: string
  url: string
  send_id: string
  send_name: string
  send_avatar: string
  receive_id: string
  file_size: string
  file_name: string
  file_type: string
  av_data?: string
}
