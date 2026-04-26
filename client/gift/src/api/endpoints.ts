import { api, requestWithStatus } from './client'
import { auth } from '../stores/auth'
import { server } from '../stores/server'
import type {
  Budget,
  Credit,
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

// --- Credits (borrowings + lendings) -------------------------------------
//
// One Credit row is a borrowing for `to` and a lending for `from`. The two
// sub-APIs below share an underlying collection but are queried/mutated via
// distinct routes so the caller-must-be-X authorization rules can apply.
//
// Helper actions (repay/take/give/collect) and PUT/DELETE behave differently
// for one-OID vs two-OID credits — the server returns 200 (applied) or 201
// (FinanceRequest created, awaiting counterparty approval). Callers should
// treat 201 as "request sent" and not optimistically reflect the new amount.

// Body shapes for create — caller must send exactly one of *_user_id (hex)
// or *_name (free-form string) for the counterparty. The other side is
// implicitly the authenticated user.
export interface CreateBorrowingBody {
  from_user_id?: string
  from_name?: string
  amount: number
  resolved_amount?: number
  currency?: string
  description?: string
  date?: string
}

export interface CreateLendingBody {
  to_user_id?: string
  to_name?: string
  amount: number
  resolved_amount?: number
  currency?: string
  description?: string
  date?: string
}

// Body for repay/take/give/collect helper actions. The server figures out
// the FinanceRequestType from the route.
export interface CreditActionBody {
  amount: number
  description?: string
}

// Result of a helper action. `applied=true` (HTTP 200) means the credit is
// already in its new state — render the new amount. `applied=false` (HTTP
// 201) means a FinanceRequest was created on a two-OID credit — render
// "request sent" / await counterparty approval. The just-created request is
// the last entry in `credit.finance_requests` whose `requested_by === me`.
export interface CreditActionResult {
  applied: boolean
  credit: Credit
}

async function postCreditAction(path: string, body: CreditActionBody): Promise<CreditActionResult> {
  const { status, data } = await requestWithStatus<Credit>(path, {
    method: 'POST',
    body: JSON.stringify(body),
  })
  return { applied: status === 200, credit: data }
}

export const borrowingApi = {
  list: () => api.get<Credit[]>('/borrowings'),
  get: (id: string) => api.get<Credit>(`/borrowings/${id}`),
  create: (body: CreateBorrowingBody) => api.post<Credit>('/borrowings', body),
  // PUT only works on one-OID credits; two-OID returns 409 — use repay/take.
  update: (id: string, body: Partial<Credit>) => api.put<Credit>(`/borrowings/${id}`, body),
  remove: (id: string) => api.delete<null>(`/borrowings/${id}`),
  // Mark money paid back to the lender (increase_resolved_amount).
  repay: (id: string, body: CreditActionBody) =>
    postCreditAction(`/borrowings/${id}/repay`, body),
  // Borrow more on the same credit (increase_amount).
  take: (id: string, body: CreditActionBody) =>
    postCreditAction(`/borrowings/${id}/take`, body),
  approveRequest: (id: string, reqId: string) =>
    api.post<Credit>(`/borrowings/${id}/requests/${reqId}/approve`, {}),
  rejectRequest: (id: string, reqId: string) =>
    api.post<Credit>(`/borrowings/${id}/requests/${reqId}/reject`, {}),
}

export const lendingApi = {
  list: () => api.get<Credit[]>('/lendings'),
  get: (id: string) => api.get<Credit>(`/lendings/${id}`),
  create: (body: CreateLendingBody) => api.post<Credit>('/lendings', body),
  update: (id: string, body: Partial<Credit>) => api.put<Credit>(`/lendings/${id}`, body),
  remove: (id: string) => api.delete<null>(`/lendings/${id}`),
  // Lend more on the same credit (increase_amount).
  give: (id: string, body: CreditActionBody) =>
    postCreditAction(`/lendings/${id}/give`, body),
  // Mark money collected back from the borrower (increase_resolved_amount).
  collect: (id: string, body: CreditActionBody) =>
    postCreditAction(`/lendings/${id}/collect`, body),
  approveRequest: (id: string, reqId: string) =>
    api.post<Credit>(`/lendings/${id}/requests/${reqId}/approve`, {}),
  rejectRequest: (id: string, reqId: string) =>
    api.post<Credit>(`/lendings/${id}/requests/${reqId}/reject`, {}),
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
