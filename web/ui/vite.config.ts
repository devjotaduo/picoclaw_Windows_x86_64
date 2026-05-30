import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The built UI is embedded by the Go launcher (web/server/dist) via go:embed.
// On Vercel (env VERCEL=1) we emit to the local ./dist instead, so it can be
// served as a static site that talks to a remote backend via VITE_API_BASE.
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: process.env.VERCEL ? 'dist' : '../server/dist',
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
