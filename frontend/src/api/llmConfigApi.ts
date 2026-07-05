import { http, type APIResponse } from './http'

export type LLMProvider = 'deepseek' | 'qwen3' | 'openai_compatible'

export type LLMConfig = {
  id: number
  name: string
  provider: LLMProvider
  baseUrl: string
  model: string
  temperature: number
  isDefault: boolean
  enabled: boolean
  updatedAt: string
}

export type SaveLLMConfigInput = {
  name: string
  provider: LLMProvider
  baseUrl: string
  model: string
  apiKey?: string
  temperature?: number
  isDefault?: boolean
  enabled?: boolean
}

export async function listLLMConfigs() {
  const { data } = await http.get<APIResponse<LLMConfig[]>>('/llm-configs')
  return data.data
}

export async function createLLMConfig(input: SaveLLMConfigInput) {
  const { data } = await http.post<APIResponse<LLMConfig>>('/llm-configs', input)
  return data.data
}

export async function updateLLMConfig(id: number, input: SaveLLMConfigInput) {
  const { data } = await http.put<APIResponse<LLMConfig>>(`/llm-configs/${id}`, input)
  return data.data
}

export async function deleteLLMConfig(id: number) {
  const { data } = await http.delete<APIResponse<{ id: number }>>(`/llm-configs/${id}`)
  return data.data
}

export async function setDefaultLLMConfig(id: number) {
  const { data } = await http.post<APIResponse<LLMConfig>>(`/llm-configs/${id}/default`)
  return data.data
}

export async function testLLMConfig(id: number, prompt?: string) {
  const { data } = await http.post<APIResponse<{ ok: boolean; message: string; content?: string }>>(`/llm-configs/${id}/test`, { prompt })
  return data.data
}
