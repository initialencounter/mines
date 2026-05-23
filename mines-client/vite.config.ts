import { fileURLToPath, URL } from 'node:url'

import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueDevTools(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
      '/login': 'http://localhost:8080',
      '/register': 'http://localhost:8080',
      '/verify': 'http://localhost:8080',
      '/reset': 'http://localhost:8080',
      '/getMinefield': 'http://localhost:8080',
      '/getRank': 'http://localhost:8080',
      '/newGame': 'http://localhost:8080',
    },
  },
  build: {
    rollupOptions: {
      onwarn(warning, warn) {
        if (warning.code === 'INVALID_ANNOTATION')
          return
        warn(warning)
      },
    },
  },
})
