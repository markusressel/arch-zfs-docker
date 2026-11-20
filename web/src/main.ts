import '@fontsource-variable/roboto'
import './styles/reset.css'
// Before App: vuetify/styles declares the order of Vuetify's CSS layers. Component styles imported first would
// declare their layer first and lose against the reset layer (buttons fell back to the inherited font size).
import vuetify from './plugins/vuetify'
import { createApp } from 'vue'
import App from './App.vue'
import { createAppRouter } from './router'
import './styles/variables.css'
import './styles/base.css'

createApp(App).use(vuetify).use(createAppRouter()).mount('#app')
