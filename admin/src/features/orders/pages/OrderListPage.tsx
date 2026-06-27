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
import { money } from '@/lib/utils'
import { useOrders } from '../hooks/useOrders'
import { adaptOrder } from '../lib/adaptOrder'
import type { OrderStatus, PaymentMethod } from '../api/orders'
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

export default function OrderListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const [status, setStatus] = useState<OrderStatus | 'all'>('all')
  const [ff, setFf] = useState<'all' | FfKey>('all')
  const [pay, setPay] = useState<PaymentMethod | 'all'>('all')
  const [search, setSearch] = useState(searchParams.get('q') ?? '')
  const [debouncedSearch, setDebouncedSearch] = useState(search)
  const [page, setPage] = useState(1)

  // Adopt the URL's ?q= (e.g. the top-bar global search navigates to /orders?q=…).
  useEffect(() => {
    const q = searchParams.get('q') ?? ''
    setSearch(q)
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

  const { data, isLoading, isError, refetch } = useOrders({
    page,
    status: status === 'all' ? '' : status,
    fulfillmentType: ff === 'all' ? '' : FF_API[ff],
    paymentMethod: pay === 'all' ? '' : pay,
    q: debouncedSearch.trim(),
  })

  const rows = useMemo(() => (data?.data ?? []).map(adaptOrder), [data])
  const meta = data?.meta
  const bulk = useBulk(rows.map((r) => r.id))

  // Reset to page 1 whenever a filter changes (search resets via its debounce).
  const onFilter = <T,>(setter: (v: T) => void) => (v: T) => {
    setter(v)
    setPage(1)
  }

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_operations'), t('nav_orders')]}
        title={t('nav_orders')}
        sub={meta ? `${meta.total.toLocaleString()} orders` : ''}
      >
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
              <button className="abtn xs" disabled title="Coming soon">
                <Icon name="download" size={13} /> {t('export')}
              </button>
              <button className="abtn xs danger" disabled title="Coming soon">
                <Icon name="refresh" size={13} /> {t('refund')}
              </button>
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
              label="orders"
              onPage={setPage}
            />
          </>
        )}
      </div>
    </div>
  )
}
