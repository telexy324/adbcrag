import { http, type APIResponse } from './http'

export async function askQuestion(data: {
  question: string
  systemName?: string
  componentName?: string
  docType?: string
  topK?: number
}) {
  const response = await http.post<APIResponse<{ answer: string; citations: any[] }>>('/qa/ask', data)
  return response.data.data
}
