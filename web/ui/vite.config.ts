import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The built UI is embedded by the Go launcher (web/server/dist) via go:embed.
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../server/dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      // Dev: forward API + SSE to the Go launcher.
      '/api': {
        target: 'http://127.0.0.1:18800',
        changeOrigin: true,
      },
    },
  },
})
