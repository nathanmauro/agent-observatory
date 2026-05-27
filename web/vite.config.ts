import { defineConfig } from 'vite'
import solid from 'vite-plugin-solid'

export default defineConfig({
  plugins: [solid()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:3284',
      '/health': 'http://127.0.0.1:3284',
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
