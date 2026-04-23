import { api } from './client'
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
  register: (email: string, username: string, password: string) =>
    api.post<RegisterResponse>('/auth/register', { email, username, password }, { auth: false }),
  logout: () => api.post<null>('/auth/logout'),
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
}

// Incomes handler returns raw JSON (arrays/objects, not {status,message,data}).
const RAW = { raw: true } as const

export interface CreateIncomeBody {
  amount: number
  currency?: string
  source?: string
  description?: string
  date?: string
}

export const incomeApi = {
  list: () => api.get<Income[]>('/incomes', RAW),
  get: (id: string) => api.get<Income>(`/incomes/${id}`, RAW),
  create: (body: CreateIncomeBody) => api.post<Income>('/incomes', body, RAW),
  update: (id: string, body: CreateIncomeBody) => api.put<Income>(`/incomes/${id}`, body, RAW),
  remove: (id: string) => api.delete<{ deleted: boolean }>(`/incomes/${id}`, RAW),
}

export interface CreateBudgetBody {
  category: string
  amount: number
  currency?: string
  period?: string
  start_date?: string
  end_date?: string
}

export const budgetApi = {
  list: () => api.get<Budget[]>('/budgets', RAW),
  get: (id: string) => api.get<Budget>(`/budgets/${id}`, RAW),
  create: (body: CreateBudgetBody) => api.post<Budget>('/budgets', body, RAW),
  update: (id: string, body: CreateBudgetBody) => api.put<Budget>(`/budgets/${id}`, body, RAW),
  remove: (id: string) => api.delete<{ deleted: boolean }>(`/budgets/${id}`, RAW),
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
  list: () => api.get<Goal[]>('/goals', RAW),
  get: (id: string) => api.get<Goal>(`/goals/${id}`, RAW),
  create: (body: CreateGoalBody) => api.post<Goal>('/goals', body, RAW),
  update: (id: string, body: CreateGoalBody) => api.put<Goal>(`/goals/${id}`, body, RAW),
  remove: (id: string) => api.delete<{ deleted: boolean }>(`/goals/${id}`, RAW),
  contribute: (id: string, amount: number) =>
    api.post<Goal>(`/goals/${id}/contribute`, { amount }, RAW),
}

export const settingsApi = {
  get: () => api.get<SettingsInfo>('/settings', RAW),
}
