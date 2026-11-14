import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Icon, Modal } from '@/components'
import { ApiError } from '@/lib/api-client'
import { toast } from '@/stores/toast'
import { SupplierProductPicker } from '@/features/suppliers/components/SupplierProductPicker'
import { listRerouteProviders, rerouteOrder, type AdminOrder, type AdminOrderItem } from '../api/orders'
import { rerouteState, supplierQty } from '../lib/reroute'

function fmt(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleString('en-US', { dateStyle: 'medium', timeStyle: 'short' })
}

/** SupplierPanel shows where a supplier-fulfilled order was sent (supplier,
 *  product, supplier ref, stuck flag, earlier attempts) and — for a stuck order
 *  — the "Send to another supplier" action (client msgs #5–#8). For a pending
 *  manual account-credit order it offers "Send via supplier", like the old
 *  admin's "إعادة الطلب API" button (client screenshots, 2026-11-12). */
export function SupplierPanel({ order, canManage }: { order: AdminOrder; canManage: boolean }) {
  const [open, setOpen] = useState(false)
  const { eligible, fromManual } = rerouteState(order)
  const apiLine = order.items.find((it) => it.fulfillmentMode === 'api')
  const show = !!apiLine || (fromManual && canManage)
  const { data: provRes } = useQuery({
    queryKey: ['admin', 'orders', 'reroute-providers'],
    queryFn: listRerouteProviders,
    enabled: show,
    staleTime: 5 * 60 * 1000,
  })
  if (!show) return null
  const providers = provRes?.data ?? []
  const name = (id?: number) => providers.find((p) => p.id === id)?.name ?? (id != null ? `#${id}` : '—')
  const f = order.fulfillment
  const line = apiLine ?? order.items[0]

  if (!apiLine) {
    // Manual order, never sent to a supplier.
    if (providers.length === 0) return null
    return (
      <div className="acard" style={{ marginBottom: 16 }}>
        <div className="panelhead">
          <Icon name="server" size={17} />
          <h3>Supplier</h3>
          <div className="ph-act">
            <button className="abtn xs primary" onClick={() => setOpen(true)}>
              <Icon name="send" size={13} /> Send via supplier
            </button>
          </div>
        </div>
        <div className="muted" style={{ padding: '12px 18px 16px', fontSize: 13 }}>
          This order is fulfilled by hand. You can send it to a connected supplier instead — they deliver it
          automatically and charge their prepaid balance.
        </div>
        {open && <RerouteModal order={order} line={line} providers={providers} onClose={() => setOpen(false)} />}
      </div>
    )
  }

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
          <b>{name(apiLine.fulfillmentProvider)}</b>
        </div>
        <div>
          <div className="faint" style={{ fontSize: 12 }}>Supplier product</div>
          <b className="mono">{apiLine.upstreamProductId || '—'}</b>
        </div>
        <div>
          <div className="faint" style={{ fontSize: 12 }}>Quantity sent</div>
          <b>{supplierQty(apiLine).toLocaleString('en-US')}</b>
        </div>
        <div>
          <div className="faint" style={{ fontSize: 12 }}>Supplier reference</div>
          <b className="mono" style={{ overflowWrap: 'anywhere' }}>
            {f.providerRef || '—'}
          </b>
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
      {open && <RerouteModal order={order} line={line} providers={providers} onClose={() => setOpen(false)} />}
    </div>
  )
}

/** The parameter names an order line sends upstream: its field keys plus
 *  playerId (the panel adapter adds it when the line has a player id). */
function sentParams(line: AdminOrderItem): string[] {
  const keys = (line.fields ?? []).map((f) => f.key)
  if (line.playerId) keys.push('playerId')
  return keys
}

function RerouteModal({
  order,
  line,
  providers,
  onClose,
}: {
  order: AdminOrder
  line: AdminOrderItem
  providers: { id: number; name: string }[]
  onClose: () => void
}) {
  const qc = useQueryClient()
  const { ambiguous, fromManual } = rerouteState(order)
  const current = fromManual ? undefined : line.fulfillmentProvider
  const [providerId, setProviderId] = useState<number>(
    providers.find((p) => p.id !== current)?.id ?? providers[0]?.id ?? 0,
  )
  const [upstream, setUpstream] = useState('')
  const [ack, setAck] = useState(false)
  const [understood, setUnderstood] = useState(false)
  const [error, setError] = useState('')
  const qty = supplierQty(line)

  const send = useMutation({
    mutationFn: () =>
      rerouteOrder(order.id, { providerId, upstreamProductId: upstream.trim(), acknowledgeAmbiguous: ack }),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['admin', 'order', order.id] })
      qc.invalidateQueries({ queryKey: ['admin', 'orders'] })
      const st = res.data?.status
      const stuck = !!res.data?.fulfillment.supplierStuckAt
      if (st === 'completed') toast.success('Delivered by the supplier')
      else if (stuck) toast.error('The supplier refused it — try another supplier or refund')
      else toast.success('Sent to the supplier')
      onClose()
    },
    onError: (e) => setError(e instanceof ApiError ? e.message : 'Could not send the order.'),
  })

  const ready = providerId > 0 && upstream.trim() !== '' && understood && (!ambiguous || ack)

  return (
    <Modal onClose={onClose} maxWidth={640}>
      <div style={{ padding: 22, maxHeight: '84vh', overflowY: 'auto' }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>
          {fromManual ? 'Send via supplier' : 'Send to another supplier'}
        </h3>
        <p className="muted" style={{ fontSize: 13, marginBottom: 14 }}>
          {fromManual
            ? 'The order is sent to the chosen supplier right away and completes automatically when they deliver.'
            : 'The order is sent to the chosen supplier right away. The current supplier is no longer checked; its reference stays in the history.'}
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
          Supplier product
        </label>
        <SupplierProductPicker
          providerId={providerId}
          value={upstream}
          onChange={setUpstream}
          qty={qty}
          sends={sentParams(line)}
        />

        <div style={{ marginTop: 14, fontSize: 13 }}>
          <div className="faint" style={{ fontSize: 12, marginBottom: 4 }}>
            What we'll send
          </div>
          <div>
            Quantity <b>{qty.toLocaleString('en-US')}</b>
            {line.amount != null ? ' (the units the customer asked for)' : ''}
            {line.playerId ? (
              <>
                {' '}
                · player <b className="mono">{line.playerId}</b>
              </>
            ) : null}
          </div>
          {(line.fields ?? []).length > 0 && (
            <div className="faint" style={{ marginTop: 2 }}>
              {(line.fields ?? []).map((f) => `${f.key}=${f.value}`).join(' · ')}
            </div>
          )}
          <div style={{ marginTop: 2 }}>
            Customer paid <b>${order.total.toFixed(2)}</b>
          </div>
        </div>

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
          <b>Double-delivery risk.</b>{' '}
          {fromManual
            ? 'Make sure nobody has already credited this order by hand — otherwise the customer receives it twice and the supplier charges us.'
            : "If the previous supplier still completes this order, the customer may receive it twice and both suppliers may charge us. Check the previous supplier's panel first when you can."}
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
