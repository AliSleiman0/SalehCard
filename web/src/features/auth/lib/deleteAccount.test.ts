import { describe, it, expect, vi, afterEach } from 'vitest'
import { canForceDelete, forfeitBalance } from './deleteAccount'
import { deleteAccount } from '../api/auth'

describe('canForceDelete', () => {
  it('offers "Delete anyway" only for wallet balance / orders in flight', () => {
    expect(canForceDelete('WALLET_NOT_EMPTY')).toBe(true)
    expect(canForceDelete('ORDERS_IN_FLIGHT')).toBe(true)
    expect(canForceDelete('PAYMENTS_PENDING')).toBe(false)
    expect(canForceDelete('TOPUPS_PENDING')).toBe(false)
    expect(canForceDelete('NETWORK_ERROR')).toBe(false)
    expect(canForceDelete(undefined)).toBe(false)
  })
})

describe('forfeitBalance', () => {
  it('quotes only balances of at least one cent', () => {
    expect(forfeitBalance(12.5)).toBe(12.5)
    expect(forfeitBalance(0.01)).toBe(0.01)
    expect(forfeitBalance(0.004)).toBeNull()
    expect(forfeitBalance(0)).toBeNull()
    expect(forfeitBalance(undefined)).toBeNull()
  })
})

describe('deleteAccount', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  function stubFetch() {
    const fetchMock = vi.fn<(url: string, init?: RequestInit) => Promise<Response>>(
      async () =>
        ({
          ok: true,
          status: 200,
          json: async () => ({ success: true, data: { deleted: true } }),
        }) as Response,
    )
    vi.stubGlobal('fetch', fetchMock)
    return fetchMock
  }

  it('sends no body without force', async () => {
    const fetchMock = stubFetch()
    await deleteAccount()
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toMatch(/\/api\/v1\/users\/me$/)
    expect(init?.method).toBe('DELETE')
    expect(init?.body).toBeUndefined()
  })

  it('sends {"force": true} only when forced', async () => {
    const fetchMock = stubFetch()
    await deleteAccount(true)
    const [, init] = fetchMock.mock.calls[0]
    expect(init?.method).toBe('DELETE')
    expect(JSON.parse(String(init?.body))).toEqual({ force: true })
  })
})
