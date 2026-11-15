import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Trans, useTranslation } from 'react-i18next'
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
  setWatch,
  type LinkRow,
} from '../lib/supplierLinks'
import { SupplierSearchModal } from './SupplierSearchModal'
import { relativeTimeInline } from '@/lib/utils'

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
  const { t } = useTranslation()
  const [searching, setSearching] = useState(false)
  const provName = (id: number) => providers.find((p) => p.id === id)?.name ?? t('prod_supplier_n', { id })

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
          {t('prod_links_empty')}
        </div>
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table className="tbl" aria-label={t('prod_links_aria')} style={{ width: '100%', fontSize: 13 }}>
            <thead>
              <tr>
                <th style={{ width: 54 }}>{t('prod_links_col_order')}</th>
                <th>{t('prod_links_col_supplier')}</th>
                <th>{t('prod_links_col_product')}</th>
                <th>
                  {t('prod_cost')}
                  {blockUnits > 1 ? ` / ${blockUnits.toLocaleString('en-US')}` : ''}
                </th>
                <th>{t('prod_margin')}</th>
                <th>{t('status')}</th>
                <th title={t('prod_links_autocheck_tip')}>{t('prod_links_autocheck')}</th>
                <th style={{ textAlign: 'right' }}>{t('actions')}</th>
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
                          aria-label={t('prod_move_up', { name: provName(r.provider) })}
                          disabled={i === 0}
                          onClick={() => onChange(moveLink(rows, i, -1))}
                        >
                          <Icon name="chevup" size={12} />
                        </button>
                        <button
                          type="button"
                          className="abtn xs ghost"
                          aria-label={t('prod_move_down', { name: provName(r.provider) })}
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
                              ? t('prod_q_checking')
                              : q.status === 'ok'
                                ? q.available
                                  ? t('prod_q_available')
                                  : t('prod_q_unavailable')
                                : q.status === 'unknown'
                                  ? t('prod_q_unknown')
                                  : t('prod_q_failed')
                          }
                          style={{ width: 9, height: 9, borderRadius: 5, background: dot, flex: 'none' }}
                        />
                        {r.active && <span className="st st-ok">{t('active')}</span>}
                        {!r.enabled && <span className="st st-mute">{t('prod_link_disabled')}</span>}
                        {q?.status === 'unknown' && <span className="st st-danger">{t('prod_link_not_in_catalog')}</span>}
                        {qtyWarn && <span className="st st-warn">{t('prod_link_qty_range')}</span>}
                        {q?.stale && <span className="st st-mute">{t('prod_link_stale')}</span>}
                        {r.pausedReason === 'loss' && (
                          <span
                            className="st st-danger"
                            title={t('prod_link_paused_tip')}
                          >
                            {t('prod_link_paused_loss')}
                          </span>
                        )}
                      </div>
                      {r.sync?.checkedAt && (
                        <div className="faint" style={{ fontSize: 11.5, marginTop: 2 }}>
                          {t('prod_link_checked', { when: relativeTimeInline(r.sync.checkedAt) })}
                          {r.sync.costChangedAt
                            ? t('prod_link_cost_changed', { when: relativeTimeInline(r.sync.costChangedAt) })
                            : ''}
                        </div>
                      )}
                    </td>
                    <td style={{ whiteSpace: 'nowrap', fontSize: 12.5 }}>
                      <label style={{ display: 'inline-flex', gap: 4, alignItems: 'center', marginRight: 8, cursor: 'pointer' }}>
                        <input
                          type="checkbox"
                          aria-label={t('prod_price_sync_aria', { name: provName(r.provider), id: r.upstreamId })}
                          checked={r.priceSync}
                          onChange={(e) => onChange(setWatch(rows, i, 'priceSync', e.target.checked))}
                        />
                        {t('prod_watch_price')}
                      </label>
                      <label style={{ display: 'inline-flex', gap: 4, alignItems: 'center', cursor: 'pointer' }}>
                        <input
                          type="checkbox"
                          aria-label={t('prod_stock_sync_aria', { name: provName(r.provider), id: r.upstreamId })}
                          checked={r.statusSync}
                          onChange={(e) => onChange(setWatch(rows, i, 'statusSync', e.target.checked))}
                        />
                        {t('prod_stock')}
                      </label>
                    </td>
                    <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                      {!r.active && (
                        <button type="button" className="abtn xs" onClick={() => onChange(makeActive(rows, i))}>
                          {t('prod_make_active')}
                        </button>
                      )}{' '}
                      {!r.active && (
                        <button
                          type="button"
                          className="abtn xs ghost"
                          onClick={() => onChange(setEnabled(rows, i, !r.enabled))}
                        >
                          {r.enabled ? t('prod_link_disable') : t('prod_link_enable')}
                        </button>
                      )}{' '}
                      <button
                        type="button"
                        className="abtn xs ghost"
                        aria-label={t('prod_remove_link_aria', { name: provName(r.provider), id: r.upstreamId })}
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
          {t('prod_quotes_error')}
        </div>
      )}
      <div style={{ display: 'flex', gap: 10, alignItems: 'center', marginTop: 10 }}>
        <button type="button" className="abtn sm" onClick={() => setSearching(true)} disabled={rows.length >= 6}>
          <Icon name="search" size={13} /> {t('prod_find_all')}
        </button>
        <span className="ahint" style={{ margin: 0 }}>
          <Trans i18nKey="prod_links_hint" components={{ b: <b /> }} />
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
