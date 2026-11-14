import { useMemo, useState } from 'react'
import { useCan } from '@/stores/auth'
import { useSupplierCatalog } from '../hooks/useSuppliers'
import { filterCatalog } from '../lib/filterCatalog'
import { estimateCost, pickWarnings, qtyHint, type PickContext } from '../lib/supplierPick'

const MAX_ROWS = 50

interface Props {
  /** Supplier id (0/negative = none chosen yet). */
  providerId: number
  value: string
  onChange: (upstreamId: string) => void
  /** Quantity this order will send — shows the supplier cost and checks the limits. */
  qty?: number
  /** Range customers may type on a per-unit product (product editor). */
  amountRange?: PickContext['amountRange']
  /** Parameter names the order sends — flags supplier params it lacks. */
  sends?: string[]
}

/** SupplierProductPicker chooses a supplier's own product id from its live
 *  catalog (name, id, price, quantity limits) instead of free text — a typed
 *  name like "soulchill" was refused upstream as Product Not Found
 *  (2026-11-13). Falls back to typing the id when the admin lacks
 *  suppliers.view or the catalog can't be loaded. */
export function SupplierProductPicker({ providerId, value, onChange, qty, amountRange, sends }: Props) {
  const can = useCan()
  // A supplier without a catalog (e.g. the dev mock) shows the error state and
  // the admin types the id.
  const canBrowse = can('suppliers.view') && providerId > 0
  const catalog = useSupplierCatalog(providerId > 0 ? providerId : null, {
    enabled: canBrowse,
    staleTime: 5 * 60 * 1000,
  })
  const [search, setSearch] = useState('')
  const rows = useMemo(() => catalog.data?.data?.products ?? [], [catalog.data])
  const matches = useMemo(() => filterCatalog(rows, search).slice(0, MAX_ROWS), [rows, search])
  const loaded = canBrowse && catalog.isSuccess ? rows : undefined
  const selected = loaded?.find((p) => p.upstreamId === value.trim())
  const warnings = pickWarnings({ value, catalog: loaded, qty, amountRange, sends })
  const cost = qty != null ? estimateCost(selected, qty) : null

  return (
    <div>
      <input
        className="afield mono"
        aria-label="Supplier product id"
        placeholder="e.g. 364"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
      {canBrowse && (
        <div style={{ marginTop: 8 }}>
          <input
            className="afield"
            aria-label="Search the supplier catalog"
            placeholder="Search this supplier's catalog…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <div
            role="listbox"
            aria-label="Supplier products"
            style={{ maxHeight: 200, overflowY: 'auto', marginTop: 6, border: '1px solid var(--border)', borderRadius: 8 }}
          >
            {catalog.isLoading ? (
              <div className="faint" style={{ padding: 10, fontSize: 13 }}>
                Loading catalog…
              </div>
            ) : catalog.isError ? (
              <div className="faint" style={{ padding: 10, fontSize: 13 }}>
                Couldn't load this supplier's catalog — type the supplier product id above.
              </div>
            ) : matches.length === 0 ? (
              <div className="faint" style={{ padding: 10, fontSize: 13 }}>
                No products
              </div>
            ) : (
              matches.map((p) => {
                const on = value.trim() === p.upstreamId
                const hint = qtyHint(p)
                return (
                  <button
                    type="button"
                    role="option"
                    aria-selected={on}
                    key={p.upstreamId}
                    onClick={() => onChange(p.upstreamId)}
                    style={{
                      all: 'unset',
                      boxSizing: 'border-box',
                      width: '100%',
                      padding: '7px 10px',
                      cursor: 'pointer',
                      fontSize: 13,
                      display: 'flex',
                      gap: 10,
                      justifyContent: 'space-between',
                      background: on ? 'var(--surface-2)' : undefined,
                    }}
                  >
                    <span>
                      {p.name}
                      {p.category && <span className="faint"> · {p.category}</span>}
                    </span>
                    <span className="faint mono" style={{ whiteSpace: 'nowrap' }}>
                      #{p.upstreamId} · {p.price} {p.currency}
                      {hint ? ` · ${hint}` : ''}
                      {p.available ? '' : ' · unavailable'}
                    </span>
                  </button>
                )
              })
            )}
          </div>
        </div>
      )}
      {selected && (
        <div className="ahint" style={{ marginTop: 8 }}>
          Selected: <b>{selected.name}</b> · {selected.price} {selected.currency} each
          {qtyHint(selected) ? ` · ${qtyHint(selected)}` : ''}
          {selected.params.length > 0 ? ` · needs ${selected.params.join(', ')}` : ''}
          {cost != null && qty != null && (
            <>
              {' '}
              · about <b>{cost} {selected.currency}</b> for {qty.toLocaleString('en-US')}
            </>
          )}
        </div>
      )}
      {warnings.map((w) => (
        <div key={w} role="alert" style={{ color: 'var(--warn)', fontSize: 12.5, fontWeight: 600, marginTop: 6 }}>
          {w}
        </div>
      ))}
    </div>
  )
}
