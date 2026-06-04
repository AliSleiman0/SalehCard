import { useState } from 'react'
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
  ComingSoonNote,
} from '@/components'
import { useBulk } from '@/hooks/useBulk'
import { money } from '@/lib/utils'
import { orders } from '@/lib/mock/demo'
import type { FfKey } from '@/components'

const FF_DOT: Record<FfKey, string> = {
  code: 'var(--ff-code)',
  credit: 'var(--ff-credit)',
  transfer: 'var(--ff-transfer)',
}

export default function OrderListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const q = params.get('q')?.toLowerCase() ?? ''
  const [status, setStatus] = useState('all')
  const [ff, setFf] = useState<'all' | FfKey>('all')

  // TODO: wire to GET /api/admin/orders (currently mock; backend route stubbed 501).
  const rows = orders.filter(
    (o) =>
      (status === 'all' || o.status === status) &&
      (ff === 'all' || o.ff === ff) &&
      (!q || o.id.toLowerCase().includes(q) || o.email.toLowerCase().includes(q) || o.customer.toLowerCase().includes(q))
  )
  const bulk = useBulk(rows.map((r) => r.id))

  return (
    <div className="page page-wide">
      <PageHead crumbs={[t('grp_operations'), t('nav_orders')]} title={t('nav_orders')} sub="842 orders today · 14 shown">
        <button className="abtn">
          <Icon name="download" size={15} /> {t('export')}
        </button>
        <button className="abtn">
          <Icon name="refresh" size={15} /> Refresh
        </button>
      </PageHead>

      <ComingSoonNote mock />

      <div className="acard">
        <div className="toolbar">
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input placeholder="Order ID, customer email, or code…" defaultValue={q} />
          </div>
          <select className="select" value={status} onChange={(e) => setStatus(e.target.value)}>
            <option value="all">All statuses</option>
            <option value="delivered">Delivered</option>
            <option value="processing">Processing</option>
            <option value="refunded">Refunded</option>
            <option value="failed">Failed</option>
          </select>
          <select className="select">
            <option>All payment</option>
            <option>Wallet</option>
            <option>Visa</option>
            <option>USDT</option>
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
              <Chip key={k} on={ff === k} onClick={() => setFf(k as 'all' | FfKey)} dotColor={k !== 'all' ? FF_DOT[k as FfKey] : undefined}>
                {l}
              </Chip>
            ))}
          </div>
          <div className="tb-spacer" />
          <button className="abtn sm">
            <Icon name="filter" size={14} /> Date range
          </button>
        </div>

        {bulk.some && (
          <div className="bulkbar">
            <Checkbox on onClick={bulk.clear} />
            <span>
              {bulk.sel.length} {t('selected')}
            </span>
            <div className="ba-act">
              <button className="abtn xs">
                <Icon name="download" size={13} /> {t('export')}
              </button>
              <button className="abtn xs danger">
                <Icon name="refresh" size={13} /> {t('refund')}
              </button>
            </div>
          </div>
        )}

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
                    <span className="mono strong">{o.id}</span>
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
        <Pagination total={842} pages={5} label="orders" />
      </div>
    </div>
  )
}
