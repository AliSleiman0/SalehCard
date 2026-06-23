import type { ApiResponse } from '@/types'

let _accessToken: string | null = null

export function setAccessToken(t: string | null): void {
  _accessToken = t
}

const BASE = (): string =>
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? 'http://localhost:8080'

let _isRefreshing = false
let _didRetry = false

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  extraHeaders?: Record<string, string>,
): Promise<ApiResponse<T>> {
  const url = `${BASE()}${path}`

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...extraHeaders,
  }

  if (_accessToken) {
    headers['Authorization'] = `Bearer ${_accessToken}`
  }

  const init: RequestInit = {
    method,
    headers,
    credentials: 'include',
  }

  if (body !== undefined) {
    init.body = JSON.stringify(body)
  }

  let response: Response

  try {
    response = await fetch(url, init)
  } catch (err) {
    throw new Error(`Network error: ${err instanceof Error ? err.message : String(err)}`)
  }

  // The refresh endpoint is exempt: a 401 from it means the session is gone, so
  // there is nothing to retry — re-entering the refresh flow would just POST
  // /auth/refresh a second time and log a duplicate 401.
  const isRefreshCall = path === '/api/v1/auth/refresh'

  if (response.status === 401 && !_didRetry && !isRefreshCall) {
    _didRetry = true

    if (!_isRefreshing) {
      _isRefreshing = true
      try {
        const refreshRes = await fetch(`${BASE()}/api/v1/auth/refresh`, {
          method: 'POST',
          credentials: 'include',
        })

        if (refreshRes.ok) {
          const refreshData = await refreshRes.json() as { data?: { accessToken?: string } }
          const newToken = refreshData?.data?.accessToken ?? null
          setAccessToken(newToken)
        } else {
          import('@/stores/auth').then((m) => {
            m.useAuthStore.getState().logout()
          })
          _isRefreshing = false
          _didRetry = false
          return response.json() as Promise<ApiResponse<T>>
        }
      } catch {
        import('@/stores/auth').then((m) => {
          m.useAuthStore.getState().logout()
        })
        _isRefreshing = false
        _didRetry = false
        throw new Error('Token refresh failed')
      }
      _isRefreshing = false
    }

    // Retry original request with new token
    const retryHeaders: Record<string, string> = {
      'Content-Type': 'application/json',
      ...extraHeaders,
    }
    if (_accessToken) {
      retryHeaders['Authorization'] = `Bearer ${_accessToken}`
    }

    try {
      response = await fetch(url, { ...init, headers: retryHeaders })
    } catch (err) {
      _didRetry = false
      throw new Error(`Network error on retry: ${err instanceof Error ? err.message : String(err)}`)
    }

    _didRetry = false
  } else {
    _didRetry = false
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
  patch<T>(path: string, body?: unknown): Promise<ApiResponse<T>> {
    return request<T>('PATCH', path, body)
  },
  delete<T>(path: string): Promise<ApiResponse<T>> {
    return request<T>('DELETE', path)
  },
}
