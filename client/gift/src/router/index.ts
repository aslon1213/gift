import { createRouter, createWebHistory } from 'vue-router'
import { auth } from '../stores/auth'
import { server } from '../stores/server'

const LoginView = () => import('../views/LoginView.vue')
const RegisterView = () => import('../views/RegisterView.vue')
const GroupsView = () => import('../views/GroupsView.vue')
const GroupDetailView = () => import('../views/GroupDetailView.vue')
const SpendingsView = () => import('../views/SpendingsView.vue')
const IncomesView = () => import('../views/IncomesView.vue')
const ServerSetupView = () => import('../views/ServerSetupView.vue')
const HomeView = () => import('../views/HomeView.vue')
const BudgetsView = () => import('../views/BudgetsView.vue')
const GoalsView = () => import('../views/GoalsView.vue')
const SettingsView = () => import('../views/SettingsView.vue')
const LedgerView = () => import('../views/LedgerView.vue')
const LedgerDetailView = () => import('../views/LedgerDetailView.vue')

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/home' },
    { path: '/setup', component: ServerSetupView, meta: { setup: true } },
    { path: '/login', component: LoginView, meta: { guest: true } },
    { path: '/register', component: RegisterView, meta: { guest: true } },
    { path: '/home', component: HomeView, meta: { requiresAuth: true } },
    { path: '/dashboard', redirect: '/home' },
    { path: '/groups', component: GroupsView, meta: { requiresAuth: true } },
    { path: '/groups/:id', component: GroupDetailView, meta: { requiresAuth: true }, props: true },
    { path: '/spendings', component: SpendingsView, meta: { requiresAuth: true } },
    { path: '/incomes', component: IncomesView, meta: { requiresAuth: true } },
    { path: '/budgets', component: BudgetsView, meta: { requiresAuth: true } },
    { path: '/goals', component: GoalsView, meta: { requiresAuth: true } },
    { path: '/settings', component: SettingsView, meta: { requiresAuth: true } },
    { path: '/ledger', component: LedgerView, meta: { requiresAuth: true } },
    {
      path: '/borrowings/:id',
      component: LedgerDetailView,
      meta: { requiresAuth: true },
      props: (route) => ({ side: 'borrowing', id: route.params.id }),
    },
    {
      path: '/lendings/:id',
      component: LedgerDetailView,
      meta: { requiresAuth: true },
      props: (route) => ({ side: 'lending', id: route.params.id }),
    },
  ],
})

router.beforeEach((to) => {
  const configured = server.isConfigured.value
  const authed = auth.isAuthenticated.value

  if (!configured && !to.meta.setup) {
    return { path: '/setup', query: { redirect: to.fullPath } }
  }
  if (configured && to.meta.setup && authed) {
    return { path: '/home' }
  }
  if (to.meta.requiresAuth && !authed) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.meta.guest && authed) {
    return { path: '/home' }
  }
  return true
})
