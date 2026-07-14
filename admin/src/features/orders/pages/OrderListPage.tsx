import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  Icon,
  PageHead,
  Art,
  Avatar,
  FfBadge,
  StatusBadge,
  PayChip,
  Checkbox,
  Chip,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import type { FfKey } from '@/components'
import { useBulk } from '@/hooks/useBulk'
import { money, downloadCsv, downloadPdf } from '@/lib/utils'
import { useCan } from '@/stores/auth'
import { toast } from '@/stores/toast'
import { useOrders } from '../hooks/useOrders'
import { useRefundBulk } from '../hooks/useOrderMutations'
import { adaptOrder } from '../lib/adaptOrder'
import { listOrders, type OrderStatus, type PaymentMethod, type AdminOrder } from '../api/orders'
import type { FulfillmentType } from '@/types'

const FF_DOT: Record<FfKey, string> = {
  code: 'var(--ff-code)',
  credit: 'var(--ff-credit)',
  transfer: 'var(--ff-transfer)',
}

// UI fulfillment keys (FfKey) → API fulfillmentType values (inverse of ffKey()).
const FF_API: Record<FfKey, FulfillmentType> = {
  code: 'code',
  credit: 'account_credit',
  transfer: 'transfer',
}

// Statuses accepted from the ?status= URL param (e.g. dashboard KPI deep-links).
// Anything else falls back to 'all'.
const ORDER_STATUSES: readonly OrderStatus[] = ['pending', 'processing', 'completed', 'failed', 'refunded']
function statusFromParam(raw: string | null): OrderStatus | 'all' {
  return ORDER_STATUSES.includes(raw as OrderStatus) ? (raw as OrderStatus) : 'all'
}

