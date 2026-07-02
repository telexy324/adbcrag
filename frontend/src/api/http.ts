import axios from 'axios'

export type APIResponse<T> = {
  code: number
  message: string
  data: T
}

const apiTimeoutMs = Number(import.meta.env.VITE_API_TIMEOUT_MS || 120000)

export const http = axios.create({
  baseURL: '/api',
  timeout: apiTimeoutMs,
})

http.interceptors.response.use((response) => {
  const payload = response.data as APIResponse<unknown>
  if (payload.code !== 0) {
    throw new Error(payload.message || '请求失败')
  }
  return response
})
