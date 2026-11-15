import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Icon, Modal } from '@/components'
import { ApiError } from '@/lib/api-client'
import { orderNumber } from '@/lib/orderNumber'
import {
  listRerouteProviders,
  rerouteBulkOrders,
  MAX_BULK_RESEND,
  type BulkRerouteOutcome,
} from '../api/orders'

const STATUS_LABEL: Record<BulkRerouteOutcome['status'], { text: string; cls: string }> = {
  completed: { text: 'Delivered', cls: 'st-ok' },
  sent: { text: 'Sent', cls: 'st-ok' },
  refused: { text: 'Refused by supplier', cls: 'st-danger' },
  needs_ack: { text: 'Needs confirmation', cls: 'st-warn' },
  no_link: { text: 'No supplier', cls: 'st-warn' },
  not_eligible: { text: 'Skipped', cls: 'st-mute' },
  conflict: { text: 'Changed — skipped', cls: 'st-mute' },
  error: { text: 'Error', cls: 'st-danger' },
}

/** BulkResendModal re-sends up to MAX_BULK_RESEND selected orders to a supplier
 *  in one go (the old admin's bulk "re-order via API"): each order goes to its
 *  product's first usable linked supplier — or to its link at one chosen
 *  supplier — and the per-order outcome is listed. */
export function BulkResendModal({ ids, onClose, onDone }: { ids: string[]; onClose: () => void; onDone: () => void }) {
  const qc = useQueryClient()
  const { data: provRes } = useQuery({
    queryKey: ['admin', 'orders', 'reroute-providers'],
    queryFn: listRerouteProviders,
    staleTime: 5 * 60 * 1000,
  })
  const providers = provRes?.data ?? []
  const [providerId, setProviderId] = useState(0)
  const [understood, setUnderstood] = useState(false)
  const [ackAtSupplier, setAckAtSupplier] = useState(false)
  const [sendAnyway, setSendAnyway] = useState(false)
  const [error, setError] = useState('')
  const name = (id?: number) => providers.find((p) => p.id === id)?.name ?? (id ? `#${id}` : '')

  const run = useMutation({
    mutationFn: () =>
      rerouteBulkOrders({
        ids,
        ...(providerId ? { providerId } : {}),
        acknowledgeAtSupplier: ackAtSupplier,
        acknowledgeAmbiguous: ackAtSupplier,
        acknowledgeLowBalance: sendAnyway,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'orders'] })
      qc.invalidateQueries({ queryKey: ['admin', 'order'] })
    },
    onError: (e) => setError(e instanceof ApiError ? e.message : 'Could not re-send the orders.'),
  })
  const result = run.data?.data
  const tooMany = ids.length > MAX_BULK_RESEND

  return (
    <Modal onClose={result ? onDone : onClose} maxWidth={680}>
      <div style={{ padding: 22, maxHeight: '86vh', overflowY: 'auto' }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>
          Re-send {ids.length} order{ids.length === 1 ? '' : 's'} to a supplier
        </h3>
        {!result ? (
          <>
            <p className="muted" style={{ fontSize: 13, marginBottom: 12 }}>
              Each order is placed again as a new supplier order, right away. Orders that are completed, refunded or
              not sendable are skipped.
            </p>
            <label className="alabel">Supplier</label>
            <select
              className="afield"
              aria-label="Supplier"
              value={providerId}
              onChange={(e) => setProviderId(Number(e.target.value))}
            >
              <option value={0}>Each order's first linked supplier (Active first, then the backup order)</option>
              {providers.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name} — only orders whose product is linked there
                </option>
              ))}
            </select>
            <div className="ahint" style={{ marginTop: 6 }}>
              {providerId
                ? 'Orders go to their product’s link at this supplier, even if it is the one they were sent to before.'
                : 'Skips the supplier product each order was already sent to.'}
            </div>

            <div
              style={{
                marginTop: 14,
                padding: 12,
                borderRadius: 10,
                border: '1px solid var(--border-strong)',
                background: 'var(--surface-2)',
                fontSize: 13,
              }}
            >
              <b>Double-delivery risk.</b> If a previous supplier still completes one of these orders, the customer may
              receive it twice and both suppliers may charge us.
              <label style={{ display: 'flex', gap: 8, marginTop: 10, alignItems: 'center' }}>
                <input type="checkbox" checked={understood} onChange={(e) => setUnderstood(e.target.checked)} /> I
                understand
              </label>
              <label style={{ display: 'flex', gap: 8, marginTop: 6, alignItems: 'center', color: 'var(--danger)' }}>
                <input type="checkbox" checked={ackAtSupplier} onChange={(e) => setAckAtSupplier(e.target.checked)} />{' '}
                Also re-send orders a supplier may still deliver (otherwise they are skipped)
              </label>
              <label style={{ display: 'flex', gap: 8, marginTop: 6, alignItems: 'center' }}>
                <input type="checkbox" checked={sendAnyway} onChange={(e) => setSendAnyway(e.target.checked)} /> Send
                even if a supplier's balance looks too low (the account allows credit)
              </label>
            </div>
            {tooMany && (
              <div role="alert" style={{ color: 'var(--danger)', fontSize: 13, marginTop: 10 }}>
                Select at most {MAX_BULK_RESEND} orders — each one is a new, billed supplier order.
              </div>
            )}
            {error && (
              <div role="alert" style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginTop: 10 }}>
                {error}
              </div>
            )}
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 18 }}>
              <button className="abtn" onClick={onClose}>
                Cancel
              </button>
              <button
                className="abtn primary"
                disabled={!understood || tooMany || run.isPending}
                onClick={() => {
                  setError('')
                  run.mutate()
                }}
              >
                <Icon name="send" size={14} /> {run.isPending ? 'Sending…' : `Re-send ${ids.length}`}
              </button>
            </div>
          </>
        ) : (
          <>
            <p style={{ fontSize: 13, marginBottom: 10 }}>
              Sent <b>{result.sent}</b> of {result.total}.
            </p>
            <table className="tbl" aria-label="Re-send results" style={{ width: '100%', fontSize: 13 }}>
              <thead>
                <tr>
                  <th>Order</th>
                  <th>Result</th>
                  <th>Details</th>
                </tr>
              </thead>
              <tbody>
                {result.results.map((r) => {
                  const s = STATUS_LABEL[r.status] ?? STATUS_LABEL.error
                  return (
                    <tr key={r.id}>
                      <td className="mono">{orderNumber(r.id)}</td>
                      <td>
                        <span className={`st ${s.cls}`}>{s.text}</span>
                      </td>
                      <td>
                        {r.provider ? (
                          <span>
                            {name(r.provider)} <span className="faint mono">#{r.upstreamId}</span>
                            {r.message ? ' — ' : ''}
                          </span>
                        ) : null}
                        {r.message}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
            <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 18 }}>
              <button className="abtn primary" onClick={onDone}>
                Done
              </button>
            </div>
          </>
        )}
      </div>
    </Modal>
  )
}
