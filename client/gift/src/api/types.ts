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
