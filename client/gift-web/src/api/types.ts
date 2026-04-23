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

export interface User {
  _id?: string
  id?: string
  email: string
  name: string
  balance?: number
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
