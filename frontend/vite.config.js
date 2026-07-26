import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api/auth': { target: 'http://localhost:8082', rewrite: (path) => path.replace(/^\/api\/auth/, '') },
      '/api/matchmaker': { target: 'http://localhost:8083', rewrite: (path) => path.replace(/^\/api\/matchmaker/, '') },
      '/ws': { target: 'ws://localhost:8081', ws: true },
    },
  },
})
