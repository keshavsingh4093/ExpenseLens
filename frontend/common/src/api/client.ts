/**
 * Fetch wrapper shared by the portals. Paths are absolute (`/openapi/...`
 * for unauthenticated routes, `/api/...` for authenticated ones). Responses
 * use the `{success, data, error, code, fields}` envelope; `data` is
 * returned, failures throw ApiRequestError. A 401 on `/api/...` triggers one
 * single-flight refresh, then the request is retried once.
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

type Envelope<T> = { success: boolean; data?: T; error?: string; code?: string; fields?: Record<string, string> }

let accessToken: string | null = null
let refreshing: Promise<boolean> | null = null
let refreshPath: string | null = '/openapi/auth/refresh'

export const tokenStore = {
  get: () => accessToken,
  set: (t: string | null) => {
    accessToken = t
  },
}

/** Per-app setup. `refreshPath: null` disables silent refresh. */
export function configureApi(opts: { refreshPath?: string | null }) {
  if (opts.refreshPath !== undefined) refreshPath = opts.refreshPath
}

async function refresh(): Promise<boolean> {
  if (!refreshPath) return false
  if (!refreshing) {
    refreshing = fetch(refreshPath, { method: 'POST', credentials: 'include' })
      .then(async (r) => {
        if (!r.ok) return false
        const json = (await r.json()) as Envelope<{ accessToken: string }>
        if (!json.success || !json.data?.accessToken) return false
        tokenStore.set(json.data.accessToken)
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
  const res = await fetch(path, {
    credentials: 'include',
    ...rest,
    headers: {
      ...(isForm ? {} : { 'Content-Type': 'application/json' }),
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      ...(headers ?? {}),
    },
    body: body === undefined ? undefined : isForm ? body : JSON.stringify(body),
  })

  if (res.status === 401 && retry && path.startsWith('/api/')) {
    if (await refresh()) return api<T>(path, { ...opts, retry: false })
  }

  if (res.status === 204) return undefined as T

  const text = await res.text()
  let json: Envelope<T> | null = null
  try {
    json = text ? (JSON.parse(text) as Envelope<T>) : null
  } catch {
    json = null
  }
  if (!res.ok || !json || json.success !== true) {
    throw new ApiRequestError({
      status: res.status,
      code: json?.code ?? 'unknown',
      message: json?.error ?? res.statusText ?? 'Request failed',
      fields: json?.fields,
    })
  }
  return json.data as T
}
