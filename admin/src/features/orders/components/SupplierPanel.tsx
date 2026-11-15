import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Icon, Modal } from '@/components'
import { ApiError } from '@/lib/api-client'
import { toast } from '@/stores/toast'
import { SupplierProductPicker } from '@/features/suppliers/components/SupplierProductPicker'
import { fmtCost } from '@/features/products/lib/supplierLinks'
import {
  getSupplierOptions,
  listRerouteProviders,
  rerouteOrder,
  type AdminOrder,
  type AdminOrderItem,
  type RerouteInput,
  type SupplierOption,
} from '../api/orders'
import { orderMargin, rerouteState, supplierQty } from '../lib/reroute'

function fmt(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleString('en-US', { dateStyle: 'medium', timeStyle: 'short' })
}

/** SupplierPanel shows where a supplier-fulfilled order was sent (supplier,
 *  product, supplier ref, stuck flag, earlier attempts) and the "Send to a
 *  supplier" action: the product's linked suppliers, each with this order's
 *  cost, one click to send (client video 2026-11-15, the old admin's per-order
 *  supplier list). For a pending manual account-credit order it offers "Send
 *  via supplier", like the old admin's "إعادة الطلب API" button. */
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
        {open && <SendViaSupplierModal order={order} line={line} providers={providers} onClose={() => setOpen(false)} />}
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
              <Icon name="send" size={13} /> Send to a supplier
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
              #{f.supplierAttempt + 1}
              {f.supplierAttemptAuto ? ' · automatic fallback' : ` · since ${fmt(f.supplierAttemptAt)}`}
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
                    {a.auto ? (
                      <>
                        {' '}
                        <span className="bdg" title={a.reason}>
                          auto
                        </span>
                        {a.reason && (
                          <div className="faint" style={{ fontSize: 12, overflowWrap: 'anywhere' }}>
                            {a.reason}
                          </div>
                        )}
                      </>
                    ) : (
                      a.reroutedBy ? ` · ${a.reroutedBy}` : ''
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {open && <SendViaSupplierModal order={order} line={line} providers={providers} onClose={() => setOpen(false)} />}
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

/** SendViaSupplierModal lists the product's linked suppliers with this order's
 *  cost at each and a Send button per row; any other supplier product can
 *  still be picked by hand under "Other supplier product…". */
export function SendViaSupplierModal({
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
  const { fromManual } = rerouteState(order)
  const opts = useQuery({
    queryKey: ['admin', 'order', order.id, 'supplier-options'],
    queryFn: () => getSupplierOptions(order.id),
  })
  const data = opts.data?.data
  const ambiguous = data?.ambiguous ?? rerouteState(order).ambiguous
  const needsAck = !!data?.needsAck
  const [understood, setUnderstood] = useState(false)
  const [ackAtSupplier, setAckAtSupplier] = useState(false)
  const [ackAmbiguous, setAckAmbiguous] = useState(false)
  const [sendAnyway, setSendAnyway] = useState(false)
  const [error, setError] = useState('')
  const [lowBalanceAsked, setLowBalanceAsked] = useState(false)
  const [sendingKey, setSendingKey] = useState('')
  const [showOther, setShowOther] = useState(false)
  const anyLowBalance = lowBalanceAsked || (data?.options ?? []).some((o) => o.lowBalance)
  const acksOK = understood && (!ambiguous || ackAmbiguous) && (!needsAck || ackAtSupplier)

  const send = useMutation({
    mutationFn: (target: Pick<RerouteInput, 'linkId' | 'providerId' | 'upstreamProductId'>) =>
      rerouteOrder(order.id, {
        ...target,
        acknowledgeAtSupplier: needsAck && ackAtSupplier,
        acknowledgeAmbiguous: ambiguous && ackAmbiguous,
        acknowledgeLowBalance: sendAnyway,
      }),
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
    onError: (e) => {
      setSendingKey('')
      if (e instanceof ApiError) {
        setError(e.message)
        if (e.code === 'REROUTE_LOW_BALANCE') setLowBalanceAsked(true)
        if (e.code === 'REROUTE_NEEDS_ACK' || e.code === 'REROUTE_CONFLICT') opts.refetch()
      } else {
        setError('Could not send the order.')
      }
    },
  })
  const sendTo = (key: string, target: Pick<RerouteInput, 'linkId' | 'providerId' | 'upstreamProductId'>) => {
    setError('')
    setSendingKey(key)
    send.mutate(target)
  }

  const qty = data?.qty ?? supplierQty(line)
  const options = data?.options ?? []

  return (
    <Modal onClose={onClose} maxWidth={820}>
      <div style={{ padding: 22, maxHeight: '86vh', overflowY: 'auto' }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>
          {fromManual ? 'Send via supplier' : 'Send to a supplier'}
        </h3>
        <p className="muted" style={{ fontSize: 13, marginBottom: 12 }}>
          {fromManual
            ? 'The order is sent to the supplier you pick right away and completes automatically when they deliver.'
            : "The order is sent to the supplier you pick right away, as a new order there. The current supplier is no longer checked; its reference stays in the history."}
        </p>

        <div style={{ fontSize: 13, marginBottom: 12 }}>
          Quantity <b>{qty.toLocaleString('en-US')}</b>
          {line.amount != null ? ' (the units the customer asked for)' : ''}
          {line.playerId ? (
            <>
              {' '}
              · player <b className="mono">{line.playerId}</b>
            </>
          ) : null}
          {' · '}customer paid <b>${(data?.linePrice ?? order.total).toFixed(2)}</b>
          {(line.fields ?? []).length > 0 && (
            <div className="faint" style={{ marginTop: 2 }}>
              {(line.fields ?? []).map((f) => `${f.key}=${f.value}`).join(' · ')}
            </div>
          )}
        </div>

        {opts.isLoading ? (
          <div className="faint" style={{ fontSize: 13 }}>
            Checking the product's suppliers…
          </div>
        ) : opts.isError ? (
          <div role="alert" style={{ color: 'var(--danger)', fontSize: 13 }}>
            Couldn't load this order's suppliers.
          </div>
        ) : data && !data.canSend ? (
          <div role="alert" style={{ color: 'var(--danger)', fontSize: 13 }}>
            {data.message || "This order can't be sent to a supplier."}
          </div>
        ) : options.length === 0 ? (
          <div className="faint" style={{ fontSize: 13 }}>
            {data?.productMissing
              ? 'The product was deleted, so there is no supplier list — pick a supplier product below.'
              : 'No supplier is linked to this product yet — link them in the product editor, or pick a supplier product below.'}
          </div>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table className="tbl" aria-label="Linked suppliers" style={{ width: '100%', fontSize: 13 }}>
              <thead>
                <tr>
                  <th>Supplier</th>
                  <th>Supplier product</th>
                  <th>Cost of this order</th>
                  <th>Margin</th>
                  <th>Status</th>
                  <th style={{ textAlign: 'right' }} />
                </tr>
              </thead>
              <tbody>
                {options.map((o) => (
                  <OptionRow
                    key={o.linkId}
                    o={o}
                    linePrice={data?.linePrice ?? 0}
                    currency={data?.currency ?? 'USD'}
                    disabled={!acksOK || send.isPending || (!!o.lowBalance && !sendAnyway)}
                    sending={send.isPending && sendingKey === o.linkId}
                    onSend={() => sendTo(o.linkId, { linkId: o.linkId })}
                  />
                ))}
              </tbody>
            </table>
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
          <b>Double-delivery risk.</b>{' '}
          {fromManual
            ? 'Make sure nobody has already credited this order by hand — otherwise the customer receives it twice and the supplier charges us.'
            : "If the previous supplier still completes this order, the customer may receive it twice and both suppliers may charge us. Check the previous supplier's panel first when you can."}
          <label style={{ display: 'flex', gap: 8, marginTop: 10, alignItems: 'center' }}>
            <input type="checkbox" checked={understood} onChange={(e) => setUnderstood(e.target.checked)} /> I
            understand
          </label>
          {needsAck && (
            <label style={{ display: 'flex', gap: 8, marginTop: 6, alignItems: 'center', color: 'var(--danger)' }}>
              <input type="checkbox" checked={ackAtSupplier} onChange={(e) => setAckAtSupplier(e.target.checked)} /> The
              current supplier may still deliver this order — send it again anyway
            </label>
          )}
          {ambiguous && (
            <label style={{ display: 'flex', gap: 8, marginTop: 6, alignItems: 'center', color: 'var(--danger)' }}>
              <input type="checkbox" checked={ackAmbiguous} onChange={(e) => setAckAmbiguous(e.target.checked)} /> The
              previous supplier's result is unknown — send anyway
            </label>
          )}
          {anyLowBalance && (
            <label style={{ display: 'flex', gap: 8, marginTop: 6, alignItems: 'center' }}>
              <input type="checkbox" checked={sendAnyway} onChange={(e) => setSendAnyway(e.target.checked)} /> Send even
              if a supplier's balance looks too low (the account allows credit)
            </label>
          )}
        </div>

        {error && (
          <div role="alert" style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginTop: 12 }}>
            {error}
          </div>
        )}

        {(!data || data.canSend) && (
          <div style={{ marginTop: 14 }}>
            <button
              type="button"
              className="abtn xs ghost"
              aria-expanded={showOther}
              onClick={() => setShowOther((v) => !v)}
            >
              <Icon name={showOther ? 'chevup' : 'chevdown'} size={12} /> Other supplier product…
            </button>
            {showOther && (
              <OtherSupplierProduct
                line={line}
                qty={qty}
                providers={providers}
                current={fromManual ? undefined : line.fulfillmentProvider}
                disabled={!acksOK || send.isPending || (lowBalanceAsked && !sendAnyway)}
                sending={send.isPending && sendingKey === 'other'}
                onSend={(providerId, upstreamProductId) => sendTo('other', { providerId, upstreamProductId })}
                onEdit={() => {
                  setError('')
                  setLowBalanceAsked(false)
                }}
              />
            )}
          </div>
        )}

        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 18 }}>
          <button className="abtn" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </Modal>
  )
}

function OptionRow({
  o,
  linePrice,
  currency,
  disabled,
  sending,
  onSend,
}: {
  o: SupplierOption
  linePrice: number
  currency: string
  disabled: boolean
  sending: boolean
  onSend: () => void
}) {
  const margin = orderMargin(linePrice, o.cost, o.currency, currency)
  const dot =
    o.catalogStatus === 'ok' ? (o.available ? 'var(--ok)' : 'var(--danger)') : o.catalogStatus === 'unknown' ? 'var(--danger)' : 'var(--border-strong)'
  const label = o.providerName || `supplier #${o.provider}`
  return (
    <tr style={{ opacity: o.blocked ? 0.6 : 1 }}>
      <td style={{ fontWeight: 600 }}>
        {label}
        {o.balance != null && (
          <div className="faint mono" style={{ fontSize: 12, fontWeight: 400 }}>
            balance {fmtCost(o.balance, o.balanceCurrency)}
          </div>
        )}
      </td>
      <td>
        {o.name ?? '—'}
        <div className="faint mono" style={{ fontSize: 12 }}>
          #{o.upstreamId || '—'}
        </div>
      </td>
      <td className="mono">{o.cost != null ? fmtCost(o.cost, o.currency) : '—'}</td>
      <td className="mono" style={{ color: margin != null && margin <= 0 ? 'var(--danger)' : undefined }}>
        {margin != null ? `${margin.toFixed(1)}%` : '—'}
      </td>
      <td>
        <div style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' }}>
          <span style={{ width: 9, height: 9, borderRadius: 5, background: dot, flex: 'none' }} />
          {o.active && <span className="st st-ok">Active</span>}
          {o.current && <span className="st st-mute">Current</span>}
          {o.tried && <span className="st st-mute">Tried</span>}
          {o.lowBalance && <span className="st st-warn">Low balance</span>}
          {o.stale && <span className="st st-mute">Old data</span>}
        </div>
        {o.blocked && (
          <div className="faint" style={{ fontSize: 12, marginTop: 2 }}>
            {o.blocked}
          </div>
        )}
      </td>
      <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
        <button
          type="button"
          className="abtn xs primary"
          aria-label={`Send to ${label}`}
          disabled={disabled || !!o.blocked}
          onClick={onSend}
        >
          <Icon name="send" size={12} /> {sending ? 'Sending…' : o.current ? 'Send again' : 'Send'}
        </button>
      </td>
    </tr>
  )
}

/** The hand-picked route: any connected supplier + a product from its catalog
 *  (the PR #158 picker — switching supplier clears the product). */
function OtherSupplierProduct({
  line,
  qty,
  providers,
  current,
  disabled,
  sending,
  onSend,
  onEdit,
}: {
  line: AdminOrderItem
  qty: number
  providers: { id: number; name: string }[]
  current?: number
  disabled: boolean
  sending: boolean
  onSend: (providerId: number, upstream: string) => void
  onEdit: () => void
}) {
  const [providerId, setProviderId] = useState<number>(
    providers.find((p) => p.id !== current)?.id ?? providers[0]?.id ?? 0,
  )
  const [upstream, setUpstream] = useState('')
  // A supplier product id belongs to ONE supplier: switching supplier must
  // clear it, or the old id goes to the new supplier (prod 2026-11-14:
  // order #D74816BC went to speedcard with jentel's id).
  const changeProvider = (id: number) => {
    setProviderId(id)
    setUpstream('')
    onEdit()
  }
  return (
    <div style={{ marginTop: 10, padding: 12, border: '1px solid var(--border)', borderRadius: 10 }}>
      <label className="alabel">Supplier</label>
      <select className="afield" aria-label="Supplier" value={providerId} onChange={(e) => changeProvider(Number(e.target.value))}>
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
        onChange={(v) => {
          setUpstream(v)
          onEdit()
        }}
        qty={qty}
        sends={sentParams(line)}
      />
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 12 }}>
        <button
          type="button"
          className="abtn primary"
          disabled={disabled || providerId <= 0 || upstream.trim() === ''}
          onClick={() => onSend(providerId, upstream.trim())}
        >
          <Icon name="send" size={14} /> {sending ? 'Sending…' : 'Send now'}
        </button>
      </div>
    </div>
  )
}
