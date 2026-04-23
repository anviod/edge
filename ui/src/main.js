import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import ArcoVue from '@arco-design/web-vue'
import '@arco-design/web-vue/dist/arco.css'
import './styles/globals.css'

const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  messages: {
    zh: {}
  }
})

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(ArcoVue)
app.use(i18n)

app.mount('#app')
