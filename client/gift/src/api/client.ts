import type { ApiResponse, RefreshResponse } from './types'
import { auth } from '../stores/auth'
import { server } from '../stores/server'

function apiBase(): string {
  const s = server.baseUrl.value ?? ''
  return `${s}/api/v1`
}

export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

export interface RequestOptions {
  auth?: boolean
}

async function rawFetch(path: string, init: RequestInit, withAuth: boolean): Promise<Response> {
  const headers = new Headers(init.headers ?? {})
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  if (withAuth && auth.accessToken.value) {
    headers.set('Authorization', `Bearer ${auth.accessToken.value}`)
  }
  return fetch(`${apiBase()}${path}`, { ...init, headers })
}

// Every endpoint returns the unified envelope { status, message, data }, so
// there is a single parse path: surface `message` on error, return `data` on
// success.
async function parseWrapped<T>(res: Response): Promise<T> {
  const body = (await res.json().catch(() => null)) as ApiResponse<T> | null
  if (!res.ok || body?.status === 'error') {
    throw new ApiError(body?.message ?? `Request failed (${res.status})`, res.status)
  }
  return (body?.data as T) ?? (null as unknown as T)
}

async function refreshOnce(): Promise<boolean> {
  const rt = auth.refreshToken.value
  if (!rt) return false
  const res = await rawFetch('/auth/refresh', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: rt }),
  }, false)
  if (!res.ok) return false
  const body = (await res.json().catch(() => null)) as ApiResponse<RefreshResponse> | null
  if (!body || body.status !== 'success' || !body.data) return false
  auth.setTokens(body.data.token, body.data.refresh_token)
  return true
}

// Returns the parsed body alongside the HTTP status. Most callers only care
// about the body and use `request()` below; the credits endpoints need the
// status to distinguish 200 (one-OID, applied) from 201 (two-OID, awaiting
// counterparty approval).
export async function requestWithStatus<T>(
  path: string,
  init: RequestInit = {},
  opts: RequestOptions = {},
): Promise<{ status: number; data: T }> {
  const withAuth = opts.auth !== false
  let res = await rawFetch(path, init, withAuth)
  if (res.status === 401 && withAuth && auth.refreshToken.value) {
    const ok = await refreshOnce()
    if (ok) {
      res = await rawFetch(path, init, withAuth)
    } else {
      auth.clear()
    }
  }
  const data = await parseWrapped<T>(res)
  return { status: res.status, data }
}

export async function request<T>(
  path: string,
  init: RequestInit = {},
  opts: RequestOptions = {},
): Promise<T> {
  const { data } = await requestWithStatus<T>(path, init, opts)
  return data
}

export const api = {
  get: <T>(path: string, opts?: RequestOptions) =>
    request<T>(path, { method: 'GET' }, opts),
  post: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
    request<T>(path, { method: 'POST', body: body != null ? JSON.stringify(body) : undefined }, opts),
  put: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
    request<T>(path, { method: 'PUT', body: body != null ? JSON.stringify(body) : undefined }, opts),
  delete: <T>(path: string, opts?: RequestOptions) =>
    request<T>(path, { method: 'DELETE' }, opts),
}
