import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import type { LinkRow } from '../lib/supplierLinks'

const quoteSupplierLinks = vi.fn()
const searchSuppliers = vi.fn()
vi.mock('../api/products', () => ({
  quoteSupplierLinks: (links: unknown) => quoteSupplierLinks(links),
  searchSuppliers: (q: string) => searchSuppliers(q),
}))

import { SupplierLinksTable } from './SupplierLinksTable'

const providers = [
  { id: 10, name: 'jentel' },
  { id: 11, name: 'speedcard' },
]

function wrap(ui: ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

let latest: LinkRow[] = []
function Harness({ initial }: { initial: LinkRow[] }) {
  const [rows, setRows] = useState(initial)
  latest = rows
  return (
    <SupplierLinksTable
      rows={rows}
      onChange={setRows}
      providers={providers}
      sellPrice={1.23}
      blockUnits={10000}
      amountRange={{ min: 10000, max: 50000000 }}
      searchHint="SOUL STAR"
    />
  )
}

const jentel: LinkRow = { key: 'a', id: 'a', provider: 10, upstreamId: '3258', enabled: true, active: true }

describe('SupplierLinksTable', () => {
  beforeEach(() => {
    quoteSupplierLinks.mockReset()
    searchSuppliers.mockReset()
    quoteSupplierLinks.mockImplementation((links: { provider: number; upstreamId: string }[]) =>
      Promise.resolve({
        data: {
          quotes: links.map((l) =>
            l.provider === 10
              ? { ...l, status: 'ok', name: 'SOUL STAR', price: 0.0001096153, currency: 'USD', available: true, qtyMin: 10000, qtyMax: 50000000 }
              : { ...l, status: 'ok', name: 'SOUL STAR', price: 0.00013, currency: 'USD', available: false },
          ),
        },
      }),
    )
    searchSuppliers.mockResolvedValue({
      data: {
        results: [
          { provider: 10, providerName: 'jentel', upstreamId: '3258', name: 'SOUL STAR', price: 0.0001096, currency: 'USD', available: true },
          { provider: 11, providerName: 'speedcard', upstreamId: '4410', name: 'SOUL STAR', price: 0.00013, currency: 'USD', available: false },
        ],
        unavailable: [{ provider: 12, providerName: 'gift4card', reason: 'unavailable' }],
      },
    })
  })

  it('shows the live cost per selling block and the margin', async () => {
    wrap(<Harness initial={[jentel]} />)
    expect(await screen.findByText('$1.10')).toBeInTheDocument()
    expect(screen.getByText('10.9%')).toBeInTheDocument()
    expect(screen.getByText('Active', { selector: 'span' })).toBeInTheDocument()
  })

  it('links a product found at all suppliers with one click, then makes it active', async () => {
    wrap(<Harness initial={[jentel]} />)
    fireEvent.click(screen.getByRole('button', { name: /Find at all suppliers/ }))
    expect(await screen.findByText(/gift4card: couldn't load/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Link SOUL STAR at jentel' })).toBeDisabled() // already linked
    fireEvent.click(screen.getByRole('button', { name: 'Link SOUL STAR at speedcard' }))
    expect(latest.map((r) => `${r.provider}:${r.upstreamId}:${r.active}`)).toEqual(['10:3258:true', '11:4410:false'])
    expect(searchSuppliers).toHaveBeenCalledWith('SOUL STAR')

    fireEvent.click(screen.getByRole('button', { name: 'Done' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Make active' }))
    expect(latest.map((r) => r.active)).toEqual([false, true])
    fireEvent.click(screen.getByRole('button', { name: 'Move speedcard up' }))
    expect(latest.map((r) => r.provider)).toEqual([11, 10])
  })
})
