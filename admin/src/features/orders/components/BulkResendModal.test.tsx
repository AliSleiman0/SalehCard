import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

const rerouteBulkOrders = vi.fn()
vi.mock('../api/orders', () => ({
  MAX_BULK_RESEND: 5,
  listRerouteProviders: () => Promise.resolve({ data: [{ id: 10, name: 'jentel' }] }),
  rerouteBulkOrders: (input: unknown) => rerouteBulkOrders(input),
}))

import { BulkResendModal } from './BulkResendModal'

const A = '6a3f03cdea6747f81d0bad01'
const B = '6a3f03cdea6747f81d0bad02'

function renderModal(ids: string[]) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const onDone = vi.fn()
  render(
    <QueryClientProvider client={qc}>
      <BulkResendModal ids={ids} onClose={vi.fn()} onDone={onDone} />
    </QueryClientProvider>,
  )
  return { onDone }
}

describe('BulkResendModal', () => {
  it('re-sends after the double-delivery OK and lists each order outcome', async () => {
    rerouteBulkOrders.mockResolvedValue({
      data: {
        sent: 1,
        total: 2,
        results: [
          { id: A, status: 'completed', provider: 10, upstreamId: '3258' },
          { id: B, status: 'needs_ack', message: 'the current supplier may still deliver this order' },
        ],
      },
    })
    const { onDone } = renderModal([A, B])
    const go = screen.getByRole('button', { name: /Re-send 2/ })
    expect(go).toBeDisabled()
    fireEvent.click(screen.getByRole('checkbox', { name: /I understand/ }))
    fireEvent.click(go)

    const table = await screen.findByRole('table', { name: 'Re-send results' })
    expect(rerouteBulkOrders).toHaveBeenCalledWith({
      ids: [A, B],
      acknowledgeAtSupplier: false,
      acknowledgeAmbiguous: false,
      acknowledgeLowBalance: false,
    })
    expect(within(table).getByText('Delivered')).toBeInTheDocument()
    expect(within(table).getByText('Needs confirmation')).toBeInTheDocument()
    expect(screen.getByText(/Sent/)).toHaveTextContent('Sent 1 of 2.')
    fireEvent.click(screen.getByRole('button', { name: 'Done' }))
    expect(onDone).toHaveBeenCalled()
  })

  it('refuses more than 5 orders', () => {
    renderModal(['1', '2', '3', '4', '5', '6'])
    fireEvent.click(screen.getByRole('checkbox', { name: /I understand/ }))
    expect(screen.getByRole('button', { name: /Re-send 6/ })).toBeDisabled()
    expect(screen.getByRole('alert')).toHaveTextContent(/at most 5/)
  })
})
