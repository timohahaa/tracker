import { fileURLToPath, URL } from 'node:url'

import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// Dev-proxy targets the Go backend (make run -> :8080). The browser talks to the
// Vite dev origin, so the httpOnly session cookie stays same-origin and is
// forwarded transparently through the proxy.
const BACKEND_TARGET = 'http://localhost:8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      '/api': { target: BACKEND_TARGET, changeOrigin: true },
      '/mcp': { target: BACKEND_TARGET, changeOrigin: true },
      '/docs': { target: BACKEND_TARGET, changeOrigin: true },
    },
  },
})
