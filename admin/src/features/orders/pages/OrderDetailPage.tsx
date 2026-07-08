import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Icon,
  PageHead,
  Art,
  Avatar,
  FfBadge,
  StatusBadge,
  PayChip,
  Modal,
  LoadingSpinner,
  ErrorState,
} from '@/components'
import { money } from '@/lib/utils'
import { ApiError } from '@/lib/api-client'
import { useCan } from '@/stores/auth'
import { useOrder } from '../hooks/useOrders'
import { useRefundOrder, useCompleteOrder, useFailOrder } from '../hooks/useOrderMutations'
import { adaptOrder } from '../lib/adaptOrder'
import type { AdminOrder } from '../api/orders'

export default function OrderDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const can = useCan()
  const canManage = can('orders.manage')
  const { data, isLoading, isError, refetch } = useOrder(id)
  const [masked, setMasked] = useState(true)
  const [refundOpen, setRefundOpen] = useState(false)
  const [completeOpen, setCompleteOpen] = useState(false)
  const [failOpen, setFailOpen] = useState(false)

  if (isLoading) return <div className="page"><LoadingSpinner /></div>
  if (isError || !data?.data) {
    return (
      <div className="page">
        <ErrorState message="Couldn't load this order." onRetry={() => refetch()} />
      </div>
    )
  }

  const o = data.data
  const v = adaptOrder(o)
  const item = o.items[0]
  // Mirrors the server's transition rules: refund allows processing|completed,
  // manual completion allows processing only, and the pending→failed cleanup
  // (money-neutral) allows pending only.
  const refundable = canManage && (o.status === 'processing' || o.status === 'completed')
  const completable = canManage && o.status === 'processing'
  const failable = canManage && o.status === 'pending'
  const refunded = o.status === 'refunded'

  return (
    <div className="page">
      <PageHead
        crumbs={[t('nav_orders'), v.id.slice(-8)]}
        title={
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 12 }}>
            Order {v.id.slice(-8)} <StatusBadge s={v.status} />
          </span>
        }
        sub={`Placed ${v.date} · ${v.customer}`}
      >
        <button className="abtn" onClick={() => navigate('/orders')}>
          <Icon name="chevleft" size={15} /> {t('back')}
        </button>
        {refundable && (
          <button className="abtn danger" onClick={() => setRefundOpen(true)}>
            <Icon name="refresh" size={15} /> {t('refund')}
          </button>
        )}
      </PageHead>

      <div className="formgrid">
        <div className="fieldset">
          <div className="acard">
            <div className="panelhead">
              <Icon name="bag" size={17} />
              <h3>Items</h3>
              <div className="ph-act">
                <FfBadge ff={v.ff} />
              </div>
            </div>
            <div style={{ padding: 18, display: 'flex', flexDirection: 'column', gap: 12 }}>
              {o.items.map((it, i) => (
                <div key={i} style={{ display: 'flex', gap: 14, alignItems: 'center' }}>
                  <Art art={v.art} size={56} radius={12} />
                  <div style={{ flex: 1 }}>
                    <div style={{ fontWeight: 800, fontSize: 15 }}>{it.title.en}</div>
                    <div className="faint" style={{ fontSize: 12.5 }}>
                      Qty {it.qty} · {o.currency}
                      {it.denomination ? ` · ${it.denomination}` : ''}
                    </div>
                    {it.fields && it.fields.length > 0 && (
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 14px', marginTop: 6 }}>
                        {it.fields.map((f) => (
                          <span key={f.key} style={{ fontSize: 12.5 }}>
                            <span className="faint">{f.label?.en || f.key}: </span>
                            <b style={{ fontWeight: 700 }}>{f.value}</b>
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                  <div className="num strong" style={{ fontSize: 16 }}>
                    {money(it.price * it.qty, v.cur)}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {v.ff === 'code' && (
            <div className="acard" style={{ borderColor: 'var(--ff-code-bd)' }}>
              <div className="panelhead">
                <span className="bdg ff-code">
                  <i className="d" />
                  Code delivery
                </span>
                <div className="ph-act">
                  {refunded ? (
                    <span className="st st-danger">
                      <i className="d" />
                      Refunded
                    </span>
                  ) : (
                    <span className={o.fulfillment.deliveredCode ? 'st st-ok' : 'st st-warn'}>
                      <i className="d" />
                      {o.fulfillment.deliveredCode ? 'Delivered' : 'Pending'}
                    </span>
                  )}
                </div>
              </div>
              <div style={{ padding: 18 }}>
                {refunded && <RefundedNotice order={o} />}
                <label className="alabel">Delivered code (admin view)</label>
                <div className={'avault' + (masked ? ' masked' : '')}>
                  <span className="vc">{o.fulfillment.deliveredCode || '— not yet delivered —'}</span>
                  {o.fulfillment.deliveredCode && (
                    <div style={{ display: 'flex', gap: 6 }}>
                      <button className="abtn xs" onClick={() => setMasked(!masked)}>
                        <Icon name={masked ? 'eye' : 'eyeoff'} size={13} /> {masked ? 'Reveal' : 'Hide'}
                      </button>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}

          {v.ff === 'credit' && (
            <div className="acard" style={{ borderColor: 'var(--ff-credit-bd)' }}>
              <div className="panelhead">
                <span className="bdg ff-credit">
                  <i className="d" />
                  Account credit
                </span>
                <div className="ph-act">
                  {refunded ? (
                    <span className="st st-danger">
                      <i className="d" />
                      Refunded
                    </span>
                  ) : (
                    <span className={o.status === 'completed' ? 'st st-ok' : 'st st-warn'}>
                      <i className="d" />
                      {o.status === 'completed' ? 'Credited' : 'Processing'}
                    </span>
                  )}
                </div>
              </div>
              <div style={{ padding: 18 }}>
                {refunded && <RefundedNotice order={o} />}
                <div className="deflist">
                  <div className="defrow">
                    <span className="dk">Player / Account ID</span>
                    <span className="dv mono">{item?.playerId || o.fulfillment.creditedToId || '—'}</span>
                  </div>
                  <div className="defrow">
                    <span className="dk">Amount</span>
                    <span className="dv">{money(o.total, v.cur)}</span>
                  </div>
                </div>
                {completable && (
                  <div style={{ marginTop: 16, paddingTop: 16, borderTop: '1px solid var(--border)' }}>
                    <label className="alabel">Manually complete this credit</label>
                    <button className="abtn sm ok" onClick={() => setCompleteOpen(true)}>
                      <Icon name="check" size={14} /> Mark credit as completed
                    </button>
                  </div>
                )}
              </div>
            </div>
          )}

          {v.ff === 'transfer' && (
            <div className="acard" style={{ borderColor: 'var(--ff-transfer-bd)' }}>
              <div className="panelhead">
                <span className="bdg ff-transfer">
                  <i className="d" />
                  Money transfer
                </span>
                <div className="ph-act">
                  <StatusBadge s={v.status} />
                </div>
              </div>
              <div style={{ padding: 18 }}>
                {refunded && <RefundedNotice order={o} />}
                <div className="deflist">
                  <div className="defrow">
                    <span className="dk">Reference</span>
                    <span className="dv mono">{o.fulfillment.transferRef || '—'}</span>
                  </div>
                  <div className="defrow">
                    <span className="dk">Recipient</span>
                    <span className="dv">{item?.recipient?.name || '—'}</span>
                  </div>
                  <div className="defrow">
                    <span className="dk">Country</span>
                    <span className="dv">{item?.recipient?.country || '—'}</span>
                  </div>
                </div>
                {completable && (
                  <div style={{ marginTop: 16, paddingTop: 16, borderTop: '1px solid var(--border)' }}>
                    <label className="alabel">Manually update transfer status</label>
                    <button className="abtn sm ok" onClick={() => setCompleteOpen(true)}>
                      <Icon name="check" size={14} /> Mark transfer as completed
                    </button>
                  </div>
                )}
              </div>
            </div>
          )}

          <div className="acard">
            <div className="panelhead">
              <Icon name="activity" size={17} />
              <h3>Order activity</h3>
            </div>
            <div style={{ padding: 18 }}>
              <div className="timeline">
                {(o.fulfillment.statusTimeline?.length
                  ? o.fulfillment.statusTimeline.map((e) => [e.status, e.note || v.date, 'done'] as [string, string, string])
                  : ([
                      ['Order placed', v.date, 'done'],
                      ['Payment captured', v.date, 'done'],
                      [
                        o.status === 'refunded' ? 'Refund issued' : o.status === 'completed' ? 'Fulfilled' : 'In progress',
                        v.date,
                        o.status === 'failed' ? 'todo' : 'done',
                      ],
                    ] as [string, string, string][])
                ).map(([title, sub, st], i) => (
                  <div className={'tlitem ' + st} key={i}>
                    <div className="tldot">
                      <Icon name={st === 'todo' ? 'clock' : 'check'} size={14} />
                    </div>
                    <div className="tlbody">
                      <b>{title}</b>
                      <span>{sub}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        <div className="fieldset">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Customer</h3>
            <div style={{ display: 'flex', gap: 11, alignItems: 'center', marginBottom: 14 }}>
              <div style={{ transform: 'scale(1.3)', transformOrigin: 'left center' }}>
                <Avatar name={v.customer} />
              </div>
              <div style={{ marginInlineStart: 8 }}>
                <div style={{ fontWeight: 800 }}>{v.customer}</div>
                <div className="faint" style={{ fontSize: 12.5 }}>
                  {v.email || o.customer?.phone || '—'}
                </div>
              </div>
            </div>
            <button className="abtn sm" style={{ width: '100%' }} onClick={() => navigate(`/users/${o.userId}`)}>
              <Icon name="users" size={14} /> View customer profile
            </button>
          </div>

          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Payment summary</h3>
            <div className="deflist">
              <div className="defrow">
                <span className="dk">Subtotal</span>
                <span className="dv num">{money(o.subtotal, v.cur)}</span>
              </div>
              <div className="defrow">
                <span className="dk">Method</span>
                <span className="dv">
                  <PayChip p={v.pay} />
                </span>
              </div>
              <div className="defrow">
                <span className="dk strong" style={{ color: 'var(--text)', fontWeight: 800 }}>
                  Total
                </span>
                <span className="dv num" style={{ fontSize: 17 }}>
                  {money(o.total, v.cur)}
                </span>
              </div>
            </div>
          </div>

          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>Quick actions</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {completable && (
                <button className="abtn sm ok" style={{ justifyContent: 'flex-start' }} onClick={() => setCompleteOpen(true)}>
                  <Icon name="check" size={14} /> Mark as completed
                </button>
              )}
              {refundable && (
                <button className="abtn sm danger" style={{ justifyContent: 'flex-start' }} onClick={() => setRefundOpen(true)}>
                  <Icon name="refresh" size={14} /> Initiate refund
                </button>
              )}
              {failable && (
                <>
                  <button className="abtn sm danger" style={{ justifyContent: 'flex-start' }} onClick={() => setFailOpen(true)}>
                    <Icon name="x" size={14} /> Mark as failed
                  </button>
                  <div className="faint" style={{ fontSize: 12 }}>
                    Stuck in pending (crashed mid-placement). Marking it failed moves no money — reverse any
                    charge via the customer's wallet adjust.
                  </div>
                </>
              )}
              {!completable && !refundable && !failable && (
                <div className="faint" style={{ fontSize: 12.5 }}>No actions available for this order.</div>
              )}
            </div>
          </div>
        </div>
      </div>

      {refundOpen && <RefundModal order={o} label={v.customer} cur={v.cur} onClose={() => setRefundOpen(false)} />}
      {completeOpen && <CompleteModal order={o} onClose={() => setCompleteOpen(false)} />}
      {failOpen && <FailModal order={o} onClose={() => setFailOpen(false)} />}
    </div>
  )
}

/**
 * Inline annotation shown on a refunded order's fulfillment card. Surfaces the
 * refund reason (from the status timeline) and the fact that delivered codes are
 * intentionally not re-pooled.
 */
function RefundedNotice({ order }: { order: AdminOrder }) {
  const reason = [...(order.fulfillment.statusTimeline ?? [])]
    .reverse()
    .find((e) => e.status === 'refunded')?.note

  return (
    <div
      style={{
        display: 'flex',
        gap: 10,
        padding: '11px 13px',
        marginBottom: 16,
        borderRadius: 11,
        background: 'rgba(255,77,109,.1)',
        border: '1px solid var(--danger)',
        color: 'var(--danger)',
      }}
    >
      <Icon name="refresh" size={16} />
      <div style={{ fontSize: 12.5, lineHeight: 1.5 }}>
        <b>This order was refunded.</b>
        {reason ? ` ${reason}.` : ''} Delivered codes are not returned to inventory.
      </div>
    </div>
  )
}

/** Confirmation dialog for refunding an order, with an optional reason. */
function RefundModal({
  order,
  label,
  cur,
  onClose,
}: {
  order: AdminOrder
  label: string
  cur: 'USD' | 'TRY'
  onClose: () => void
}) {
  const { t } = useTranslation()
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const refundM = useRefundOrder()

  const apply = () => {
    setError('')
    refundM.mutate(
      { id: order.id, reason: reason.trim() },
      {
        onSuccess: onClose,
        onError: (e) => setError(e instanceof ApiError ? e.message : 'Refund failed.'),
      },
    )
  }

  const restoresWallet = order.paymentMethod === 'wallet' && order.total > 0

  return (
    <Modal onClose={onClose} maxWidth={440}>
      <div style={{ padding: 22 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 11, marginBottom: 6 }}>
          <div
            style={{
              width: 40,
              height: 40,
              borderRadius: 11,
              background: 'rgba(255,77,109,.14)',
              color: 'var(--danger)',
              display: 'grid',
              placeItems: 'center',
            }}
          >
            <Icon name="refresh" size={20} />
          </div>
          <div>
            <h3 style={{ fontSize: 17, fontWeight: 800 }}>Refund order {order.id.slice(-8)}</h3>
            <div className="faint" style={{ fontSize: 12.5 }}>
              {money(order.total, cur)} · {label}
            </div>
          </div>
        </div>
        <p style={{ fontSize: 13, color: 'var(--text-dim)', margin: '14px 0 12px' }}>
          {restoresWallet
            ? `Restores ${money(order.total, cur)} to the customer's wallet.`
            : 'No wallet charge to reverse — this marks the order refunded.'}{' '}
          Delivered codes are not returned to inventory. This can't be undone.
        </p>
        <label className="alabel">Reason (optional)</label>
        <input
          className="afield"
          placeholder="e.g. customer request, failed delivery…"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 20, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={refundM.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn danger" onClick={apply} disabled={refundM.isPending}>
            <Icon name="check" size={15} /> {refundM.isPending ? 'Refunding…' : 'Confirm refund'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

/** Dialog for manually completing a processing order (credit / transfer / parked). */
function CompleteModal({ order, onClose }: { order: AdminOrder; onClose: () => void }) {
  const { t } = useTranslation()
  const [note, setNote] = useState('')
  const [transferRef, setTransferRef] = useState('')
  const [error, setError] = useState('')
  const completeM = useCompleteOrder()

  const apply = () => {
    setError('')
    completeM.mutate(
      { id: order.id, note: note.trim(), transferRef: transferRef.trim() },
      {
        onSuccess: onClose,
        onError: (e) => setError(e instanceof ApiError ? e.message : 'Completion failed.'),
      },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={440}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>
          Complete order {order.id.slice(-8)}
        </h3>
        <p style={{ fontSize: 13, color: 'var(--text-dim)', marginBottom: 16 }}>
          Confirm the credit/transfer was delivered outside the platform. The order moves to
          <b> completed</b> and the customer sees it immediately.
        </p>
        <label className="alabel">Reference (optional)</label>
        <input
          className="afield"
          placeholder="e.g. provider transaction id…"
          value={transferRef}
          onChange={(e) => setTransferRef(e.target.value)}
        />
        <label className="alabel" style={{ marginTop: 12 }}>Note (optional)</label>
        <input
          className="afield"
          placeholder="e.g. credited via partner portal…"
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 20, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={completeM.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn ok" onClick={apply} disabled={completeM.isPending}>
            <Icon name="check" size={15} /> {completeM.isPending ? 'Completing…' : 'Mark completed'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

/**
 * Dialog for the pending-order cleanup: moves a stuck order to `failed`. This
 * moves NO money — the warning tells the admin to reverse any charge via the
 * customer's wallet adjustment.
 */
function FailModal({ order, onClose }: { order: AdminOrder; onClose: () => void }) {
  const { t } = useTranslation()
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const failM = useFailOrder()

  const apply = () => {
    setError('')
    failM.mutate(
      { id: order.id, reason: reason.trim() },
      {
        onSuccess: onClose,
        onError: (e) => setError(e instanceof ApiError ? e.message : 'Could not mark failed.'),
      },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={440}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>
          Mark order {order.id.slice(-8)} as failed
        </h3>
        <p style={{ fontSize: 13, color: 'var(--text-dim)', marginBottom: 16 }}>
          Clears a stuck <b>pending</b> order (crashed mid-placement). This moves the order to
          <b> failed</b> and does <b>not</b> move any money. If the wallet was charged, reverse it
          via the customer's wallet adjustment.
        </p>
        <label className="alabel">Reason (optional)</label>
        <input
          className="afield"
          placeholder="e.g. stuck pending, never fulfilled…"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 20, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={failM.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn danger" onClick={apply} disabled={failM.isPending}>
            <Icon name="x" size={15} /> {failM.isPending ? 'Marking…' : 'Mark as failed'}
          </button>
        </div>
      </div>
    </Modal>
  )
}
