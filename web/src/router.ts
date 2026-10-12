import { createRouter, createWebHashHistory } from 'vue-router'
import AnalyticsView from './views/AnalyticsView.vue'
import BuildsView from './views/BuildsView.vue'
import DashboardView from './views/DashboardView.vue'
import InstallView from './views/InstallView.vue'
import SettingsView from './views/SettingsView.vue'

export const routes = [
  { path: '/', name: 'dashboard', component: DashboardView, meta: { label: 'Dashboard' } },
  { path: '/install', name: 'install', component: InstallView, meta: { label: 'Install' } },
  { path: '/builds', name: 'builds', component: BuildsView, meta: { label: 'Builds' } },
  { path: '/analytics', name: 'analytics', component: AnalyticsView, meta: { label: 'Analytics' } },
  { path: '/settings', name: 'settings', component: SettingsView, meta: { label: 'Settings' } },
]

// Hash history: the Go server serves the app under /ui/ and needs no SPA fallback.
export const createAppRouter = () => createRouter({ history: createWebHashHistory(import.meta.env.BASE_URL), routes })
