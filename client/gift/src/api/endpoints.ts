import { api } from './client'
import { auth } from '../stores/auth'
import { server } from '../stores/server'
import type {
  Budget,
  Goal,
  Group,
  Income,
  LoginResponse,
  RegisterResponse,
  SettingsInfo,
  Spending,
  User,
} from './types'

export const authApi = {
  login: (email: string, password: string) =>
    api.post<LoginResponse>('/auth/login', { email, password }, { auth: false }),
  register: (
    email: string,
    username: string,
    password: string,
    currency: 'USD' | 'EUR' | 'UZS' = 'UZS',
  ) =>
    api.post<RegisterResponse>(
      '/auth/register',
      { email, username, password, currency },
      { auth: false },
    ),
  logout: () => api.post<null>('/auth/logout'),
  me: () => api.get<User>('/auth/me'),
}

// Users endpoints return {data: user} (no status/message wrapper).
// parseWrapped() tolerates missing status and still picks up `.data`.
export const userApi = {
  getById: (id: string) => api.get<User>(`/users/${id}`),
  search: (query: string) =>
    api.get<User[]>(`/users?query=${encodeURIComponent(query)}`),
  update: (id: string, body: Partial<User> & { password?: string; confirm_password?: string }) =>
    api.put<User>(`/users/${id}`, body),
}

export const groupApi = {
  list: (name?: string) => api.get<Group[]>(`/groups${name ? `?name=${encodeURIComponent(name)}` : ''}`),
  get: (id: string) => api.get<Group>(`/groups/${id}`),
  create: (body: { name: string; owner_id: string; member_ids: string[] }) =>
    api.post<Group>('/groups', body),
  update: (id: string, body: Partial<{ name: string; owner_id: string; member_ids: string[] }>) =>
    api.put<Group>(`/groups/${id}`, body),
  remove: (id: string) => api.delete<null>(`/groups/${id}`),
  invite: (id: string, memberId: string) =>
    api.post<Group>(`/groups/${id}/invite`, { member_id: memberId }),
  removeMember: (id: string, memberId: string) =>
    api.post<Group>(`/groups/${id}/remove`, { member_id: memberId }),
}

export interface SpendingQuery {
  user_id?: string
  group_id?: string
  category?: string
  start_date?: string
  end_date?: string
  limit?: number
  offset?: number
}

export const spendingApi = {
  query: (q: SpendingQuery = {}) => {
    const params = new URLSearchParams()
    for (const [k, v] of Object.entries(q)) {
      if (v !== undefined && v !== '' && v !== null) params.set(k, String(v))
    }
    const qs = params.toString()
    return api.get<Spending[]>(`/spendings${qs ? `?${qs}` : ''}`)
  },
  get: (id: string) => api.get<Spending>(`/spendings/${id}`),
  create: (body: {
    group_id: string
    amount: number
    currency?: string
    category?: string
    description?: string
    date?: string
  }) => api.post<Spending>('/spendings', body),
  update: (id: string, body: Partial<Spending>) =>
    api.put<Spending>(`/spendings/${id}`, body),
  remove: (id: string) => api.delete<null>(`/spendings/${id}`),
  linkBudget: (id: string, budgetId: string) =>
    api.post<null>(`/spendings/${id}/budgets/${budgetId}/link`, {}),
  unlinkBudget: (id: string, budgetId: string) =>
    api.post<null>(`/spendings/${id}/budgets/${budgetId}/unlink`, {}),
}

export interface CreateIncomeBody {
  amount: number
  currency?: string
  source?: string
  description?: string
  date?: string
}

export const incomeApi = {
  list: () => api.get<Income[]>('/incomes'),
  get: (id: string) => api.get<Income>(`/incomes/${id}`),
  create: (body: CreateIncomeBody) => api.post<Income>('/incomes', body),
  update: (id: string, body: CreateIncomeBody) => api.put<Income>(`/incomes/${id}`, body),
  remove: (id: string) => api.delete<{ deleted: boolean }>(`/incomes/${id}`),
}

export interface CreateBudgetBody {
  category: string
  limit: number
  amount?: number
  currency?: string
  period?: string
  start_date?: string
  end_date?: string
}

export const budgetApi = {
  list: () => api.get<Budget[]>('/budgets'),
  get: (id: string) => api.get<Budget>(`/budgets/${id}`),
  create: (body: CreateBudgetBody) => api.post<Budget>('/budgets', body),
  update: (id: string, body: Partial<CreateBudgetBody>) =>
    api.put<Budget>(`/budgets/${id}`, body),
  remove: (id: string) => api.delete<{ deleted: boolean }>(`/budgets/${id}`),
  increase: (id: string, amount: number) =>
    api.post<Budget>(`/budgets/${id}/increase?amount=${encodeURIComponent(String(amount))}`, {}),
  decrease: (id: string, amount: number) =>
    api.post<Budget>(`/budgets/${id}/decrease?amount=${encodeURIComponent(String(amount))}`, {}),
}

export interface CreateGoalBody {
  name: string
  description?: string
  target_amount: number
  current_amount?: number
  currency?: string
  deadline?: string
}

export const goalApi = {
  list: () => api.get<Goal[]>('/goals'),
  get: (id: string) => api.get<Goal>(`/goals/${id}`),
  create: (body: CreateGoalBody) => api.post<Goal>('/goals', body),
  update: (id: string, body: CreateGoalBody) => api.put<Goal>(`/goals/${id}`, body),
  remove: (id: string) => api.delete<{ deleted: boolean }>(`/goals/${id}`),
  contribute: (id: string, amount: number) =>
    api.post<Goal>(`/goals/${id}/contribute`, { amount }),
}

export const settingsApi = {
  get: () => api.get<SettingsInfo>('/settings'),
  // Downloads an export — JSON as a .json file, CSV as a .zip of per-collection CSVs.
  // Uses fetch directly because the response is binary/text, not a wrapped JSON payload.
  export: async (format: 'json' | 'csv') => {
    const res = await fetch(`${server.baseUrl.value}/api/v1/settings/export_data?format=${format}`, {
      method: 'POST',
      headers: auth.accessToken.value
        ? { Authorization: `Bearer ${auth.accessToken.value}` }
        : undefined,
    })
    if (!res.ok) {
      throw new Error(`Export failed (${res.status})`)
    }
    const blob = await res.blob()
    const stamp = new Date().toISOString().slice(0, 19).replace(/[T:]/g, '-')
    const ext = format === 'csv' ? 'zip' : 'json'
    const fname = `gift-export-${stamp}.${ext}`
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = fname
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  },
}
