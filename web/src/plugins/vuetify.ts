import 'vuetify/styles'

import { createVuetify } from 'vuetify'
import type { ThemeDefinition } from 'vuetify'
import { aliases, mdi } from 'vuetify/iconsets/mdi-svg'

/**
 * GitHub's dark palette, the look of the UI before Vuetify. `border` and `terminal` are extra colors for the
 * styles of our own components (`styles/variables.css` maps the old CSS variables onto the theme).
 */
export const DARK: ThemeDefinition = {
  dark: true,
  colors: {
    background: '#0d1117',
    surface: '#161b22',
    'surface-light': '#21262d',
    'surface-variant': '#30363d',
    'on-surface': '#f0f6fc',
    'on-background': '#f0f6fc',
    'on-surface-variant': '#8b949e',
    primary: '#58a6ff',
    'primary-darken-1': '#79c0ff',
    secondary: '#8b949e',
    success: '#3fb950',
    warning: '#d29922',
    error: '#f85149',
    info: '#58a6ff',
    border: '#30363d',
    terminal: '#030712',
  },
}

/** GitHub's light palette. The log terminal stays dark in both themes. */
export const LIGHT: ThemeDefinition = {
  dark: false,
  colors: {
    background: '#ffffff',
    surface: '#f6f8fa',
    'surface-light': '#eaeef2',
    'surface-variant': '#d1d9e0',
    'on-surface': '#1f2328',
    'on-background': '#1f2328',
    'on-surface-variant': '#59636e',
    primary: '#0969da',
    'primary-darken-1': '#0550ae',
    secondary: '#59636e',
    success: '#1a7f37',
    warning: '#9a6700',
    error: '#cf222e',
    info: '#0969da',
    border: '#d1d9e0',
    terminal: '#0d1117',
  },
}

export type ThemeChoice = 'system' | 'light' | 'dark'

export default createVuetify({
  icons: { defaultSet: 'mdi', aliases, sets: { mdi } },
  // Call sites only state what differs from these.
  defaults: {
    VAlert: { variant: 'tonal', density: 'compact' },
    VBtn: { variant: 'tonal', rounded: 'md' },
    VBtnToggle: { density: 'compact', variant: 'outlined', divided: true },
    VCard: { variant: 'flat', border: true, rounded: 'lg' },
    VChip: { variant: 'tonal', size: 'small' },
    VCheckbox: { density: 'compact', hideDetails: 'auto', color: 'primary' },
    VCombobox: { variant: 'outlined', density: 'compact', hideDetails: 'auto' },
    VDialog: { VCard: { variant: 'elevated', border: true } },
    VProgressLinear: { color: 'primary', rounded: true },
    VSelect: { variant: 'outlined', density: 'compact', hideDetails: 'auto' },
    VTable: { density: 'compact' },
    VTextField: { variant: 'outlined', density: 'compact', hideDetails: 'auto' },
    VTooltip: { location: 'top' },
  },
  theme: {
    defaultTheme: 'system',
    themes: { light: LIGHT, dark: DARK },
  },
})
