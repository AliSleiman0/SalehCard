import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
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
import { useKeyedPage } from '@/hooks/useKeyedPage'
import { money, downloadCsv, downloadPdf } from '@/lib/utils'
import { useCan } from '@/stores/auth'
import { toast } from '@/stores/toast'
import { useOrders } from '../hooks/useOrders'
import { useRefundBulk } from '../hooks/useOrderMutations'
import { adaptOrder, productEditPath } from '../lib/adaptOrder'
import { listOrders, MAX_BULK_RESEND, type OrderStatus, type PaymentMethod, type AdminOrder } from '../api/orders'
import { BulkResendModal } from '../components/BulkResendModal'
import type { FulfillmentType } from '@/types'
import { orderNumber } from '@/lib/orderNumber'

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

// The list's main filters live in the URL so dashboard KPI tiles can deep-link
// to them and a filtered view is shareable and survives a refresh:
//   ?status=pending|processing|completed|failed|refunded
//   ?type=code|credit|transfer      (fulfillment type)
//   ?date=today                     (current business day, server-side window)
//   ?stuck=1                        (supplier-stuck processing orders)
//   ?pay=wallet|card|usdt           (payment method)
// Anything unrecognised falls back to "no filter".
const ORDER_STATUSES: readonly OrderStatus[] = ['pending', 'processing', 'completed', 'failed', 'refunded']
function statusFromParam(raw: string | null): OrderStatus | 'all' {
  return ORDER_STATUSES.includes(raw as OrderStatus) ? (raw as OrderStatus) : 'all'
}
function ffFromParam(raw: string | null): FfKey | 'all' {
  return raw === 'code' || raw === 'credit' || raw === 'transfer' ? raw : 'all'
}
function payFromParam(raw: string | null): PaymentMethod | 'all' {
  return raw === 'wallet' || raw === 'card' || raw === 'usdt' ? raw : 'all'
}

