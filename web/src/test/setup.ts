import { config } from '@vue/test-utils'
import vuetify from '../plugins/vuetify'

// Every mounted component gets the app's Vuetify instance, with its defaults and themes.
config.global.plugins.push(vuetify)

// Vuetify measures elements with ResizeObserver, which happy-dom does not have.
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as unknown as typeof ResizeObserver

// Vuetify's overlays (menus, dialogs) position themselves with visualViewport, which happy-dom does not have.
globalThis.visualViewport ??= Object.assign(new EventTarget(), {
  width: 1024,
  height: 768,
  offsetLeft: 0,
  offsetTop: 0,
  pageLeft: 0,
  pageTop: 0,
  scale: 1,
  onresize: null,
  onscroll: null,
  onscrollend: null,
}) as unknown as VisualViewport
