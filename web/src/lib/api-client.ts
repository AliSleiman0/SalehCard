import type { ApiResponse } from '@/types'

let _accessToken: string | null = null

export function setAccessToken(t: string | null): void {
  _accessToken = t
}

const BASE = (): string =>
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? 'http://localhost:8080'

// Single-flight refresh: concurrent 401s share ONE /auth/refresh call rather than
// each firing their own (dashboards fan out wallet+orders+kyc+notifications in
// parallel). Resolves to the new access token, or null when the session is gone.
let _refreshPromise: Promise<string | null> | null = null

async function doRefresh(): Promise<string | null> {
  try {
    const res = await fetch(`${BASE()}/api/v1/auth/refresh`, {
      method: 'POST',
      credentials: 'include',
    })
    if (!res.ok) {
      import('@/stores/auth').then((m) => m.useAuthStore.getState().logout())
      return null
    }
    const data = (await res.json()) as { data?: { accessToken?: string } }
    const token = data?.data?.accessToken ?? null
    setAccessToken(token)
    return token
  } catch {
    import('@/stores/auth').then((m) => m.useAuthStore.getState().logout())
    return null
  }
}

function refreshOnce(): Promise<string | null> {
  _refreshPromise ??= doRefresh().finally(() => {
    _refreshPromise = null
  })
  return _refreshPromise
}

function buildHeaders(body: unknown, extra?: Record<string, string>): Record<string, string> {
  const headers: Record<string, string> = { ...extra }
  // Let the browser set the multipart boundary itself for FormData bodies.
  if (!(body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }
  if (_accessToken) {
    headers['Authorization'] = `Bearer ${_accessToken}`
  }
  return headers
}

function encodeBody(body: unknown): BodyInit | undefined {
  if (body === undefined) return undefined
  return body instanceof FormData ? body : JSON.stringify(body)
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  extraHeaders?: Record<string, string>,
): Promise<ApiResponse<T>> {
  const url = `${BASE()}${path}`

  const init: RequestInit = {
    method,
    headers: buildHeaders(body, extraHeaders),
    credentials: 'include',
    body: encodeBody(body),
  }

  let response: Response
  try {
    response = await fetch(url, init)
  } catch (err) {
    throw new Error(`Network error: ${err instanceof Error ? err.message : String(err)}`)
  }

  // The refresh endpoint is exempt: a 401 from it means the session is gone, so
  // there is nothing to retry.
  const isRefreshCall = path === '/api/v1/auth/refresh'

  if (response.status === 401 && !isRefreshCall) {
    const newToken = await refreshOnce()
    if (!newToken) {
      // Session gone (doRefresh already logged out) — surface the original 401.
      return response.json() as Promise<ApiResponse<T>>
    }
    // Retry once with the fresh token (headers rebuilt so Authorization updates).
    try {
      response = await fetch(url, {
        method,
        headers: buildHeaders(body, extraHeaders),
        credentials: 'include',
        body: encodeBody(body),
      })
    } catch (err) {
      throw new Error(`Network error on retry: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  return response.json() as Promise<ApiResponse<T>>
}

export const apiClient = {
  get<T>(path: string): Promise<ApiResponse<T>> {
    return request<T>('GET', path)
  },
  post<T>(
    path: string,
    body?: unknown,
    headers?: Record<string, string>,
  ): Promise<ApiResponse<T>> {
    return request<T>('POST', path, body, headers)
  },
  // postForm sends multipart/form-data (the browser sets the boundary). Used for
  // file uploads (KYC documents).
  postForm<T>(
    path: string,
    form: FormData,
    headers?: Record<string, string>,
  ): Promise<ApiResponse<T>> {
    return request<T>('POST', path, form, headers)
  },
  patch<T>(path: string, body?: unknown): Promise<ApiResponse<T>> {
    return request<T>('PATCH', path, body)
  },
  delete<T>(path: string): Promise<ApiResponse<T>> {
    return request<T>('DELETE', path)
  },
}
