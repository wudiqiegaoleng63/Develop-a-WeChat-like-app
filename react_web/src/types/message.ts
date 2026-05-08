// Backend message types
export enum MessageType {
  TEXT = 0,
  VOICE = 1,
  FILE = 2,
  AV = 3,
}

export interface ChatMessage {
  uuid: string
  session_id: string
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
  status: number
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
}
