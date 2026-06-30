import axios from 'axios'

export type APIResponse<T> = {
  code: number
  message: string
  data: T
}

export const http = axios.create({
  baseURL: '/api',
  timeout: 120000,
})

http.interceptors.response.use((response) => {
  const payload = response.data as APIResponse<unknown>
  if (payload.code !== 0) {
    throw new Error(payload.message || '请求失败')
  }
  return response
})
