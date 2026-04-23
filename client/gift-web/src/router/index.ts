import { createRouter, createWebHistory } from 'vue-router'
import { auth } from '../stores/auth'
import { server } from '../stores/server'

const LoginView = () => import('../views/LoginView.vue')
const RegisterView = () => import('../views/RegisterView.vue')
const DashboardView = () => import('../views/DashboardView.vue')
const GroupsView = () => import('../views/GroupsView.vue')
const GroupDetailView = () => import('../views/GroupDetailView.vue')
const SpendingsView = () => import('../views/SpendingsView.vue')
const IncomesView = () => import('../views/IncomesView.vue')
const ServerSetupView = () => import('../views/ServerSetupView.vue')

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/dashboard' },
    { path: '/setup', component: ServerSetupView, meta: { setup: true } },
    { path: '/login', component: LoginView, meta: { guest: true } },
    { path: '/register', component: RegisterView, meta: { guest: true } },
    { path: '/dashboard', component: DashboardView, meta: { requiresAuth: true } },
    { path: '/groups', component: GroupsView, meta: { requiresAuth: true } },
    { path: '/groups/:id', component: GroupDetailView, meta: { requiresAuth: true }, props: true },
    { path: '/spendings', component: SpendingsView, meta: { requiresAuth: true } },
    { path: '/incomes', component: IncomesView, meta: { requiresAuth: true } },
  ],
})

router.beforeEach((to) => {
  const configured = server.isConfigured.value
  const authed = auth.isAuthenticated.value

  if (!configured && !to.meta.setup) {
    return { path: '/setup', query: { redirect: to.fullPath } }
  }
  if (configured && to.meta.setup && authed) {
    return { path: '/dashboard' }
  }
  if (to.meta.requiresAuth && !authed) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.meta.guest && authed) {
    return { path: '/dashboard' }
  }
  return true
})
