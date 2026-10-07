import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import { initI18n } from './i18n'
import './style.css';

initI18n();
createApp(App).use(createPinia()).use(router).mount('#app')
