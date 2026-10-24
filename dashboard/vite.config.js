import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Dev proxy sends /api and /health to the backend so `npm run dev` works
// against a port-forwarded backend. In production Nginx proxies instead.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
    },
  },
  build: { outDir: 'dist' },
})
