import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Icon, Modal } from '@/components'
import { ApiError } from '@/lib/api-client'
import { searchSuppliers, type SupplierSearchHit } from '../api/products'
import { fmtCost } from '../lib/supplierLinks'

function qtyText(h: SupplierSearchHit): string {
  if (h.qtyValues && h.qtyValues.length > 0) return `qty ${h.qtyValues.join(' / ')}`
  const n = (v: number) => v.toLocaleString('en-US')
  if (h.qtyMin != null && h.qtyMax != null) return `qty ${n(h.qtyMin)}–${n(h.qtyMax)}`
  if (h.qtyMin != null) return `qty ≥ ${n(h.qtyMin)}`
  if (h.qtyMax != null) return `qty ≤ ${n(h.qtyMax)}`
  return ''
}

/** SupplierSearchModal searches one product name across every connected
 *  supplier (the old admin's "Multi-link"), grouped by supplier, so a supplier
 *  product is linked with one click — never by typing its id. */
export function SupplierSearchModal({
  initialQuery,
  isLinked,
  onLink,
  onClose,
}: {
  initialQuery: string
  isLinked: (provider: number, upstreamId: string) => boolean
  onLink: (hit: SupplierSearchHit) => void
  onClose: () => void
}) {
  const [text, setText] = useState(initialQuery)
  const [q, setQ] = useState(initialQuery.trim())
  // Debounce typing so each keystroke doesn't search every supplier.
  useEffect(() => {
    const t = setTimeout(() => setQ(text.trim()), 350)
    return () => clearTimeout(t)
  }, [text])

  const search = useQuery({
    queryKey: ['admin', 'products', 'supplier-search', q],
    queryFn: () => searchSuppliers(q),
    enabled: q.length >= 2,
    staleTime: 60 * 1000,
  })
  const res = search.data?.data
  const groups = new Map<string, SupplierSearchHit[]>()
  for (const h of res?.results ?? []) {
    const k = `${h.providerName} (#${h.provider})`
    groups.set(k, [...(groups.get(k) ?? []), h])
  }

  return (
    <Modal onClose={onClose} maxWidth={720}>
      <div style={{ padding: 22, maxHeight: '84vh', overflowY: 'auto' }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>Find at all suppliers</h3>
        <p className="muted" style={{ fontSize: 13, marginBottom: 12 }}>
          Searches every connected supplier's product list. Link the same product at several suppliers — the
          order of the links is the order they're tried in.
        </p>
        <input
          className="afield"
          aria-label="Search all suppliers"
          placeholder="Product name or supplier id…"
          value={text}
          autoFocus
          onChange={(e) => setText(e.target.value)}
        />

        {q.length < 2 ? (
          <div className="faint" style={{ marginTop: 14, fontSize: 13 }}>
            Type at least 2 characters.
          </div>
        ) : search.isLoading ? (
          <div className="faint" style={{ marginTop: 14, fontSize: 13 }}>
            Searching suppliers…
          </div>
        ) : search.isError ? (
          <div role="alert" style={{ marginTop: 14, fontSize: 13, color: 'var(--danger)' }}>
            {search.error instanceof ApiError ? search.error.message : "Couldn't search the suppliers."}
          </div>
        ) : (
          <>
            {(res?.unavailable ?? []).map((m) => (
              <div key={m.provider} className="ahint" style={{ marginTop: 10, color: 'var(--warn)' }}>
                {m.providerName}:{' '}
                {m.reason === 'unavailable' ? "couldn't load its product list right now" : "can't list its products"}
              </div>
            ))}
            {groups.size === 0 ? (
              <div className="faint" style={{ marginTop: 14, fontSize: 13 }}>
                No supplier sells a product matching “{q}”.
              </div>
            ) : (
              [...groups.entries()].map(([name, hits]) => (
                <div key={name} style={{ marginTop: 16 }}>
                  <div style={{ fontWeight: 700, fontSize: 13, marginBottom: 6 }}>{name}</div>
                  <div style={{ border: '1px solid var(--border)', borderRadius: 8 }}>
                    {hits.map((h) => {
                      const linked = isLinked(h.provider, h.upstreamId)
                      return (
                        <div
                          key={h.upstreamId}
                          style={{
                            display: 'flex',
                            gap: 10,
                            alignItems: 'center',
                            padding: '7px 10px',
                            fontSize: 13,
                            borderTop: '1px solid var(--border)',
                          }}
                        >
                          <span
                            title={h.available ? 'available at the supplier' : 'unavailable at the supplier'}
                            style={{
                              width: 8,
                              height: 8,
                              borderRadius: 4,
                              flex: 'none',
                              background: h.available ? 'var(--ok)' : 'var(--danger)',
                            }}
                          />
                          <span style={{ flex: 1, minWidth: 0 }}>
                            {h.name}
                            {h.category && <span className="faint"> · {h.category}</span>}
                            <div className="faint mono" style={{ fontSize: 12 }}>
                              #{h.upstreamId} · {fmtCost(h.price, h.currency)} each
                              {qtyText(h) ? ` · ${qtyText(h)}` : ''}
                            </div>
                          </span>
                          <button
                            type="button"
                            className="abtn xs"
                            disabled={linked}
                            aria-label={`Link ${h.name} at ${h.providerName}`}
                            onClick={() => onLink(h)}
                          >
                            {linked ? (
                              'Linked'
                            ) : (
                              <>
                                <Icon name="plus" size={12} /> Link
                              </>
                            )}
                          </button>
                        </div>
                      )
                    })}
                  </div>
                </div>
              ))
            )}
          </>
        )}

        <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 18 }}>
          <button className="abtn" onClick={onClose}>
            Done
          </button>
        </div>
      </div>
    </Modal>
  )
}
