import { fileURLToPath, URL } from 'node:url'
import path from 'node:path'
import fs from 'node:fs'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import tailwindcss from '@tailwindcss/vite'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import viteCompression from 'vite-plugin-compression'

const webRoot = __dirname
const clientSrc = path.resolve(webRoot, 'client')
const webNodeModules = path.resolve(webRoot, 'node_modules')
const webDep = (name: string) => path.resolve(webNodeModules, name)

// 钉到 web/node_modules，避免 @client 与 src 拉起重复 Vue 实例。
const sharedDeps = [
  'vue',
  'pinia',
  'vue-router',
  '@vueuse/core',
  'marked',
  'marked-highlight',
  'dompurify',
  'highlight.js',
  'codemirror',
  '@codemirror/view',
  '@codemirror/state',
  '@codemirror/language',
  '@codemirror/lang-markdown',
  '@codemirror/language-data',
  '@codemirror/commands',
  '@codemirror/theme-one-dark',
  '@tiptap/core',
  '@tiptap/pm',
  '@tiptap/starter-kit',
  '@tiptap/vue-3',
]
const clientDepAliases = Object.fromEntries(
  sharedDeps.filter((dep) => fs.existsSync(webDep(dep))).map((dep) => [dep, webDep(dep)]),
)

function blockWails() {
  const blocked = /@wailsio\/runtime|bindings\/chengzi-agent-/
  return {
    name: 'block-wails-in-web',
    enforce: 'pre' as const,
    resolveId(source: string) {
      if (blocked.test(source)) {
        throw new Error(`网页不能引用 Wails 绑定: ${source}`)
      }
      return null
    },
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    blockWails(),
    vue(),
    vueDevTools(),
    tailwindcss(),
    AutoImport({
      resolvers: [ElementPlusResolver()],
    }),
    Components({
      resolvers: [ElementPlusResolver()],
    }),
    viteCompression({
      algorithm: 'gzip',
      threshold: 10240,
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@client': clientSrc,
      ...clientDepAliases,
    },
    dedupe: ['vue', 'pinia', 'vue-router', '@vueuse/core'],
  },
  server: {
    port: 5173,
    fs: {
      allow: [webRoot],
    },
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8183',
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'element-plus': ['element-plus', '@element-plus/icons-vue'],
          vendor: ['vue', 'vue-router', 'pinia'],
        },
      },
    },
  },
})
