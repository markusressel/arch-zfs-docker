<script setup lang="ts">
import { mdiFolderOutline, mdiPackageVariantClosed, mdiThemeLightDark, mdiWeatherNight, mdiWeatherSunny } from '@mdi/js'
import { computed } from 'vue'
import { useSettings } from '../../composables/useSettings'
import { useThemeChoice } from '../../composables/useThemeChoice'
import AppButton from '../ui/AppButton.vue'

const { settings } = useSettings()
const { choice, choices } = useThemeChoice()

// The demo build (GitHub Pages) has no server: no file index, and a chip that says what it is. The screenshots
// are taken from the demo with ?screenshot, which shows the header of a real instance.
const demo = import.meta.env.MODE === 'demo' && !new URLSearchParams(window.location.search).has('screenshot')

// Plain file index of the pacman repository, served by the Go server outside of the web app.
const filesUrl = computed(() => `/${settings.value?.repoName || 'zfslocal'}/x86_64/`)

const THEME_ICONS = {
  system: mdiThemeLightDark,
  light: mdiWeatherSunny,
  dark: mdiWeatherNight,
}
const THEME_LABELS = {
  system: 'System theme',
  light: 'Light theme',
  dark: 'Dark theme',
}
function nextTheme() {
  choice.value = choices[(choices.indexOf(choice.value) + 1) % choices.length]!
}
</script>

<template>
  <v-app-bar flat border="b" color="surface" density="comfortable">
    <div class="app-bar-content d-flex align-center ga-3 px-4">
      <RouterLink :to="{ name: 'dashboard' }" class="brand d-flex align-center ga-3" title="Back to the dashboard">
        <v-icon :icon="mdiPackageVariantClosed" color="primary" size="28" />
        <h1 class="text-title-large font-weight-bold">Arch Linux ZFS Repository</h1>
      </RouterLink>
      <v-spacer />
      <v-chip v-if="demo" color="warning" href="https://github.com/markusressel/arch-zfs-docker" target="_blank" rel="noopener">
        Demo with made-up data
      </v-chip>
      <v-chip color="success" class="d-none d-sm-flex">{{ settings?.repoName || 'zfslocal' }} • x86_64</v-chip>
      <AppButton v-if="!demo" size="small" :href="filesUrl"><v-icon :icon="mdiFolderOutline" start />Browse files</AppButton>
      <v-btn
        :icon="THEME_ICONS[choice]"
        :title="`${THEME_LABELS[choice]} (click to switch)`"
        :aria-label="`${THEME_LABELS[choice]} (click to switch)`"
        variant="text"
        size="small"
        @click="nextTheme"
      />
    </div>
  </v-app-bar>
</template>

<style scoped>
.app-bar-content {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}

.brand {
  color: inherit;
  text-decoration: none;
}
</style>
