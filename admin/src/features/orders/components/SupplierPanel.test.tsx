import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import type { AdminOrder, SupplierOptions } from '../api/orders'
import { ApiError } from '@/lib/api-client'

vi.mock('@/stores/auth', () => ({ useCan: () => () => false }))
vi.mock('@/features/suppliers/api/suppliers', () => ({ getSupplierCatalog: vi.fn() }))

const rerouteOrder = vi.fn()
const getSupplierOptions = vi.fn()
vi.mock('../api/orders', () => ({
  listRerouteProviders: () =>
    Promise.resolve({
      data: [
        { id: 10, name: 'jentel' },
        { id: 11, name: 'speedcard' },
      ],
    }),
  rerouteOrder: (id: string, input: unknown) => rerouteOrder(id, input),
  getSupplierOptions: (id: string) => getSupplierOptions(id),
}))

import { SupplierPanel } from './SupplierPanel'

const stuckOrder = {
  id: 'o1',
  status: 'processing',
  total: 1.23,
  currency: 'USD',
  items: [
    {
      fulfillmentMode: 'api',
      fulfillmentType: 'account_credit',
      fulfillmentProvider: 10,
      upstreamProductId: '3258',
      qty: 1,
      amount: 10000,
      playerId: '74087345',
    },
  ],
  fulfillment: { statusTimeline: [], supplierStuckAt: '2026-11-14T15:00:00Z', providerRef: '' },
} as unknown as AdminOrder

type Opt = SupplierOptions['options'][number]
const link = (over: Partial<Opt>): Opt => ({
  linkId: 'x',
  provider: 10,
  upstreamId: '3258',
  name: 'SOUL STAR',
  enabled: true,
  active: false,
  current: false,
  tried: false,
  catalogStatus: 'ok',
  available: true,
  currency: 'USD',
  qtyOk: true,
  ...over,
})

const baseOptions: SupplierOptions = {
  canSend: true,
  fromManual: false,
  needsAck: false,
  ambiguous: false,
  qty: 10000,
  linePrice: 1.23,
  currency: 'USD',
  options: [
    link({ linkId: 'a', provider: 10, providerName: 'jentel', active: true, current: true, cost: 1.096 }),
    link({ linkId: 'b', provider: 11, providerName: 'speedcard', cost: 1.2 }),
    link({
      linkId: 'c',
      provider: 12,
      providerName: 'gift4card',
      upstreamId: '9',
      name: undefined,
      catalogStatus: 'unknown',
      available: false,
      blocked: 'not in this supplier product list any more',
    }),
  ],
}

function wrap(ui: ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

async function openModal(opts: SupplierOptions = baseOptions) {
  getSupplierOptions.mockResolvedValue({ data: opts })
  wrap(<SupplierPanel order={stuckOrder} canManage />)
  fireEvent.click(await screen.findByRole('button', { name: /Send to a supplier/ }))
  await screen.findByRole('table', { name: 'Linked suppliers' })
}

describe('SupplierPanel send-to-supplier modal', () => {
  beforeEach(() => {
    rerouteOrder.mockReset()
    getSupplierOptions.mockReset()
  })

  it('lists the product suppliers with this order cost and sends with one click — nothing typed', async () => {
    rerouteOrder.mockResolvedValue({ data: { ...stuckOrder, status: 'completed' } })
    await openModal()
    const table = screen.getByRole('table', { name: 'Linked suppliers' })
    expect(within(table).getByText('$1.10')).toBeInTheDocument()
    expect(within(table).getByText('10.9%')).toBeInTheDocument()
    expect(within(table).getByText('Current')).toBeInTheDocument()

    const toSpeedcard = screen.getByRole('button', { name: 'Send to speedcard' })
    expect(toSpeedcard).toBeDisabled() // "I understand" first
    fireEvent.click(screen.getByRole('checkbox', { name: /I understand/ }))
    expect(toSpeedcard).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Send to gift4card' })).toBeDisabled() // not in its catalog
    expect(screen.getByRole('button', { name: 'Send to jentel' })).toHaveTextContent('Send again')

    fireEvent.click(toSpeedcard)
    await vi.waitFor(() => expect(rerouteOrder).toHaveBeenCalledTimes(1))
    expect(rerouteOrder.mock.calls[0][1]).toMatchObject({ linkId: 'b', acknowledgeAtSupplier: false })
  })

  it('an order the supplier may still deliver needs the extra confirmation', async () => {
    rerouteOrder.mockResolvedValue({ data: stuckOrder })
    await openModal({ ...baseOptions, needsAck: true })
    fireEvent.click(screen.getByRole('checkbox', { name: /I understand/ }))
    const send = screen.getByRole('button', { name: 'Send to speedcard' })
    expect(send).toBeDisabled()
    fireEvent.click(screen.getByRole('checkbox', { name: /may still deliver/ }))
    expect(send).toBeEnabled()
    fireEvent.click(send)
    await vi.waitFor(() => expect(rerouteOrder).toHaveBeenCalledTimes(1))
    expect(rerouteOrder.mock.calls[0][1]).toMatchObject({ linkId: 'b', acknowledgeAtSupplier: true })
  })

  it('other supplier product: switching supplier clears the product (the #D74816BC mistake)', async () => {
    await openModal()
    fireEvent.click(screen.getByRole('checkbox', { name: /I understand/ }))
    fireEvent.click(screen.getByRole('button', { name: /Other supplier product/ }))
    const box = screen.getByPlaceholderText('e.g. 364') as HTMLInputElement
    fireEvent.change(box, { target: { value: '3258' } })
    expect(screen.getByRole('button', { name: /Send now/ })).toBeEnabled()

    fireEvent.change(screen.getByRole('combobox', { name: 'Supplier' }), { target: { value: '10' } })
    expect((screen.getByPlaceholderText('e.g. 364') as HTMLInputElement).value).toBe('')
    expect(screen.getByRole('button', { name: /Send now/ })).toBeDisabled()
  })

  it('shows the server refusal and offers "send anyway" on a low balance', async () => {
    rerouteOrder.mockRejectedValueOnce(
      new ApiError(409, 'REROUTE_LOW_BALANCE', 'that supplier balance (0.47 USD) is below this order cost'),
    )
    rerouteOrder.mockResolvedValueOnce({ data: { ...stuckOrder, status: 'completed' } })
    await openModal()
    fireEvent.click(screen.getByRole('checkbox', { name: /I understand/ }))
    fireEvent.click(screen.getByRole('button', { name: /Other supplier product/ }))
    fireEvent.change(screen.getByPlaceholderText('e.g. 364'), { target: { value: '3258' } })
    fireEvent.click(screen.getByRole('button', { name: /Send now/ }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/balance \(0.47 USD\)/)
    const send = screen.getByRole('button', { name: /Send now/ })
    expect(send).toBeDisabled()
    fireEvent.click(screen.getByRole('checkbox', { name: /balance looks too low/ }))
    expect(send).toBeEnabled()
    fireEvent.click(send)
    await vi.waitFor(() => expect(rerouteOrder).toHaveBeenCalledTimes(2))
    expect(rerouteOrder.mock.calls[1][1]).toMatchObject({ upstreamProductId: '3258', acknowledgeLowBalance: true })
  })
})
