import { defineConfig } from 'vite'
import solid from 'vite-plugin-solid'

export default defineConfig({
  plugins: [solid()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:3284',
        ws: true,
      },
      '/health': 'http://127.0.0.1:3284',
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