export default function OrderListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const can = useCan()
  const canManage = can('orders.manage')
  const [searchParams] = useSearchParams()

  const [status, setStatus] = useState<OrderStatus | 'all'>(() => statusFromParam(searchParams.get('status')))
  const [ff, setFf] = useState<'all' | FfKey>('all')
  const [pay, setPay] = useState<PaymentMethod | 'all'>('all')
  const [search, setSearch] = useState(searchParams.get('q') ?? '')
  const [debouncedSearch, setDebouncedSearch] = useState(search)
  const [page, setPage] = useState(1)
  const [exporting, setExporting] = useState(false)

  // Adopt the URL's ?q= (e.g. the top-bar global search navigates to /orders?q=…).
  useEffect(() => {
    const q = searchParams.get('q') ?? ''
    setSearch(q)
  }, [searchParams])

  // Adopt the URL's ?status= (e.g. the dashboard KPI tiles deep-link to
  // /orders?status=failed). Garbage/absent → 'all'.
  useEffect(() => {
    setStatus(statusFromParam(searchParams.get('status')))
    setPage(1)
  }, [searchParams])

  // Debounce the search term so we issue one request per pause, not per keystroke
  // (each non-hex term also drives a user-email lookup server-side).
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  // Filters shared by the list query and the CSV export (page added per call).
  const filters = {
    status: status === 'all' ? ('' as const) : status,
    fulfillmentType: ff === 'all' ? ('' as const) : FF_API[ff],
    paymentMethod: pay === 'all' ? ('' as const) : pay,
    q: debouncedSearch.trim(),
  }

  const { data, isLoading, isError, refetch } = useOrders({ page, ...filters })

  const rows = useMemo(() => (data?.data ?? []).map(adaptOrder), [data])
  const meta = data?.meta
  const bulk = useBulk(rows.map((r) => r.id))
  const refundBulk = useRefundBulk()

  // Bulk refund: each order refunds independently (its own atomic lock + ledger),
  // so the result is a per-order summary — some may be skipped as not-refundable.
  const runBulkRefund = () => {
    if (!window.confirm(`Refund ${bulk.sel.length} order(s)? Wallet-paid orders are credited back — this cannot be undone.`))
      return
    refundBulk.mutate(
      { ids: bulk.sel },
      {
        onSuccess: (res) => {
          const r = res.data
          const skipped = r ? r.results.filter((x) => x.status !== 'refunded').length : 0
          const msg = `Refunded ${r?.refunded ?? 0} of ${r?.total ?? 0}.${skipped ? ` ${skipped} skipped (not refundable).` : ''}`
          if (skipped) toast.info(msg)
          else toast.success(msg)
          bulk.clear()
        },
      },
    )
  }

  // Reset to page 1 whenever a filter changes (search resets via its debounce).
  const onFilter = <T,>(setter: (v: T) => void) => (v: T) => {
    setter(v)
    setPage(1)
  }

  // Export orders matching the current filters (all pages — the backend caps
  // limit at 100, so page to total). selectedOnly narrows to the checked rows.
  const handleExport = async (format: 'csv' | 'pdf' = 'csv', selectedOnly = false) => {
    if (exporting) return
    setExporting(true)
    try {
      const all: AdminOrder[] = []
      let p = 1
      let pages = 1
      do {
        const res = await listOrders({ ...filters, page: p, limit: 100 })
        all.push(...(res.data ?? []))
        pages = res.meta?.pages ?? 1
        p++
      } while (p <= pages)

      let view = all.map(adaptOrder)
      if (selectedOnly) view = view.filter((o) => bulk.sel.includes(o.id))
      const header = ['Order ID', 'Customer', 'Email', 'Product', 'Qty', 'Amount', 'Currency', 'Payment', 'Status', 'Date']
      const dataRows = view.map((o) => [o.id, o.customer, o.email, o.product, o.qty, o.amount.toFixed(2), o.cur, o.pay, o.status, o.date])
      const date = new Date().toISOString().slice(0, 10)
      if (format === 'pdf') {
        await downloadPdf(`orders-${date}.pdf`, `Orders — ${date}`, header, dataRows)
      } else {
        downloadCsv(`orders-${date}.csv`, [header, ...dataRows])
      }
    } finally {
      setExporting(false)
    }
  }

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_operations'), t('nav_orders')]}
        title={t('nav_orders')}
        sub={meta ? `${meta.total.toLocaleString()} orders` : ''}
      >
        <button className="abtn" onClick={() => handleExport('csv')} disabled={exporting}>
          <Icon name="download" size={15} /> {exporting ? '…' : t('export')}
        </button>
        <button className="abtn" onClick={() => handleExport('pdf')} disabled={exporting}>
          <Icon name="file" size={15} /> PDF
        </button>
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> Refresh
        </button>
      </PageHead>

      <div className="acard">
        <div className="toolbar">
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input
              placeholder="Order ID, customer email, or code…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <select className="select" value={status} onChange={(e) => onFilter(setStatus)(e.target.value as OrderStatus | 'all')}>
            <option value="all">All statuses</option>
            <option value="completed">Completed</option>
            <option value="processing">Processing</option>
            <option value="pending">Pending</option>
            <option value="refunded">Refunded</option>
            <option value="failed">Failed</option>
          </select>
          <select className="select" value={pay} onChange={(e) => onFilter(setPay)(e.target.value as PaymentMethod | 'all')}>
            <option value="all">All payment</option>
            <option value="wallet">Wallet</option>
            <option value="card">Card</option>
            <option value="usdt">USDT</option>
          </select>
          <div className="chiprow">
            {(
              [
                ['all', t('all')],
                ['code', t('ff_code')],
                ['credit', t('ff_credit')],
                ['transfer', t('ff_transfer')],
              ] as [string, string][]
            ).map(([k, l]) => (
              <Chip
                key={k}
                on={ff === k}
                onClick={() => onFilter(setFf)(k as 'all' | FfKey)}
                dotColor={k !== 'all' ? FF_DOT[k as FfKey] : undefined}
              >
                {l}
              </Chip>
            ))}
          </div>
        </div>

        {bulk.some && (
          <div className="bulkbar">
            <Checkbox on onClick={bulk.clear} />
            <span>
              {bulk.sel.length} {t('selected')}
            </span>
            <div className="ba-act">
              <button className="abtn xs" onClick={() => handleExport('csv', true)} disabled={exporting}>
                <Icon name="download" size={13} /> {exporting ? '…' : t('export')}
              </button>
              {canManage && (
                <button className="abtn xs danger" onClick={runBulkRefund} disabled={refundBulk.isPending}>
                  <Icon name="refresh" size={13} /> {refundBulk.isPending ? '…' : t('refund')}
                </button>
              )}
            </div>
          </div>
        )}

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message="Couldn't load orders." onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title="No orders found" />
        ) : (
          <>
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th style={{ width: 36 }}>
                      <Checkbox on={bulk.all} onClick={bulk.toggleAll} />
                    </th>
                    <th>Order ID</th>
                    <th>Customer</th>
                    <th>Product</th>
                    <th>Type</th>
                    <th>Amount</th>
                    <th>Payment</th>
                    <th>{t('status')}</th>
                    <th>Date</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((o) => (
                    <tr key={o.id} className="clickable" onClick={() => navigate(`/orders/${o.id}`)}>
                      <td onClick={(e) => e.stopPropagation()}>
                        <Checkbox on={bulk.sel.includes(o.id)} onClick={() => bulk.toggle(o.id)} />
                      </td>
                      <td>
                        <span className="mono strong">{o.id.slice(-8)}</span>
                      </td>
                      <td>
                        <div className="cellprod">
                          <Avatar name={o.customer} />
                          <div className="pn">
                            <b>{o.customer}</b>
                            <span>{o.email}</span>
                          </div>
                        </div>
                      </td>
                      <td>
                        <div className="cellprod">
                          <Art art={o.art} size={28} radius={6} />
                          <div className="pn">
                            <b style={{ fontSize: 12.5 }}>{o.product}</b>
                            <span>×{o.qty}</span>
                          </div>
                        </div>
                      </td>
                      <td>
                        <FfBadge ff={o.ff} />
                      </td>
                      <td className="num strong">{money(o.amount, o.cur)}</td>
                      <td>
                        <PayChip p={o.pay} />
                      </td>
                      <td>
                        <StatusBadge s={o.status} />
                      </td>
                      <td className="muted" style={{ fontSize: 12 }}>
                        {o.date}
                      </td>
                      <td onClick={(e) => e.stopPropagation()}>
                        <div className="row-actions">
                          <span className="iact">
                            <Icon name="chevright" size={16} />
                          </span>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Pagination
              page={meta?.page ?? 1}
              pages={meta?.pages ?? 1}
              total={meta?.total ?? rows.length}
              shown={rows.length}
              limit={meta?.limit}
              label={t('pg_orders')}
              onPage={setPage}
            />
          </>
        )}
      </div>
    </div>
  )
}
