import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/echarts') || id.includes('echarts-for-react')) {
            return 'echarts'
          }
          if (id.includes('node_modules/cytoscape') || id.includes('cytoscape-fcose')) {
            return 'cytoscape'
          }
          if (id.includes('node_modules/@tanstack')) {
            return 'query'
          }
          if (id.includes('node_modules/react-dom') || id.includes('node_modules/react/')) {
            return 'react-vendor'
          }
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        ws: true,
      },
      '/health': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/ready': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
})
