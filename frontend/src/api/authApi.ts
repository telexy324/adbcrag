import { http, type APIResponse } from './http'

export type CurrentUser = {
  id: number
  username: string
  displayName: string
  role: 'admin' | 'user'
  enabled: boolean
}

export type LoginResult = {
  accessToken: string
  user: CurrentUser
}

export type AppUser = CurrentUser & {
  lastLoginAt?: string
  passwordUpdatedAt?: string
  createdAt: string
}

export async function login(username: string, password: string) {
  const { data } = await http.post<APIResponse<LoginResult>>('/auth/login', { username, password })
  return data.data
}

export async function me() {
  const { data } = await http.get<APIResponse<CurrentUser>>('/auth/me')
  return data.data
}

export async function changePassword(input: { oldPassword: string; newPassword: string }) {
  const { data } = await http.post<APIResponse<{ changed: boolean }>>('/auth/change-password', input)
  return data.data
}

export async function listUsers() {
  const { data } = await http.get<APIResponse<AppUser[]>>('/users')
  return data.data
}

export async function createUser(input: { username: string; displayName?: string; password: string; role: 'admin' | 'user'; enabled?: boolean }) {
  const { data } = await http.post<APIResponse<AppUser>>('/users', input)
  return data.data
}

export async function updateUser(id: number, input: { displayName?: string; role?: 'admin' | 'user'; enabled?: boolean }) {
  const { data } = await http.put<APIResponse<AppUser>>(`/users/${id}`, input)
  return data.data
}

export async function resetUserPassword(id: number, password: string) {
  const { data } = await http.post<APIResponse<{ reset: boolean }>>(`/users/${id}/reset-password`, { password })
  return data.data
}
