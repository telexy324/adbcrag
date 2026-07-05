import { http, type APIResponse } from './http'

export type Citation = {
  documentId: number
  documentTitle: string
  chunkId: number
  sourceSection: string
  content: string
  score: number
}

export async function askQuestion(data: {
  question: string
  systemName?: string
  componentName?: string
  docType?: string
  topK?: number
}) {
  const response = await http.post<APIResponse<{ answer: string; citations: Citation[] }>>('/qa/ask', data)
  return response.data.data
}
