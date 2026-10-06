import { createApp } from 'vue'
import App from './App.vue'
import { createAppRouter } from './router'
import './styles/variables.css'
import './styles/base.css'

createApp(App).use(createAppRouter()).mount('#app')
