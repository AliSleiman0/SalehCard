import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Icon } from '@/components'
import { quoteSupplierLinks, type LinkQuote } from '../api/products'
import {
  addLink,
  blockCost,
  fmtCost,
  makeActive,
  marginPct,
  moveLink,
  removeLink,
  setEnabled,
  type LinkRow,
} from '../lib/supplierLinks'
import { SupplierSearchModal } from './SupplierSearchModal'

function qtyOK(q: LinkQuote, n: number): boolean {
  if (q.qtyValues && q.qtyValues.length > 0) return q.qtyValues.some((v) => Number(v) === n)
  if (q.qtyMin != null && n < q.qtyMin) return false
  if (q.qtyMax != null && n > q.qtyMax) return false
  return true
}

/** SupplierLinksTable is the product's supplier list (the old admin's "API
 *  link" tab): every supplier product it can be fulfilled by, in priority
 *  order, each with its live cost, margin and an availability dot. One row is
 *  active — orders go there. Rows are added from a cross-supplier search, so
 *  no supplier id is ever typed. */
export function SupplierLinksTable({
  rows,
  onChange,
  providers,
  sellPrice,
  blockUnits,
  amountRange,
  searchHint,
}: {
  rows: LinkRow[]
  onChange: (rows: LinkRow[]) => void
  providers: { id: number; name: string }[]
  /** Selling price of one block (per-unit) or of the first package. */
  sellPrice: number
  /** Units per selling block on a per-unit product (1 for packages). */
  blockUnits: number
  /** The amount range customers may type (per-unit products). */
  amountRange?: { min?: number; max?: number }
  /** Pre-fills the supplier search (the product title). */
  searchHint: string
}) {
  const [searching, setSearching] = useState(false)
  const provName = (id: number) => providers.find((p) => p.id === id)?.name ?? `supplier #${id}`

  const pairs = rows.map((r) => ({ provider: r.provider, upstreamId: r.upstreamId }))
  const quotes = useQuery({
    queryKey: ['admin', 'products', 'supplier-quote', pairs.map((p) => `${p.provider}:${p.upstreamId}`).join(',')],
    queryFn: () => quoteSupplierLinks(pairs),
    enabled: pairs.length > 0,
    staleTime: 60 * 1000,
  })
  const quoteFor = (r: LinkRow) =>
    quotes.data?.data?.quotes.find((q) => q.provider === r.provider && q.upstreamId === r.upstreamId)

  return (
    <div>
      {rows.length === 0 ? (
        <div className="faint" style={{ fontSize: 13, padding: '8px 0' }}>
          No supplier linked yet.
        </div>
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table className="tbl" aria-label="Supplier links" style={{ width: '100%', fontSize: 13 }}>
            <thead>
              <tr>
                <th style={{ width: 54 }}>Order</th>
                <th>Supplier</th>
                <th>Supplier product</th>
                <th>Cost{blockUnits > 1 ? ` / ${blockUnits.toLocaleString('en-US')}` : ''}</th>
                <th>Margin</th>
                <th>Status</th>
                <th style={{ textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r, i) => {
                const q = quoteFor(r)
                const cost = blockCost(q, blockUnits)
                const margin = marginPct(sellPrice, cost)
                const dot =
                  !q || q.status === 'not_checked' || q.status === 'unavailable'
                    ? 'var(--border-strong)'
                    : q.status === 'ok' && q.available
                      ? 'var(--ok)'
                      : 'var(--danger)'
                const qtyWarn =
                  q?.status === 'ok' &&
                  amountRange &&
                  [amountRange.min, amountRange.max].some((v) => v != null && !qtyOK(q, v))
                return (
                  <tr key={r.key} style={{ opacity: r.enabled ? 1 : 0.55 }}>
                    <td>
                      <div style={{ display: 'flex', gap: 2 }}>
                        <button
                          type="button"
                          className="abtn xs ghost"
                          aria-label={`Move ${provName(r.provider)} up`}
                          disabled={i === 0}
                          onClick={() => onChange(moveLink(rows, i, -1))}
                        >
                          <Icon name="chevup" size={12} />
                        </button>
                        <button
                          type="button"
                          className="abtn xs ghost"
                          aria-label={`Move ${provName(r.provider)} down`}
                          disabled={i === rows.length - 1}
                          onClick={() => onChange(moveLink(rows, i, 1))}
                        >
                          <Icon name="chevdown" size={12} />
                        </button>
                      </div>
                    </td>
                    <td style={{ fontWeight: 600 }}>{provName(r.provider)}</td>
                    <td>
                      {q?.name ?? r.name ?? '—'}
                      <div className="faint mono" style={{ fontSize: 12 }}>
                        #{r.upstreamId}
                      </div>
                    </td>
                    <td className="mono">
                      {quotes.isLoading ? '…' : cost != null ? fmtCost(cost, q?.currency) : '—'}
                    </td>
                    <td className="mono" style={{ color: margin != null && margin <= 0 ? 'var(--danger)' : undefined }}>
                      {margin != null ? `${margin.toFixed(1)}%` : '—'}
                    </td>
                    <td>
                      <div style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' }}>
                        <span
                          title={
                            !q
                              ? 'checking…'
                              : q.status === 'ok'
                                ? q.available
                                  ? 'available at the supplier'
                                  : 'unavailable at the supplier'
                                : q.status === 'unknown'
                                  ? "not in the supplier's product list"
                                  : "couldn't check"
                          }
                          style={{ width: 9, height: 9, borderRadius: 5, background: dot, flex: 'none' }}
                        />
                        {r.active && <span className="st st-ok">Active</span>}
                        {!r.enabled && <span className="st st-mute">Disabled</span>}
                        {q?.status === 'unknown' && <span className="st st-danger">Not in catalog</span>}
                        {qtyWarn && <span className="st st-warn">Qty range</span>}
                        {q?.stale && <span className="st st-mute">Old data</span>}
                      </div>
                    </td>
                    <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                      {!r.active && (
                        <button type="button" className="abtn xs" onClick={() => onChange(makeActive(rows, i))}>
                          Make active
                        </button>
                      )}{' '}
                      {!r.active && (
                        <button
                          type="button"
                          className="abtn xs ghost"
                          onClick={() => onChange(setEnabled(rows, i, !r.enabled))}
                        >
                          {r.enabled ? 'Disable' : 'Enable'}
                        </button>
                      )}{' '}
                      <button
                        type="button"
                        className="abtn xs ghost"
                        aria-label={`Remove ${provName(r.provider)} #${r.upstreamId}`}
                        onClick={() => onChange(removeLink(rows, i))}
                      >
                        <Icon name="trash" size={12} />
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
      {quotes.isError && (
        <div className="ahint" style={{ marginTop: 6, color: 'var(--warn)' }}>
          Couldn't load live supplier prices right now.
        </div>
      )}
      <div style={{ display: 'flex', gap: 10, alignItems: 'center', marginTop: 10 }}>
        <button type="button" className="abtn sm" onClick={() => setSearching(true)} disabled={rows.length >= 6}>
          <Icon name="search" size={13} /> Find at all suppliers
        </button>
        <span className="ahint" style={{ margin: 0 }}>
          Orders go to the <b>Active</b> supplier. The order of the rows is the backup order.
        </span>
      </div>
      {searching && (
        <SupplierSearchModal
          initialQuery={searchHint}
          isLinked={(p, u) => rows.some((r) => r.provider === p && r.upstreamId === u)}
          onLink={(h) => onChange(addLink(rows, h.provider, h.upstreamId, h.name))}
          onClose={() => setSearching(false)}
        />
      )}
    </div>
  )
}
