import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import type { AdminOrder } from '../api/orders'
import { ApiError } from '@/lib/api-client'

vi.mock('@/stores/auth', () => ({ useCan: () => () => false }))
vi.mock('@/features/suppliers/api/suppliers', () => ({ getSupplierCatalog: vi.fn() }))

const rerouteOrder = vi.fn()
vi.mock('../api/orders', () => ({
  listRerouteProviders: () =>
    Promise.resolve({
      data: [
        { id: 10, name: 'jentel' },
        { id: 11, name: 'speedcard' },
      ],
    }),
  rerouteOrder: (id: string, input: unknown) => rerouteOrder(id, input),
}))

import { SupplierPanel } from './SupplierPanel'

const stuckOrder = {
  id: 'o1',
  status: 'processing',
  total: 1.23,
  items: [{ fulfillmentMode: 'api', fulfillmentType: 'account_credit', fulfillmentProvider: 10, qty: 1, amount: 10000, playerId: '74087345' }],
  fulfillment: { statusTimeline: [], supplierStuckAt: '2026-11-14T15:00:00Z', providerRef: '' },
} as unknown as AdminOrder

function wrap(ui: ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

async function openModal() {
  wrap(<SupplierPanel order={stuckOrder} canManage />)
  fireEvent.click(await screen.findByRole('button', { name: /Send to another supplier/ }))
  fireEvent.click(screen.getByRole('checkbox', { name: /I understand/ }))
}

describe('SupplierPanel re-send modal', () => {
  beforeEach(() => rerouteOrder.mockReset())

  it('clears the supplier product when the supplier changes (the #D74816BC mistake)', async () => {
    await openModal()
    const box = screen.getByPlaceholderText('e.g. 364') as HTMLInputElement
    fireEvent.change(box, { target: { value: '3258' } })
    const send = screen.getByRole('button', { name: /Send now/ })
    expect(send).toBeEnabled()

    fireEvent.change(screen.getByRole('combobox', { name: 'Supplier' }), { target: { value: '10' } })
    expect((screen.getByPlaceholderText('e.g. 364') as HTMLInputElement).value).toBe('')
    expect(screen.getByRole('button', { name: /Send now/ })).toBeDisabled()
  })

  it('shows the server refusal and offers "send anyway" on a low balance', async () => {
    rerouteOrder.mockRejectedValueOnce(
      new ApiError(409, 'REROUTE_LOW_BALANCE', "that supplier's balance (0.47 USD) is below this order's cost"),
    )
    rerouteOrder.mockResolvedValueOnce({ data: { ...stuckOrder, status: 'completed' } })
    await openModal()
    fireEvent.change(screen.getByPlaceholderText('e.g. 364'), { target: { value: '3258' } })
    fireEvent.click(screen.getByRole('button', { name: /Send now/ }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/balance \(0.47 USD\)/)
    const send = screen.getByRole('button', { name: /Send now/ })
    expect(send).toBeDisabled()
    fireEvent.click(screen.getByRole('checkbox', { name: /Send anyway/ }))
    expect(send).toBeEnabled()
    fireEvent.click(send)
    await vi.waitFor(() => expect(rerouteOrder).toHaveBeenCalledTimes(2))
    expect(rerouteOrder.mock.calls[1][1]).toMatchObject({ upstreamProductId: '3258', acknowledgeLowBalance: true })
  })
})
