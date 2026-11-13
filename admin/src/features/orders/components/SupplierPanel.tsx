import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Icon, Modal } from '@/components'
import { ApiError } from '@/lib/api-client'
import { toast } from '@/stores/toast'
import { useCan } from '@/stores/auth'
import { getSupplierCatalog } from '@/features/suppliers/api/suppliers'
import { listRerouteProviders, rerouteOrder, type AdminOrder } from '../api/orders'
import { rerouteState } from '../lib/reroute'

function fmt(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleString('en-US', { dateStyle: 'medium', timeStyle: 'short' })
}

/** SupplierPanel shows where a supplier-fulfilled order was sent (supplier,
 *  product, supplier ref, stuck flag, earlier attempts) and — for a stuck order
 *  — the "Send to another supplier" action (client msgs #5–#8). */
export function SupplierPanel({ order, canManage }: { order: AdminOrder; canManage: boolean }) {
  const [open, setOpen] = useState(false)
  const line = order.items.find((it) => it.fulfillmentMode === 'api')
  const { data: provRes } = useQuery({
    queryKey: ['admin', 'orders', 'reroute-providers'],
    queryFn: listRerouteProviders,
    enabled: !!line,
    staleTime: 5 * 60 * 1000,
  })
  if (!line) return null
  const providers = provRes?.data ?? []
  const name = (id?: number) => providers.find((p) => p.id === id)?.name ?? (id != null ? `#${id}` : '—')
  const f = order.fulfillment
  const { eligible } = rerouteState(order)

  return (
    <div className="acard" style={{ marginBottom: 16 }}>
      <div className="panelhead">
        <Icon name="server" size={17} />
        <h3>Supplier</h3>
        <div className="ph-act" style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          {f.supplierStuckAt && (
            <span className="st st-danger">
              <i className="d" />
              Stuck since {fmt(f.supplierStuckAt)}
            </span>
          )}
          {canManage && eligible && providers.length > 0 && (
            <button className="abtn xs primary" onClick={() => setOpen(true)}>
              <Icon name="send" size={13} /> Send to another supplier
            </button>
          )}
        </div>
      </div>
      <div style={{ padding: 18, display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: 12 }}>
        <div>
          <div className="faint" style={{ fontSize: 12 }}>Supplier</div>
          <b>{name(line.fulfillmentProvider)}</b>
        </div>
        <div>
          <div className="faint" style={{ fontSize: 12 }}>Supplier product</div>
          <b className="mono">{line.upstreamProductId || '—'}</b>
        </div>
        <div>
          <div className="faint" style={{ fontSize: 12 }}>Supplier reference</div>
          <b className="mono">{f.providerRef || '—'}</b>
        </div>
        {!!f.supplierAttempt && (
          <div>
            <div className="faint" style={{ fontSize: 12 }}>Attempt</div>
            <b>
              #{f.supplierAttempt + 1} · since {fmt(f.supplierAttemptAt)}
            </b>
          </div>
        )}
      </div>
      {f.supplierAttempts && f.supplierAttempts.length > 0 && (
        <div style={{ padding: '0 18px 16px' }}>
          <div className="faint" style={{ fontSize: 12, marginBottom: 6 }}>
            Earlier attempts — if one of these suppliers still delivers, reconcile it manually.
          </div>
          <table className="tbl">
            <thead>
              <tr>
                <th>Supplier</th>
                <th>Product</th>
                <th>Reference</th>
                <th>Moved on</th>
              </tr>
            </thead>
            <tbody>
              {f.supplierAttempts.map((a, i) => (
                <tr key={i}>
                  <td>{name(a.provider)}</td>
                  <td className="mono">{a.upstreamProductId || '—'}</td>
                  <td className="mono">{a.providerRef || '—'}</td>
                  <td>
                    {fmt(a.reroutedAt)}
                    {a.reroutedBy ? ` · ${a.reroutedBy}` : ''}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {open && <RerouteModal order={order} providers={providers} onClose={() => setOpen(false)} />}
    </div>
  )
}

function RerouteModal({
  order,
  providers,
  onClose,
}: {
  order: AdminOrder
  providers: { id: number; name: string }[]
  onClose: () => void
}) {
  const qc = useQueryClient()
  const can = useCan()
  const { ambiguous } = rerouteState(order)
  const current = order.items.find((it) => it.fulfillmentMode === 'api')?.fulfillmentProvider
  const [providerId, setProviderId] = useState<number>(
    providers.find((p) => p.id !== current)?.id ?? providers[0]?.id ?? 0,
  )
  const [upstream, setUpstream] = useState('')
  const [search, setSearch] = useState('')
  const [ack, setAck] = useState(false)
  const [understood, setUnderstood] = useState(false)
  const [error, setError] = useState('')

  // The supplier catalog lives under the suppliers domain; admins without it
  // type the supplier product id instead.
  const canBrowse = can('suppliers.view')
  const catalog = useQuery({
    queryKey: ['admin', 'suppliers', providerId, 'catalog'],
    queryFn: () => getSupplierCatalog(providerId),
    enabled: canBrowse && providerId > 0,
    staleTime: 5 * 60 * 1000,
  })
  const matches = useMemo(() => {
    const q = search.trim().toLowerCase()
    const all = catalog.data?.data?.products ?? []
    return (q ? all.filter((p) => p.name.toLowerCase().includes(q) || p.upstreamId.includes(q)) : all).slice(0, 40)
  }, [catalog.data, search])

  const send = useMutation({
    mutationFn: () =>
      rerouteOrder(order.id, { providerId, upstreamProductId: upstream.trim(), acknowledgeAmbiguous: ack }),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['admin', 'order', order.id] })
      qc.invalidateQueries({ queryKey: ['admin', 'orders'] })
      const st = res.data?.status
      toast.success(st === 'completed' ? 'Delivered by the new supplier' : 'Sent to the new supplier')
      onClose()
    },
    onError: (e) => setError(e instanceof ApiError ? e.message : 'Could not send the order.'),
  })

  const ready = providerId > 0 && upstream.trim() !== '' && understood && (!ambiguous || ack)

  return (
    <Modal onClose={onClose} maxWidth={620}>
      <div style={{ padding: 22, maxHeight: '84vh', overflowY: 'auto' }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>Send to another supplier</h3>
        <p className="muted" style={{ fontSize: 13, marginBottom: 14 }}>
          The order is sent to the chosen supplier right away. The current supplier is no longer checked; its reference
          stays in the history.
        </p>

        <label className="alabel">Supplier</label>
        <select className="afield" value={providerId} onChange={(e) => setProviderId(Number(e.target.value))}>
          {providers.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
              {p.id === current ? ' (current)' : ''}
            </option>
          ))}
        </select>

        <label className="alabel" style={{ marginTop: 12 }}>
          Supplier product id
        </label>
        <input
          className="afield mono"
          placeholder="e.g. 364"
          value={upstream}
          onChange={(e) => setUpstream(e.target.value)}
        />
        {canBrowse && (
          <div style={{ marginTop: 8 }}>
            <input
              className="afield"
              placeholder="Search this supplier's catalog…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
            <div style={{ maxHeight: 180, overflowY: 'auto', marginTop: 6, border: '1px solid var(--border)', borderRadius: 8 }}>
              {catalog.isLoading ? (
                <div className="faint" style={{ padding: 10, fontSize: 13 }}>
                  Loading catalog…
                </div>
              ) : matches.length === 0 ? (
                <div className="faint" style={{ padding: 10, fontSize: 13 }}>
                  No products
                </div>
              ) : (
                matches.map((p) => (
                  <div
                    key={p.upstreamId}
                    onClick={() => setUpstream(p.upstreamId)}
                    style={{
                      padding: '7px 10px',
                      cursor: 'pointer',
                      fontSize: 13,
                      display: 'flex',
                      justifyContent: 'space-between',
                      background: upstream === p.upstreamId ? 'var(--surface-2)' : undefined,
                    }}
                  >
                    <span>{p.name}</span>
                    <span className="faint mono">
                      {p.upstreamId} · {p.price} {p.currency}
                      {p.available ? '' : ' · unavailable'}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>
        )}

        <div
          style={{
            marginTop: 16,
            padding: 12,
            borderRadius: 10,
            border: '1px solid var(--border-strong)',
            background: 'var(--surface-2)',
            fontSize: 13,
          }}
        >
          <b>Double-delivery risk.</b> If the previous supplier still completes this order, the customer may receive it
          twice and both suppliers may charge us. Check the previous supplier's panel first when you can.
          <label style={{ display: 'flex', gap: 8, marginTop: 10, alignItems: 'center' }}>
            <input type="checkbox" checked={understood} onChange={(e) => setUnderstood(e.target.checked)} /> I
            understand
          </label>
          {ambiguous && (
            <label style={{ display: 'flex', gap: 8, marginTop: 6, alignItems: 'center', color: 'var(--danger)' }}>
              <input type="checkbox" checked={ack} onChange={(e) => setAck(e.target.checked)} /> The previous
              supplier's result is unknown — send anyway
            </label>
          )}
        </div>

        {error && <div style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginTop: 12 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 18 }}>
          <button className="abtn" onClick={onClose}>
            Cancel
          </button>
          <button className="abtn primary" disabled={!ready || send.isPending} onClick={() => send.mutate()}>
            <Icon name="send" size={14} /> {send.isPending ? 'Sending…' : 'Send now'}
          </button>
        </div>
      </div>
    </Modal>
  )
}
