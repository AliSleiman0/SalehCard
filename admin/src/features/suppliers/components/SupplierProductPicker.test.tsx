import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import type { CatalogProductView } from '../api/suppliers'

const perms = { browse: true }
vi.mock('@/stores/auth', () => ({
  useCan: () => (perm: string) => (perm === 'suppliers.view' ? perms.browse : false),
}))

const getSupplierCatalog = vi.fn()
vi.mock('../api/suppliers', () => ({
  getSupplierCatalog: (id: number) => getSupplierCatalog(id),
}))

import { SupplierProductPicker } from './SupplierProductPicker'

const soulchill: CatalogProductView = {
  upstreamId: '2',
  name: 'SoulChill',
  category: 'Apps',
  parentId: '',
  price: 0.0018,
  basePrice: 0.0018,
  currency: 'USD',
  available: true,
  productType: 'amount',
  params: ['playerId'],
  qtyMin: 1000,
  qtyMax: 1000000,
  qtyValues: [],
  mapped: false,
}

function wrap(ui: ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

describe('SupplierProductPicker', () => {
  beforeEach(() => {
    perms.browse = true
    getSupplierCatalog.mockReset()
    getSupplierCatalog.mockResolvedValue({ data: { products: [soulchill] } })
  })

  it('lists the catalog and picks the supplier id on click', async () => {
    const onChange = vi.fn()
    wrap(<SupplierProductPicker providerId={10} value="" onChange={onChange} />)
    fireEvent.click(await screen.findByRole('option', { name: /SoulChill/ }))
    expect(onChange).toHaveBeenCalledWith('2')
    expect(screen.getByRole('option', { name: /qty 1,000–1,000,000/ })).toBeInTheDocument()
  })

  it('warns when the typed id is not in the catalog (the prod "soulchill" link)', async () => {
    wrap(<SupplierProductPicker providerId={10} value="soulchill" onChange={() => {}} />)
    expect(await screen.findByRole('alert')).toHaveTextContent(/not in this supplier's catalog/)
  })

  it('warns when the order quantity is below the supplier minimum', async () => {
    wrap(<SupplierProductPicker providerId={10} value="2" onChange={() => {}} qty={1} />)
    expect(await screen.findByRole('alert')).toHaveTextContent(/sends quantity 1/)
  })

  it('shows the supplier cost for the order quantity', async () => {
    wrap(<SupplierProductPicker providerId={10} value="2" onChange={() => {}} qty={5000} />)
    expect(await screen.findByText(/9 USD/)).toBeInTheDocument()
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('falls back to typing the id without suppliers.view (no catalog call)', () => {
    perms.browse = false
    wrap(<SupplierProductPicker providerId={10} value="" onChange={() => {}} />)
    expect(screen.getByLabelText('Supplier product id')).toBeInTheDocument()
    expect(screen.queryByRole('listbox')).toBeNull()
    expect(getSupplierCatalog).not.toHaveBeenCalled()
  })

  it('explains when the catalog cannot be loaded', async () => {
    getSupplierCatalog.mockRejectedValue(new Error('404'))
    wrap(<SupplierProductPicker providerId={99} value="" onChange={() => {}} />)
    expect(await screen.findByText(/Couldn't load this supplier's catalog/)).toBeInTheDocument()
  })
})
