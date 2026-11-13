import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom'
import type { OrderListParams } from '../api/orders'

vi.mock('@/stores/auth', () => ({ useCan: () => () => true }))
const calls: OrderListParams[] = []
vi.mock('../hooks/useOrders', () => ({
  useOrders: (p: OrderListParams) => {
    calls.push(p)
    return { data: { data: [], meta: { total: 0, page: 1, pages: 1, limit: 20 } }, isLoading: false, isError: false, refetch: vi.fn() }
  },
}))
vi.mock('../hooks/useOrderMutations', () => ({ useRefundBulk: () => ({ mutate: vi.fn(), isPending: false }) }))

import OrderListPage from './OrderListPage'

let search = ''
function LocationProbe() {
  search = useLocation().search
  return null
}

function renderAt(url: string) {
  return render(
    <MemoryRouter initialEntries={[url]}>
      <Routes>
        <Route
          path="/orders"
          element={
            <>
              <OrderListPage />
              <LocationProbe />
            </>
          }
        />
      </Routes>
    </MemoryRouter>,
  )
}
const last = () => calls[calls.length - 1]

describe('OrderListPage URL filters (dashboard deep links)', () => {
  beforeEach(() => {
    calls.length = 0
  })

  it("Today's orders link → completed + today", () => {
    renderAt('/orders?status=completed&date=today')
    expect(last()).toMatchObject({ status: 'completed', date: 'today', supplierStuck: false })
  })

  it('Pending transfers link → processing + transfer type', () => {
    renderAt('/orders?status=processing&type=transfer')
    expect(last()).toMatchObject({ status: 'processing', fulfillmentType: 'transfer' })
  })

  it('Stuck upstream link → supplierStuck, and picking a status leaves the stuck view', () => {
    renderAt('/orders?stuck=1')
    expect(last()).toMatchObject({ supplierStuck: true })
    const statusSelect = screen.getByDisplayValue('Processing')
    fireEvent.change(statusSelect, { target: { value: 'failed' } })
    expect(search).toBe('?status=failed')
    expect(last()).toMatchObject({ status: 'failed', supplierStuck: false })
  })

  it('garbage params fall back to no filter', () => {
    renderAt('/orders?status=bogus&type=nope&date=yesterday&stuck=yes')
    expect(last()).toMatchObject({ status: '', fulfillmentType: '', date: '', supplierStuck: false })
  })

  it('changing the date filter writes it to the URL (shareable / survives refresh)', () => {
    renderAt('/orders?status=completed')
    fireEvent.change(screen.getByDisplayValue('Any date'), { target: { value: 'today' } })
    expect(new URLSearchParams(search).get('date')).toBe('today')
    expect(new URLSearchParams(search).get('status')).toBe('completed')
    expect(last()).toMatchObject({ status: 'completed', date: 'today' })
  })
})
