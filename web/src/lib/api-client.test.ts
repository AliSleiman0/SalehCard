import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { apiClient, setAccessToken } from './api-client'

// Verifies the single-flight refresh: two concurrent requests that both 401 must
// share ONE /auth/refresh call and each retry once with the new token.
describe('api-client single-flight refresh', () => {
  beforeEach(() => {
    setAccessToken('old-token')
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('coalesces concurrent 401s into a single /auth/refresh', async () => {
    const calls: string[] = []
    const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
      calls.push(url)
      if (url.endsWith('/api/v1/auth/refresh')) {
        return {
          ok: true,
          status: 200,
          json: async () => ({ data: { accessToken: 'new-token' } }),
        } as Response
      }
      const auth = (init?.headers as Record<string, string> | undefined)?.['Authorization']
      // Requests using the stale token 401; the retry (new token) succeeds.
      if (auth === 'Bearer old-token') {
        return {
          ok: false,
          status: 401,
          json: async () => ({ success: false, error: { code: 'UNAUTHORIZED', message: 'nope' } }),
        } as Response
      }
      return {
        ok: true,
        status: 200,
        json: async () => ({ success: true, data: { url } }),
      } as Response
    })
    vi.stubGlobal('fetch', fetchMock)

    const [a, b] = await Promise.all([
      apiClient.get('/api/v1/orders'),
      apiClient.get('/api/v1/wallet'),
    ])

    expect(a.success).toBe(true)
    expect(b.success).toBe(true)
    const refreshCalls = calls.filter((u) => u.endsWith('/api/v1/auth/refresh'))
    expect(refreshCalls).toHaveLength(1)
  })
})
