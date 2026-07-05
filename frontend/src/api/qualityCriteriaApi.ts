import { http, type APIResponse } from './http'

export type QualityCriteria = {
  id: number
  name: string
  content: string
  isDefault: boolean
  createdBy: string
  createdAt: string
  updatedAt: string
}

export type SaveQualityCriteriaInput = {
  name: string
  content: string
  isDefault?: boolean
}

export async function listQualityCriteria() {
  const { data } = await http.get<APIResponse<QualityCriteria[]>>('/quality-criteria')
  return data.data
}

export async function createQualityCriteria(input: SaveQualityCriteriaInput) {
  const { data } = await http.post<APIResponse<QualityCriteria>>('/quality-criteria', input)
  return data.data
}

export async function updateQualityCriteria(id: number, input: SaveQualityCriteriaInput) {
  const { data } = await http.put<APIResponse<QualityCriteria>>(`/quality-criteria/${id}`, input)
  return data.data
}

export async function deleteQualityCriteria(id: number) {
  const { data } = await http.delete<APIResponse<{ id: number }>>(`/quality-criteria/${id}`)
  return data.data
}

export async function setDefaultQualityCriteria(id: number) {
  const { data } = await http.post<APIResponse<QualityCriteria>>(`/quality-criteria/${id}/default`)
  return data.data
}
