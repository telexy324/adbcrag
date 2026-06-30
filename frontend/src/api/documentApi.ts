import { http, type APIResponse } from './http'

export type DocumentItem = {
  id: number
  title: string
  systemName: string
  componentName: string
  docType: string
  status: string
  qualityScore: number
  updatedAt: string
}

export async function uploadDocument(formData: FormData) {
  const { data } = await http.post<APIResponse<any>>('/documents/upload', formData)
  return data.data
}

export async function listDocuments(params: {
  page: number
  pageSize: number
  status?: string
  systemName?: string
  componentName?: string
  docType?: string
}) {
  const { data } = await http.get<APIResponse<{ items: DocumentItem[]; total: number }>>('/documents', { params })
  return data.data
}

export async function getDocument(id: number) {
  const { data } = await http.get<APIResponse<any>>(`/documents/${id}`)
  return data.data
}

export async function reviewDocument(id: number, data: {
  action: 'approve' | 'reject' | 'archive' | 'deprecate'
  comment?: string
}) {
  const response = await http.post<APIResponse<any>>(`/documents/${id}/review`, data)
  return response.data.data
}

export async function getDashboardStats() {
  const { data } = await http.get<APIResponse<any>>('/dashboard/stats')
  return data.data
}
