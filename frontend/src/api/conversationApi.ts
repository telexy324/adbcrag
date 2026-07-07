import { http, type APIResponse } from './http'

export type Conversation = {
  id: number
  userId: number
  title: string
  conversationType: string
  status: string
  lastMessageAt: string
  createdAt: string
}

export type ConversationMessage = {
  id: number
  conversationId: number
  userId: number
  role: 'user' | 'assistant' | 'system' | 'tool'
  content: string
  messageType: string
  createdAt: string
}

export async function listConversations() {
  const { data } = await http.get<APIResponse<Conversation[]>>('/conversations')
  return data.data
}

export async function createConversation(input: { title?: string; conversationType?: string }) {
  const { data } = await http.post<APIResponse<Conversation>>('/conversations', input)
  return data.data
}

export async function getConversationMessages(id: number) {
  const { data } = await http.get<APIResponse<{ conversation: Conversation; messages: ConversationMessage[]; summary: string }>>(`/conversations/${id}/messages`)
  return data.data
}

export async function archiveConversation(id: number) {
  const { data } = await http.delete<APIResponse<{ archived: boolean }>>(`/conversations/${id}`)
  return data.data
}
