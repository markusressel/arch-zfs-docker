import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [vue(), vuetify({ autoImport: true })],
  test: {
    include: ['src/**/*.test.ts'],
    environment: 'happy-dom',
    setupFiles: ['src/test/setup.ts'],
    // Vuetify ships CSS imports in its components; let Vite handle them
    server: { deps: { inline: ['vuetify'] } },
  },
})
