/**
 * Minimal fetch wrapper. Adds the bearer token, parses the API error envelope,
 * and transparently refreshes the access token once on 401.
 */

export type ApiError = {
  status: number
  code: string
  message: string
  fields?: Record<string, string>
}

export class ApiRequestError extends Error implements ApiError {
  status: number
  code: string
  fields?: Record<string, string>
  constructor(e: ApiError) {
    super(e.message)
    this.status = e.status
    this.code = e.code
    this.fields = e.fields
  }
}

let accessToken: string | null = null
let refreshing: Promise<boolean> | null = null

export const tokenStore = {
  get: () => accessToken,
  set: (t: string | null) => {
    accessToken = t
  },
}

const BASE = '/api/v1'

async function refresh(): Promise<boolean> {
  if (!refreshing) {
    refreshing = fetch(`${BASE}/auth/refresh`, { method: 'POST', credentials: 'include' })
      .then(async (r) => {
        if (!r.ok) return false
        const data = (await r.json()) as { accessToken: string }
        tokenStore.set(data.accessToken)
        return true
      })
      .catch(() => false)
      .finally(() => {
        refreshing = null
      })
  }
  return refreshing
}

type Opts = Omit<RequestInit, 'body'> & { body?: unknown; retry?: boolean }

export async function api<T>(path: string, opts: Opts = {}): Promise<T> {
  const { body, retry = true, headers, ...rest } = opts
  const isForm = body instanceof FormData
  const res = await fetch(`${BASE}${path}`, {
    credentials: 'include',
    ...rest,
    headers: {
      ...(isForm ? {} : { 'Content-Type': 'application/json' }),
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      ...(headers ?? {}),
    },
    body: body === undefined ? undefined : isForm ? body : JSON.stringify(body),
  })

  if (res.status === 401 && retry && !path.startsWith('/auth/')) {
    if (await refresh()) return api<T>(path, { ...opts, retry: false })
  }

  if (res.status === 204) return undefined as T

  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (!res.ok) {
    const err = data?.error ?? { code: 'unknown', message: res.statusText }
    throw new ApiRequestError({ status: res.status, ...err })
  }
  return data as T
}
