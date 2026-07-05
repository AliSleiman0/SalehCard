import type { ApiResponse } from '@/types'

// Access token is held in memory only — never localStorage / sessionStorage
// (same rule as /web). Auth logic is intentionally duplicated rather than shared
// via a package; see CONVENTIONS.md → "Admin app".
let _accessToken: string | null = null

export function setAccessToken(t: string | null): void {
  _accessToken = t
}

export function getAccessToken(): string | null {
  return _accessToken
}

const BASE = (): string =>
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? 'http://localhost:8080'

/** Error carrying the parsed API envelope so callers/react-query can branch on it. */
export class ApiError extends Error {
  status: number
  code: string
  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<ApiResponse<T>> {
  const url = `${BASE()}${path}`

  // Detect FormData from the body itself, so the flag can never disagree with
  // the payload (a FormData body passed to post() Just Works). Skip the JSON
  // Content-Type for it — the browser sets the multipart boundary itself.
  const isFormData = body instanceof FormData
  const headers: Record<string, string> = isFormData ? {} : { 'Content-Type': 'application/json' }
  if (_accessToken) headers['Authorization'] = `Bearer ${_accessToken}`

  const init: RequestInit = { method, headers, credentials: 'include' }
  if (body !== undefined) init.body = isFormData ? (body as FormData) : JSON.stringify(body)

  let response: Response
  try {
    response = await fetch(url, init)
  } catch (err) {
    throw new Error(`Network error: ${err instanceof Error ? err.message : String(err)}`)
  }

  if (response.status === 401 || response.status === 403) {
    // Unauthorized/forbidden — drop the session so the guard redirects to /login.
    import('@/stores/auth').then((m) => m.useAuthStore.getState().logout())
  }

  // 204 No Content (e.g. DELETE) has no JSON body.
  if (response.status === 204) {
    return { success: true } as ApiResponse<T>
  }

  let parsed: ApiResponse<T>
  try {
    parsed = (await response.json()) as ApiResponse<T>
  } catch {
    throw new ApiError(response.status, 'invalid_response', 'Malformed response from server')
  }

  if (!response.ok || parsed.success === false) {
    const code = parsed.error?.code ?? 'error'
    const message = parsed.error?.message ?? `Request failed (${response.status})`
    throw new ApiError(response.status, code, message)
  }

  return parsed
}

export const apiClient = {
  get<T>(path: string): Promise<ApiResponse<T>> {
    return request<T>('GET', path)
  },
  post<T>(path: string, body?: unknown): Promise<ApiResponse<T>> {
    return request<T>('POST', path, body)
  },
  put<T>(path: string, body?: unknown): Promise<ApiResponse<T>> {
    return request<T>('PUT', path, body)
  },
  patch<T>(path: string, body?: unknown): Promise<ApiResponse<T>> {
    return request<T>('PATCH', path, body)
  },
  delete<T>(path: string): Promise<ApiResponse<T>> {
    return request<T>('DELETE', path)
  },
  upload<T>(path: string, formData: FormData): Promise<ApiResponse<T>> {
    return request<T>('POST', path, formData)
  },
}
