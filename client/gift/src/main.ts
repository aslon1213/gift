import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import { router } from './router'
import { auth } from './stores/auth'
import { userStore } from './stores/user'

// If we booted with a cached token, hydrate the profile in the background so views
// can read the preferred currency without waiting on /auth/me.
if (auth.isAuthenticated.value) {
  void userStore.load()
}

createApp(App).use(router).mount('#app')
