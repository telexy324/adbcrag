import { http, type APIResponse } from './http'
import type { Citation } from './qaApi'

export type LogSource = {
  id: number
  name: string
  sourceType: 'elasticsearch' | 'server_file'
  systemName: string
  componentName: string
  environment: string
  endpoint: string
  username: string
  esIndexPattern: string
  esTimeField: string
  serverHost: string
  serverPort: number
  authType: 'password' | 'private_key'
  logPath: string
  pathAllowlist: string[] | null
  enabled: boolean
  updatedAt: string
}

export type LogSourceInput = Partial<LogSource> & {
  name: string
  sourceType: 'elasticsearch' | 'server_file'
  password?: string
  privateKey?: string
  privateKeyPassphrase?: string
  pathAllowlist?: string[]
}

export type LogItem = {
  timestamp?: string
  level: string
  message: string
  source: string
  raw: string
}

export type LogQuery = {
  sourceId: number
  timeStart?: string
  timeEnd?: string
  keyword?: string
  logLevel?: string
  logPath?: string
  limit?: number
}

export type LogAnalysisInput = LogQuery & {
  question: string
  systemName?: string
  componentName?: string
  topK?: number
}

export async function listLogSources() {
  const { data } = await http.get<APIResponse<LogSource[]>>('/log-sources')
  return data.data
}

export async function createLogSource(input: LogSourceInput) {
  const { data } = await http.post<APIResponse<LogSource>>('/log-sources', input)
  return data.data
}

export async function updateLogSource(id: number, input: LogSourceInput) {
  const { data } = await http.put<APIResponse<LogSource>>(`/log-sources/${id}`, input)
  return data.data
}

export async function deleteLogSource(id: number) {
  const { data } = await http.delete<APIResponse<{ id: number }>>(`/log-sources/${id}`)
  return data.data
}

export async function testLogSource(id: number) {
  const { data } = await http.post<APIResponse<{ ok: boolean; message: string }>>(`/log-sources/${id}/test`)
  return data.data
}

export async function previewLogs(input: LogQuery) {
  const { data } = await http.post<APIResponse<{ items: LogItem[]; total: number }>>('/logs/preview', input)
  return data.data
}

export async function analyzeLogs(input: LogAnalysisInput) {
  const { data } = await http.post<APIResponse<{
    taskId: number
    status: string
    summary: string
    possibleCauses: string[]
    evidence: string[]
    suggestions: string[]
    riskTips: string[]
    citations: Citation[]
  }>>('/log-analysis', input)
  return data.data
}
