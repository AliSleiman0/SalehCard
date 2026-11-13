import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

// Permissions the mocked useCan grants; '*' = super admin.
let perms: string[] = []
vi.mock('@/stores/auth', () => ({
  useCan: () => (p: string) => perms.includes('*') || perms.includes(p),
}))

const stats = {
  revenueToday: 120,
  revenueDeltaPct: 5,
  ordersToday: 3,
  ordersDeltaPct: 0,
  revenueSpark: [],
  ordersSpark: [],
  activeUsers: 7,
  walletTopups: 50,
  pendingTransfers: 2,
  pendingTopups: 4,
  pendingKyc: 1,
  lowStockCount: 6,
  ordersCompleted: 10,
  ordersPending: 1,
  ordersFailed: 2,
  ordersRefunded: 3,
  stuckUpstream: 1,
}
vi.mock('../hooks/useDashboard', () => ({
  useDashboardStats: () => ({ data: { data: stats } }),
  useLowStock: () => ({ data: { data: [] } }),
  useRevenueChart: () => ({ data: { data: { series: [], labels: [] } } }),
  useFulfillmentBreakdown: () => ({ data: { data: [] } }),
  useHealth: () => ({ data: { api: true, db: true } }),
}))
vi.mock('@/features/orders/hooks/useOrders', () => ({ useOrders: () => ({ data: { data: [] } }) }))
vi.mock('@/features/bridge/hooks/useBridge', () => ({
  useBridgeDevices: () => ({ data: { data: [{ id: 'd1', name: 'Phone', touchBalance: 80, alfaBalance: null }] } }),
}))
vi.mock('@/features/suppliers/hooks/useSuppliers', () => ({
  useSuppliersList: () => ({
    data: { data: { suppliers: [{ id: 'jentel', name: 'jentel', kind: 'panel', balanceText: '-$450.29', health: 'ok' }] } },
  }),
}))

import DashboardPage from './DashboardPage'

function renderPage() {
  return render(
    <MemoryRouter>
      <DashboardPage />
    </MemoryRouter>,
  )
}

/** The href of the tile whose label is `label`, or null when it isn't a link. */
function tileHref(label: string): string | null {
  const labels = [...document.querySelectorAll('.kpi .k-label')].filter((el) => el.textContent === label)
  expect(labels).toHaveLength(1)
  const tile = labels[0].closest('.kpi')
  expect(tile).not.toBeNull()
  return tile!.tagName === 'A' ? tile!.getAttribute('href') : null
}

describe('DashboardPage KPI tiles', () => {
  beforeEach(() => {
    perms = []
  })

  it('every tile links to its filtered page for a super admin', () => {
    perms = ['*']
    renderPage()
    expect(tileHref("Today's revenue")).toBe('/orders?status=completed&date=today')
    expect(tileHref("Today's orders")).toBe('/orders?status=completed&date=today')
    expect(tileHref('Active users')).toBe('/users')
    expect(tileHref('Wallet top-ups')).toBe('/topups?status=approved')
    expect(tileHref('Pending transfers')).toBe('/orders?status=processing&type=transfer')
    expect(tileHref('Pending top-ups')).toBe('/topups?status=pending')
    expect(tileHref('Pending KYC')).toBe('/kyc?status=pending')
    expect(tileHref('Low-stock alerts')).toBe('/inventory?view=low')
    expect(tileHref('Completed orders')).toBe('/orders?status=completed')
    expect(tileHref('Pending orders')).toBe('/orders?status=pending')
    expect(tileHref('Failed orders')).toBe('/orders?status=failed')
    expect(tileHref('Refunded orders')).toBe('/orders?status=refunded')
    expect(tileHref('Stuck upstream')).toBe('/orders?stuck=1')
    expect(tileHref('Touch balance')).toBe('/bridge')
    expect(tileHref('jentel balance')).toBe('/suppliers')
    expect(screen.getByText('-$450.29')).toBeInTheDocument()
  })

  it('a custom role only gets links to pages it can view', () => {
    perms = ['dashboard.view', 'kyc.view']
    renderPage()
    expect(tileHref('Pending KYC')).toBe('/kyc?status=pending')
    // Tiles still show their numbers, but as inert cards — no dead links.
    for (const label of ["Today's orders", 'Active users', 'Pending top-ups', 'Low-stock alerts', 'Failed orders', 'Stuck upstream']) {
      expect(tileHref(label)).toBeNull()
    }
    // Bridge / supplier tiles aren't rendered at all without their view perm.
    expect(screen.queryByText('Touch balance')).toBeNull()
    expect(screen.queryByText('jentel balance')).toBeNull()
  })

  it('tiles are real, keyboard-focusable links', () => {
    perms = ['*']
    renderPage()
    const link = [...document.querySelectorAll<HTMLAnchorElement>('a.kpi')].find((a) => a.textContent?.includes('Failed orders'))!
    expect(link).toHaveClass('kpi', 'kpi-link', 'crit')
    link.focus()
    expect(document.activeElement).toBe(link)
  })
})
