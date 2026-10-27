import { resolve } from 'path'
import { defineConfig } from 'electron-vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { compression } from 'vite-plugin-compression2'

export default defineConfig({
  main: {},
  preload: {},
  renderer: {
    resolve: {
      alias: {
        '@': resolve('src/renderer/src'),
        '@renderer': resolve('src/renderer/src'),
        '@shared': resolve('src/shared')
      }
    },
    plugins: [
      vue(),
      tailwindcss(),
      compression({
        algorithms: ['gzip'],
        threshold: 10240
      })
    ],
    build: {
      rollupOptions: {
        output: {
          codeSplitting: {
            groups: [
              {
                name: 'vue',
                test: /node_modules[\\/](vue|vue-router|pinia|@vue)[\\/]/
              }
            ]
          }
        }
      }
    }
  }
})
