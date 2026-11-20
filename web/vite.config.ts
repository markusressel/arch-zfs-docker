import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify'
import { writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const outDir = resolve(import.meta.dirname, '../internal/ui/dist')

// The Go binary embeds `internal/ui/dist`, which must always exist for `go build`.
// Vite empties the directory on every build, so restore the tracked placeholder afterwards.
function keepPlaceholder(): Plugin {
  return {
    name: 'keep-dist-placeholder',
    apply: 'build',
    writeBundle() {
      writeFileSync(resolve(outDir, '.gitkeep'), '')
    },
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), vuetify({ autoImport: true }), keepPlaceholder()],
  base: '/ui/',
  build: {
    outDir,
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    // `go run ./cmd/server -dev -repo-dir /tmp/repo -listen :8080`, or another server with DEV_BACKEND_URL
    // (e.g. a deployed one, to look at the UI with real data)
    proxy: {
      '/api': { target: process.env.DEV_BACKEND_URL || 'http://localhost:8080', changeOrigin: true },
    },
  },
})
