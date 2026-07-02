import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTimeoutMs = Number(env.VITE_DEV_PROXY_TIMEOUT_MS || env.VITE_UPLOAD_TIMEOUT_MS || 600000)

  return {
    plugins: [react()],
    server: {
      port: 5173,
      proxy: {
        '/api': {
          target: 'http://127.0.0.1:8080',
          timeout: proxyTimeoutMs,
          proxyTimeout: proxyTimeoutMs,
        },
      },
    },
  }
})
