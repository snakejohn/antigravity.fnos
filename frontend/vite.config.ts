import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const chunkGroups: Array<[string, string[]]> = [
  ['vendor-vue', ['/node_modules/vue/', '/node_modules/@vue/', '/node_modules/pinia/']],
  ['vendor-naive', ['/node_modules/naive-ui/', '/node_modules/@vicons/']],
  ['vendor-highlight', ['/node_modules/highlight.js/']]
]

function manualChunks(id: string): string | undefined {
  return chunkGroups.find(([, patterns]) => patterns.some((pattern) => id.includes(pattern)))?.[0]
}

export default defineConfig({
  plugins: [vue()],
  base: './',
  build: {
    rollupOptions: {
      output: {
        manualChunks
      }
    }
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:2399',
        changeOrigin: true,
        ws: true
      }
    }
  }
})
