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

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('accessToken')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use((response) => {
  const payload = response.data as APIResponse<unknown>
  if (payload.code !== 0) {
    throw new Error(payload.message || '请求失败')
  }
  return response
}, (error) => {
  if (error?.response?.status === 401) {
    localStorage.removeItem('accessToken')
    localStorage.removeItem('currentUser')
  }
  return Promise.reject(error)
})