export default function OrderListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const can = useCan()
  const canManage = can('orders.manage')
  // The product editor route is guarded by the products domain (view).
  const canViewProducts = can('products.view')
  const [searchParams, setSearchParams] = useSearchParams()

  const status = statusFromParam(searchParams.get('status'))
  const ff = ffFromParam(searchParams.get('type'))
  const today = searchParams.get('date') === 'today'
  const stuck = searchParams.get('stuck') === '1'
  const pay = payFromParam(searchParams.get('pay'))
  // Write (or clear) URL filter params in one history-replacing update.
  const setParams = (patch: Record<string, string | null>) =>
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        for (const [k, v] of Object.entries(patch)) {
          if (v) next.set(k, v)
          else next.delete(k)
        }
        return next
      },
      { replace: true },
    )
  // Picking a status leaves the stuck view (stuck orders are always processing).
  const setStatus = (v: OrderStatus | 'all') => setParams({ status: v === 'all' ? null : v, stuck: null })
  const setFf = (v: FfKey | 'all') => setParams({ type: v === 'all' ? null : v })
  const setPay = (v: PaymentMethod | 'all') => setParams({ pay: v === 'all' ? null : v })
  const [search, setSearch] = useState(searchParams.get('q') ?? '')
  const [debouncedSearch, setDebouncedSearch] = useState(search)
  const [exporting, setExporting] = useState(false)

  // Adopt the URL's ?q= (e.g. the top-bar global search navigates to /orders?q=…).
  // Keyed on the value, not the params object, so a filter's URL write doesn't
  // wipe a search typed into the box.
  const qParam = searchParams.get('q') ?? ''
  useEffect(() => {
    setSearch(qParam)
  }, [qParam])

  // Debounce the search term so we issue one request per pause, not per keystroke
  // (each non-hex term also drives a user-email lookup server-side).
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  // Filters shared by the list query and the CSV export (page added per call).
  const filters = {
    status: status === 'all' ? ('' as const) : status,
    fulfillmentType: ff === 'all' ? ('' as const) : FF_API[ff],
    paymentMethod: pay === 'all' ? ('' as const) : pay,
    q: debouncedSearch.trim(),
    date: today ? ('today' as const) : ('' as const),
    supplierStuck: stuck,
  }
  // Any filter change (in-page, a dashboard deep link, or a new search) starts
  // at page 1 — in the same render, so no request goes out for a stale page.
  const [page, setPage] = useKeyedPage(JSON.stringify(filters))

  const { data, isLoading, isError, refetch } = useOrders({ page, ...filters })

  // Not memoised: the rows' labels follow the admin's language on every render.
  const rows = (data?.data ?? []).map(adaptOrder)
  const meta = data?.meta
  const bulk = useBulk(rows.map((r) => r.id))
  const refundBulk = useRefundBulk()
  const [resending, setResending] = useState(false)

  // Bulk refund: each order refunds independently (its own atomic lock + ledger),
  // so the result is a per-order summary — some may be skipped as not-refundable.
  const runBulkRefund = () => {
    if (!window.confirm(t('ord_bulk_refund_confirm', { n: bulk.sel.length }))) return
    refundBulk.mutate(
      { ids: bulk.sel },
      {
        onSuccess: (res) => {
          const r = res.data
          const results = r?.results ?? []
          const atSupplier = results.filter((x) => x.status === 'at_supplier').length
          const skipped = results.filter((x) => x.status !== 'refunded' && x.status !== 'at_supplier').length
          const msg =
            t('ord_bulk_refunded', { refunded: r?.refunded ?? 0, total: r?.total ?? 0 }) +
            (skipped ? t('ord_bulk_skipped', { n: skipped }) : '') +
            (atSupplier ? t('ord_bulk_at_supplier', { n: atSupplier }) : '')
          const skippedAll = skipped + atSupplier
          if (skippedAll) toast.info(msg)
          else toast.success(msg)
          bulk.clear()
        },
      },
    )
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
      const header = [
        t('ord_col_id'),
        t('ord_col_customer'),
        t('ord_col_email'),
        t('col_product'),
        t('ord_col_qty'),
        t('col_amount'),
        t('ord_col_currency'),
        t('ord_col_payment'),
        t('status'),
        t('col_date'),
      ]
      const dataRows = view.map((o) => [o.id, o.customer, o.email, o.product, o.qty, o.amount.toFixed(2), o.cur, o.pay, o.status, o.date])
      const date = new Date().toISOString().slice(0, 10)
      if (format === 'pdf') {
        await downloadPdf(`orders-${date}.pdf`, t('ord_pdf_title', { date }), header, dataRows)
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
        back="/"
        crumbs={[t('grp_operations'), t('nav_orders')]}
        title={t('nav_orders')}
        sub={meta ? t('ord_list_sub', { n: meta.total.toLocaleString() }) : ''}
      >
        <button className="abtn" onClick={() => handleExport('csv')} disabled={exporting}>
          <Icon name="download" size={15} /> {exporting ? '…' : t('export')}
        </button>
        <button className="abtn" onClick={() => handleExport('pdf')} disabled={exporting}>
          <Icon name="file" size={15} /> PDF
        </button>
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> {t('ord_refresh')}
        </button>
      </PageHead>

      <div className="acard">
        <div className="toolbar">
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input
              placeholder={t('ord_search_ph')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <select
            className="select"
            value={stuck ? 'stuck' : status}
            onChange={(e) => setStatus(e.target.value as OrderStatus | 'all')}
          >
            {/* Shown only in the stuck view, so picking plain "Processing" is a
                real change that leaves it. */}
            {stuck && <option value="stuck">{t('ord_f_stuck')}</option>}
            <option value="all">{t('ord_f_all_statuses')}</option>
            <option value="completed">{t('ord_status_completed')}</option>
            <option value="processing">{t('ord_status_processing')}</option>
            <option value="pending">{t('ord_status_pending')}</option>
            <option value="refunded">{t('ord_status_refunded')}</option>
            <option value="failed">{t('ord_status_failed')}</option>
          </select>
          <select
            className="select"
            value={today ? 'today' : 'any'}
            onChange={(e) => setParams({ date: e.target.value === 'today' ? 'today' : null })}
          >
            <option value="any">{t('ord_f_any_date')}</option>
            <option value="today">{t('today')}</option>
          </select>
          <select className="select" value={pay} onChange={(e) => setPay(e.target.value as PaymentMethod | 'all')}>
            <option value="all">{t('ord_f_all_payment')}</option>
            <option value="wallet">{t('ord_pay_wallet')}</option>
            <option value="card">{t('ord_pay_card')}</option>
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
                onClick={() => setFf(k as 'all' | FfKey)}
                dotColor={k !== 'all' ? FF_DOT[k as FfKey] : undefined}
              >
                {l}
              </Chip>
            ))}
            <Chip on={stuck} onClick={() => setParams({ stuck: stuck ? null : '1', status: null })} dotColor="var(--danger)">
              {t('ord_chip_stuck')}
            </Chip>
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
                <button
                  className="abtn xs"
                  onClick={() => setResending(true)}
                  disabled={bulk.sel.length > MAX_BULK_RESEND}
                  title={
                    bulk.sel.length > MAX_BULK_RESEND
                      ? t('ord_resend_max', { n: MAX_BULK_RESEND })
                      : t('ord_resend_tip')
                  }
                >
                  <Icon name="send" size={13} /> {t('ord_resend_btn', { n: bulk.sel.length })}
                </button>
              )}
              {canManage && (
                <button className="abtn xs danger" onClick={runBulkRefund} disabled={refundBulk.isPending}>
                  <Icon name="refresh" size={13} /> {refundBulk.isPending ? '…' : t('refund')}
                </button>
              )}
            </div>
          </div>
        )}

        {resending && (
          <BulkResendModal
            ids={bulk.sel}
            onClose={() => setResending(false)}
            onDone={() => {
              setResending(false)
              bulk.clear()
            }}
          />
        )}

        {isLoading ? (
          <LoadingSpinner />
        ) : isError && !data ? (
          <ErrorState message={t('ord_load_error')} onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title={t('ord_empty')} />
        ) : (
          <>
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th style={{ width: 36 }}>
                      <Checkbox on={bulk.all} onClick={bulk.toggleAll} />
                    </th>
                    <th>{t('ord_col_id')}</th>
                    <th>{t('ord_col_customer')}</th>
                    <th>{t('col_product')}</th>
                    <th>{t('col_type')}</th>
                    <th>{t('col_amount')}</th>
                    <th>{t('ord_col_payment')}</th>
                    <th>{t('status')}</th>
                    <th>{t('col_date')}</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((o) => {
                    // Only the first line's name is shown in a row; it links to that product.
                    const first = o.raw.items[0]
                    const productHref = first ? productEditPath(o.raw, first.productId, canViewProducts) : null
                    return (
                      <tr key={o.id} className="clickable" onClick={() => navigate(`/orders/${o.id}`)}>
                        <td onClick={(e) => e.stopPropagation()}>
                          <Checkbox on={bulk.sel.includes(o.id)} onClick={() => bulk.toggle(o.id)} />
                        </td>
                        <td>
                          <span className="mono strong">{orderNumber(o.id)}</span>
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
                              <b style={{ fontSize: 12.5 }}>
                                {productHref ? (
                                  <>
                                    {/* stopPropagation: open the product, not the row's order. */}
                                    <Link
                                      to={productHref}
                                      className="prodlink"
                                      title={t('sup_open_product')}
                                      onClick={(e) => e.stopPropagation()}
                                    >
                                      {o.productTitle}
                                    </Link>
                                    {o.productMore}
                                  </>
                                ) : (
                                  o.product
                                )}
                              </b>
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
                    )
                  })}
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
