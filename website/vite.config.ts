import { writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

function versionJson(): Plugin {
  return {
    name: 'version-json',
    apply: 'build',
    closeBundle() {
      const version = (process.env.VITE_APP_VERSION || '0.0.0').trim()
      writeFileSync(resolve('dist/version.json'), JSON.stringify({ version }) + '\n')
    },
  }
}

export default defineConfig({
  plugins: [vue(), tailwindcss(), versionJson()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
