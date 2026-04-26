export interface ApiResponse<T> {
  status: 'success' | 'error'
  message: string
  data: T | null
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
}

export interface RefreshResponse {
  token: string
  refresh_token: string
}

export interface RegisterResponse {
  id: string
  email: string
  username: string
}

export type CurrencyCode = 'USD' | 'EUR' | 'UZS'

export interface User {
  _id?: string
  id?: string
  email: string
  name: string
  balance?: number
  currency?: CurrencyCode
  credits?: CreditSummary
  created_at?: string
  updated_at?: string
}

// Returned on /auth/me alongside the user. Server-side aggregation across
// every credit the user is a party to — the client should not recompute.
export interface CreditSummary {
  borrowed: number
  lent: number
  outstanding_borrowed: number
  outstanding_lent: number
  // Positive = others owe you. Negative = you owe others.
  net_credit: number
}

// FlexID: either an ObjectID reference to a registered user, or a free-form
// name (for counterparties who don't have an account). is_oid distinguishes
// the two regimes; the server treats them very differently (see Credit).
export interface FlexID {
  oid: string
  str: string
  is_oid: boolean
}

export type FinanceRequestType =
  | 'increase_amount'
  | 'decrease_amount'
  | 'increase_resolved_amount'
  | 'decrease_resolved_amount'

export type FinanceRequestStatus = 'pending' | 'approved' | 'rejected'

export interface FinanceRequest {
  id: string
  type: FinanceRequestType
  amount: number
  description: string
  requested_by: string
  status: FinanceRequestStatus
  decided_by?: string
  decided_at?: string
  created_at?: string
  updated_at?: string
}

// A single credit record. The same row is a borrowing for `to` and a lending
// for `from`; there's one collection, two views. When both sides are OIDs
// (two-OID credit) financial mutations route through finance_requests.
export interface Credit {
  _id?: string
  id?: string
  from: FlexID
  to: FlexID
  amount: number
  resolved_amount: number
  currency: string
  description: string
  date: string
  // True once resolved_amount >= amount (and amount > 0).
  resolved?: boolean
  finance_requests?: FinanceRequest[]
  created_at?: string
  updated_at?: string
}

export interface Group {
  _id?: string
  id?: string
  name: string
  owner_id: string
  member_ids: string[]
  created_at?: string
  updated_at?: string
}

export interface Spending {
  _id?: string
  id?: string
  user_id: string
  group_id: string
  amount: number
  currency: string
  category: string
  description: string
  date: string
  budgets?: string[]
  created_at?: string
  updated_at?: string
}

export interface Income {
  id?: string
  user_id: string
  amount: number
  currency: string
  source: string
  description: string
  date: string
  created_at?: string
  updated_at?: string
}

export interface Budget {
  id?: string
  user_id: string
  category: string
  limit: number
  amount: number
  currency: string
  period: string
  start_date: string
  end_date: string
  created_at?: string
  updated_at?: string
}

export interface Goal {
  id?: string
  user_id: string
  name: string
  description: string
  target_amount: number
  current_amount: number
  currency: string
  deadline: string
  created_at?: string
  updated_at?: string
}

export interface SettingsInfo {
  server: {
    host: string
    online: boolean
    started_at: string
    uptime: string
    uptime_seconds: number
    version: string
  }
  stats: {
    users: number
    groups: number
    budgets: number
    goals: number
    db_size_bytes: number
    mem_mb: number
    goroutines: number
  }
  profile: {
    id: string
    email: string
    name: string
    currency: CurrencyCode
    balance: number
  }
}
