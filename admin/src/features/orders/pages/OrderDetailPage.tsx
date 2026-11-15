import { useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { Link, useNavigate, useParams } from 'react-router-dom'
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
import { apiErrorText } from '@/lib/apiErrorText'
import { useCan } from '@/stores/auth'
import { useOrder } from '../hooks/useOrders'
import { useRefundOrder, useCompleteOrder, useFailOrder } from '../hooks/useOrderMutations'
import { adaptOrder, codeLines, creditLines, itemQtyLabel, productEditPath } from '../lib/adaptOrder'
import { timelineNote, timelineStatus } from '../lib/timelineText'
import type { AdminOrder, AdminOrderItem } from '../api/orders'
import { SupplierPanel } from '../components/SupplierPanel'
import { orderNumber } from '@/lib/orderNumber'

/** isOrderFileUrl detects a checkout photo-field upload (re-encoded JPEG under
 *  the order-files/ keyspace — the API rejects any other link for a file field). */
function isOrderFileUrl(v: string): boolean {
  return /^https?:\/\//.test(v) && v.includes('/order-files/') && v.endsWith('.jpg')
}

export default function OrderDetailPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const can = useCan()
  const canManage = can('orders.manage')
  // The product editor route is guarded by the products domain (view).
  const canViewProducts = can('products.view')
  const { data, isLoading, isError, refetch } = useOrder(id)
  // Codes start masked; each one is revealed on its own.
  const [revealed, setRevealed] = useState<Set<string>>(() => new Set())
  const [refundOpen, setRefundOpen] = useState(false)
  const [completeOpen, setCompleteOpen] = useState(false)
  const [failOpen, setFailOpen] = useState(false)

  if (isLoading) return <div className="page"><LoadingSpinner /></div>
  // A failed BACKGROUND poll (30s auto-refresh) keeps showing the cached order
  // (and any open modal); only a load with no data at all shows the error.
  if ((isError && !data) || !data?.data) {
    return (
      <div className="page">
        <PageHead back="/orders" title={t('nav_orders')} />
        <ErrorState message={t('ord_d_load_error')} onRetry={() => refetch()} />
      </div>
    )
  }

  const o = data.data
  const v = adaptOrder(o)
  const item = o.items[0]
  const credit = creditLines(o.items)
  const codes = codeLines(o.items, o.fulfillment.deliveredCode)
  const codeCount = codes.reduce((n, l) => n + l.codes.length, 0)
  // Each fulfillment card shows whenever the order has lines of that kind, so a
  // mixed order (codes delivered + manual lines processing) shows both.
  const showCodes = codes.length > 0 || v.ff === 'code'
  const showCredit = credit.lines.length > 0 || v.ff === 'credit'
  const transferItem = o.items.find((it) => it.fulfillmentType === 'transfer')
  const showTransfer = !!transferItem || v.ff === 'transfer'
  const toggleCode = (key: string) =>
    setRevealed((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
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
        back="/orders"
        crumbs={[t('nav_orders'), orderNumber(v.id)]}
        title={
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 12 }}>
            {t('ord_d_title', { n: orderNumber(v.id) })} <StatusBadge s={v.status} />
          </span>
        }
        sub={t('ord_d_sub', { date: v.date, customer: v.customer })}
      >
        {refundable && (
          <button className="abtn danger" onClick={() => setRefundOpen(true)}>
            <Icon name="refresh" size={15} /> {t('refund')}
          </button>
        )}
      </PageHead>

      <SupplierPanel order={o} canManage={canManage} />

      <div className="formgrid">
        <div className="fieldset">
          <div className="acard">
            <div className="panelhead">
              <Icon name="bag" size={17} />
              <h3>{t('ord_d_items')}</h3>
              <div className="ph-act">
                <FfBadge ff={v.ff} />
              </div>
            </div>
            <div style={{ padding: 18, display: 'flex', flexDirection: 'column', gap: 12 }}>
              {o.items.map((it, i) => {
                // Deleted product (or no Products access) → plain text, never a dead link.
                const href = productEditPath(o, it.productId, canViewProducts)
                return (
                  <div key={i} style={{ display: 'flex', gap: 14, alignItems: 'center' }}>
                    {href ? (
                      // Same target as the name link: out of the tab order (one stop per
                      // line), but labelled rather than aria-hidden — it can still take focus on click.
                      <Link to={href} className="prodlink art" tabIndex={-1} aria-label={t('ord_d_open_product_aria', { name: it.title.en })}>
                        <Art art={v.art} size={56} radius={12} />
                      </Link>
                    ) : (
                      <Art art={v.art} size={56} radius={12} />
                    )}
                    <div style={{ flex: 1 }}>
                      <div style={{ fontWeight: 800, fontSize: 15 }}>
                        {href ? (
                          <Link to={href} className="prodlink" title={t('sup_open_product')}>
                            {it.title.en}
                          </Link>
                        ) : (
                          it.title.en
                        )}
                      </div>
                      <div className="faint" style={{ fontSize: 12.5 }}>
                        {itemQtyLabel(it)} · {o.currency}
                        {/* A per-unit line's denomination just repeats the amount. */}
                        {it.denomination && it.amount == null ? ` · ${it.denomination}` : ''}
                      </div>
                      {it.fields && it.fields.length > 0 && (
                        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 14px', marginTop: 6 }}>
                          {it.fields.map((f) => (
                            <span key={f.key} style={{ fontSize: 12.5 }}>
                              <span className="faint">{f.label?.en || f.key}: </span>
                              {isOrderFileUrl(f.value) ? (
                                // A checkout photo field (QR / barcode …): show it, click to open full size.
                                <a href={f.value} target="_blank" rel="noreferrer" style={{ verticalAlign: 'middle' }}>
                                  <img
                                    src={f.value}
                                    alt={f.label?.en || f.key}
                                    style={{
                                      width: 72,
                                      height: 72,
                                      objectFit: 'cover',
                                      borderRadius: 8,
                                      border: '1px solid var(--border)',
                                    }}
                                  />
                                </a>
                              ) : (
                                <b style={{ fontWeight: 700 }}>{f.value}</b>
                              )}
                            </span>
                          ))}
                        </div>
                      )}
                    </div>
                    <div className="num strong" style={{ fontSize: 16 }}>
                      {money(it.price * it.qty, v.cur)}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>

          {showCodes && (
            <div className="acard" style={{ borderColor: 'var(--ff-code-bd)' }}>
              <div className="panelhead">
                <span className="bdg ff-code">
                  <i className="d" />
                  {t('ord_d_code_delivery')}
                </span>
                <div className="ph-act">
                  {refunded ? (
                    <span className="st st-danger">
                      <i className="d" />
                      Refunded
                    </span>
                  ) : (
                    <span className={codeCount > 0 ? 'st st-ok' : 'st st-warn'}>
                      <i className="d" />
                      {codeCount > 0
                        ? codeCount > 1
                          ? t('ord_d_n_delivered', { n: codeCount })
                          : t('ord_d_delivered')
                        : t('pending')}
                    </span>
                  )}
                </div>
              </div>
              <div style={{ padding: 18 }}>
                {refunded && <RefundedNotice order={o} />}
                {codes.length === 0 ? (
                  <>
                    <label className="alabel">{t('ord_d_code_admin')}</label>
                    <div className="avault masked">
                      <span className="vc">{t('ord_d_not_delivered')}</span>
                    </div>
                  </>
                ) : (
                  codes.map((line, li) => (
                    <div key={li} style={{ marginTop: li > 0 ? 14 : 0 }}>
                      <label className="alabel">
                        {codes.length > 1 || line.codes.length > 1
                          ? t('ord_d_codes_admin', { title: line.title, count: line.codes.length })
                          : t('ord_d_code_admin')}
                      </label>
                      {line.codes.map((code, ci) => {
                        const key = `${li}-${ci}`
                        const masked = !revealed.has(key)
                        return (
                          <div
                            key={key}
                            className={'avault' + (masked ? ' masked' : '')}
                            style={{ marginTop: ci > 0 ? 8 : 0 }}
                          >
                            <span className="vc">{code}</span>
                            <div style={{ display: 'flex', gap: 6 }}>
                              <button className="abtn xs" onClick={() => toggleCode(key)}>
                                <Icon name={masked ? 'eye' : 'eyeoff'} size={13} /> {masked ? t('ord_d_reveal') : t('ord_d_hide')}
                              </button>
                            </div>
                          </div>
                        )
                      })}
                    </div>
                  ))
                )}
              </div>
            </div>
          )}

          {showCredit && (
            <div className="acard" style={{ borderColor: 'var(--ff-credit-bd)' }}>
              <div className="panelhead">
                <span className="bdg ff-credit">
                  <i className="d" />
                  {t('ff_credit')}
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
                      {o.status === 'completed' ? t('ord_d_credited') : t('ord_status_processing')}
                    </span>
                  )}
                </div>
              </div>
              <div style={{ padding: 18 }}>
                {refunded && <RefundedNotice order={o} />}
                {credit.distinctIds > 1 && o.status === 'processing' && (
                  <div style={{ color: 'var(--warn)', fontSize: 12.5, fontWeight: 600, marginBottom: 12 }}>
                    {t('ord_d_multi_accounts', { n: credit.distinctIds })}
                  </div>
                )}
                <div className="deflist">
                  {credit.lines.map((l, i) => (
                    <div className="defrow" key={i}>
                      <span className="dk">
                        {credit.lines.length > 1 ? l.title : t('ord_d_player_id')}
                        {' · '}
                        {l.qtyLabel}
                      </span>
                      <span className="dv">
                        <span className="mono">{l.playerId || (i === 0 && o.fulfillment.creditedToId) || '—'}</span>
                        {credit.lines.length > 1 && <span className="faint"> · {money(l.total, v.cur)}</span>}
                      </span>
                    </div>
                  ))}
                  <div className="defrow">
                    <span className="dk">{t('ord_d_order_total')}</span>
                    <span className="dv">{money(o.total, v.cur)}</span>
                  </div>
                </div>
                {completable && (
                  <div style={{ marginTop: 16, paddingTop: 16, borderTop: '1px solid var(--border)' }}>
                    <label className="alabel">{t('ord_d_manual_credit')}</label>
                    <button className="abtn sm ok" onClick={() => setCompleteOpen(true)}>
                      <Icon name="check" size={14} /> {t('ord_d_mark_credit')}
                    </button>
                  </div>
                )}
              </div>
            </div>
          )}

          {showTransfer && (
            <div className="acard" style={{ borderColor: 'var(--ff-transfer-bd)' }}>
              <div className="panelhead">
                <span className="bdg ff-transfer">
                  <i className="d" />
                  {t('ff_transfer')}
                </span>
                <div className="ph-act">
                  <StatusBadge s={v.status} />
                </div>
              </div>
              <div style={{ padding: 18 }}>
                {refunded && <RefundedNotice order={o} />}
                <div className="deflist">
                  <div className="defrow">
                    <span className="dk">{t('ord_col_reference')}</span>
                    <span className="dv mono">{o.fulfillment.transferRef || '—'}</span>
                  </div>
                  <div className="defrow">
                    <span className="dk">{t('ord_d_recipient')}</span>
                    <span className="dv">{(transferItem ?? item)?.recipient?.name || '—'}</span>
                  </div>
                  <div className="defrow">
                    <span className="dk">{t('ord_d_country')}</span>
                    <span className="dv">{(transferItem ?? item)?.recipient?.country || '—'}</span>
                  </div>
                </div>
                {completable && (
                  <div style={{ marginTop: 16, paddingTop: 16, borderTop: '1px solid var(--border)' }}>
                    <label className="alabel">{t('ord_d_manual_transfer')}</label>
                    <button className="abtn sm ok" onClick={() => setCompleteOpen(true)}>
                      <Icon name="check" size={14} /> {t('ord_d_mark_transfer')}
                    </button>
                  </div>
                )}
              </div>
            </div>
          )}

          <div className="acard">
            <div className="panelhead">
              <Icon name="activity" size={17} />
              <h3>{t('ord_d_activity')}</h3>
            </div>
            <div style={{ padding: 18 }}>
              <div className="timeline">
                {(o.fulfillment.statusTimeline?.length
                  ? o.fulfillment.statusTimeline.map(
                      (e) =>
                        [
                          timelineStatus(e.status, t),
                          (e.note && timelineNote(e.note, t, i18n.language)) || v.date,
                          'done',
                        ] as [string, string, string],
                    )
                  : ([
                      [t('tl_placed'), v.date, 'done'],
                      [t('tl_payment_captured'), v.date, 'done'],
                      [
                        o.status === 'refunded'
                          ? t('tl_refund_issued')
                          : o.status === 'completed'
                            ? t('tl_fulfilled')
                            : t('tl_in_progress'),
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
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>{t('ord_col_customer')}</h3>
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
              <Icon name="users" size={14} /> {t('ord_d_view_customer')}
            </button>
          </div>

          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>{t('ord_d_payment_summary')}</h3>
            <div className="deflist">
              <div className="defrow">
                <span className="dk">{t('ord_d_subtotal')}</span>
                <span className="dv num">{money(o.subtotal, v.cur)}</span>
              </div>
              <div className="defrow">
                <span className="dk">{t('col_method')}</span>
                <span className="dv">
                  <PayChip p={v.pay} />
                </span>
              </div>
              <div className="defrow">
                <span className="dk strong" style={{ color: 'var(--text)', fontWeight: 800 }}>
                  {t('ord_d_total')}
                </span>
                <span className="dv num" style={{ fontSize: 17 }}>
                  {money(o.total, v.cur)}
                </span>
              </div>
            </div>
          </div>

          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>{t('ord_d_quick_actions')}</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {completable && (
                <button className="abtn sm ok" style={{ justifyContent: 'flex-start' }} onClick={() => setCompleteOpen(true)}>
                  <Icon name="check" size={14} /> {t('ord_d_mark_completed')}
                </button>
              )}
              {refundable && (
                <button className="abtn sm danger" style={{ justifyContent: 'flex-start' }} onClick={() => setRefundOpen(true)}>
                  <Icon name="refresh" size={14} /> {t('ord_d_initiate_refund')}
                </button>
              )}
              {failable && (
                <>
                  <button className="abtn sm danger" style={{ justifyContent: 'flex-start' }} onClick={() => setFailOpen(true)}>
                    <Icon name="x" size={14} /> {t('ord_d_mark_failed')}
                  </button>
                  <div className="faint" style={{ fontSize: 12 }}>
                    {t('ord_d_fail_hint')}
                  </div>
                </>
              )}
              {!completable && !refundable && !failable && (
                <div className="faint" style={{ fontSize: 12.5 }}>{t('ord_d_no_actions')}</div>
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
  const { t, i18n } = useTranslation()
  const note = [...(order.fulfillment.statusTimeline ?? [])]
    .reverse()
    .find((e) => e.status === 'refunded')?.note
  const reason = note ? timelineNote(note, t, i18n.language) : note

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
        <b>{t('ord_d_refunded_title')}</b>
        {reason ? ` ${reason}.` : ''} {t('ord_d_codes_not_returned')}
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
  // Double-delivery guard: the API refuses (ORDER_AT_SUPPLIER) while a supplier
  // may still deliver; the admin then confirms explicitly.
  const [atSupplier, setAtSupplier] = useState(false)
  const [supplierAck, setSupplierAck] = useState(false)

  const apply = () => {
    if (atSupplier && !supplierAck) return
    setError('')
    refundM.mutate(
      { id: order.id, reason: reason.trim(), acknowledgeSupplier: atSupplier && supplierAck },
      {
        onSuccess: onClose,
        onError: (e) => {
          if (e instanceof ApiError && e.code === 'ORDER_AT_SUPPLIER') {
            setAtSupplier(true)
            return
          }
          setError(apiErrorText(e, 'ord_d_refund_failed'))
        },
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
            <h3 style={{ fontSize: 17, fontWeight: 800 }}>{t('ord_d_refund_title', { n: orderNumber(order.id) })}</h3>
            <div className="faint" style={{ fontSize: 12.5 }}>
              {money(order.total, cur)} · {label}
            </div>
          </div>
        </div>
        <p style={{ fontSize: 13, color: 'var(--text-dim)', margin: '14px 0 12px' }}>
          {restoresWallet ? t('ord_d_restores', { amount: money(order.total, cur) }) : t('ord_d_no_wallet')}{' '}
          {t('ord_d_refund_irreversible')}
        </p>
        <label className="alabel">{t('ord_d_reason_opt')}</label>
        <input
          className="afield"
          placeholder={t('ord_d_refund_ph')}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {atSupplier && (
          <div
            role="alert"
            style={{
              marginTop: 14,
              padding: 12,
              borderRadius: 10,
              border: '1px solid var(--border-strong)',
              background: 'var(--surface-2)',
              fontSize: 13,
            }}
          >
            <b>{t('ord_d_at_supplier_title')}</b> {t('ord_d_at_supplier_refund')}
            <label style={{ display: 'flex', gap: 8, marginTop: 10, alignItems: 'center', color: 'var(--danger)' }}>
              <input type="checkbox" checked={supplierAck} onChange={(e) => setSupplierAck(e.target.checked)} />
              {t('ord_d_refund_anyway')}
            </label>
          </div>
        )}
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 20, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={refundM.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn danger" onClick={apply} disabled={refundM.isPending || (atSupplier && !supplierAck)}>
            <Icon name="check" size={15} /> {refundM.isPending ? t('ord_d_refunding') : t('ord_d_confirm_refund')}
          </button>
        </div>
      </div>
    </Modal>
  )
}

/** The amount in a checklist line: the typed units, or "× N" for a package line. */
function creditedQty(it: AdminOrderItem | undefined, qtyLabel: string): string {
  return it && !(it.amount != null && it.amount > 0) ? `× ${it.qty}` : qtyLabel
}

/** Dialog for manually completing a processing order (credit / transfer / parked). */
function CompleteModal({ order, onClose }: { order: AdminOrder; onClose: () => void }) {
  const { t } = useTranslation()
  const [note, setNote] = useState('')
  const [transferRef, setTransferRef] = useState('')
  const [error, setError] = useState('')
  const completeM = useCompleteOrder()
  // Completion is order-level, so with several credit lines the operator ticks
  // each one off before the order can be marked done (local only — not sent).
  const { lines } = creditLines(order.items)
  const creditItems = order.items.filter((it) => it.fulfillmentType === 'account_credit')
  const needsChecklist = lines.length > 1
  const [ticked, setTicked] = useState<boolean[]>(() => lines.map(() => false))
  const allTicked = !needsChecklist || ticked.every(Boolean)
  // Double-delivery guard: the API refuses (ORDER_AT_SUPPLIER) while a supplier
  // may still deliver; the admin then confirms explicitly.
  const [atSupplier, setAtSupplier] = useState(false)
  const [supplierAck, setSupplierAck] = useState(false)
  const ready = allTicked && (!atSupplier || supplierAck)

  const apply = () => {
    if (!ready) return
    setError('')
    completeM.mutate(
      { id: order.id, note: note.trim(), transferRef: transferRef.trim(), acknowledgeSupplier: atSupplier && supplierAck },
      {
        onSuccess: onClose,
        onError: (e) => {
          if (e instanceof ApiError && e.code === 'ORDER_AT_SUPPLIER') {
            setAtSupplier(true)
            return
          }
          setError(apiErrorText(e, 'ord_d_complete_failed'))
        },
      },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={440}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>
          {t('ord_d_complete_title', { n: orderNumber(order.id) })}
        </h3>
        <p style={{ fontSize: 13, color: 'var(--text-dim)', marginBottom: 16 }}>
          <Trans i18nKey="ord_d_complete_body" components={{ b: <b /> }} />
        </p>
        {needsChecklist && (
          <div style={{ marginBottom: 16 }}>
            <label className="alabel">{t('ord_d_tick_each')}</label>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {lines.map((l, i) => (
                <label key={i} style={{ display: 'flex', gap: 10, alignItems: 'center', fontSize: 13, cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={ticked[i]}
                    onChange={(e) => setTicked((prev) => prev.map((x, j) => (j === i ? e.target.checked : x)))}
                  />
                  <span>
                    <Trans
                      i18nKey="ord_d_credited_line"
                      values={{ qty: creditedQty(creditItems[i], l.qtyLabel), title: l.title, id: l.playerId || '—' }}
                      components={{ b: <b className="mono" /> }}
                    />
                  </span>
                </label>
              ))}
            </div>
          </div>
        )}
        <label className="alabel">{t('ord_d_ref_opt')}</label>
        <input
          className="afield"
          placeholder={t('ord_d_ref_ph')}
          value={transferRef}
          onChange={(e) => setTransferRef(e.target.value)}
        />
        <label className="alabel" style={{ marginTop: 12 }}>{t('ord_d_note_opt')}</label>
        <input
          className="afield"
          placeholder={t('ord_d_note_ph')}
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
        {atSupplier && (
          <div
            role="alert"
            style={{
              marginTop: 14,
              padding: 12,
              borderRadius: 10,
              border: '1px solid var(--border-strong)',
              background: 'var(--surface-2)',
              fontSize: 13,
            }}
          >
            <b>{t('ord_d_at_supplier_title')}</b> {t('ord_d_at_supplier_complete')}
            <label style={{ display: 'flex', gap: 8, marginTop: 10, alignItems: 'center', color: 'var(--danger)' }}>
              <input type="checkbox" checked={supplierAck} onChange={(e) => setSupplierAck(e.target.checked)} />
              {t('ord_d_complete_anyway')}
            </label>
          </div>
        )}
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 20, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={completeM.isPending}>
            {t('cancel')}
          </button>
          <button
            className="abtn ok"
            onClick={apply}
            disabled={completeM.isPending || !ready}
            title={allTicked ? undefined : t('ord_d_tick_first')}
          >
            <Icon name="check" size={15} /> {completeM.isPending ? t('ord_d_completing') : t('ord_d_mark_completed_btn')}
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
        onError: (e) => setError(apiErrorText(e, 'ord_d_fail_failed')),
      },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={440}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>
          {t('ord_d_fail_title', { n: orderNumber(order.id) })}
        </h3>
        <p style={{ fontSize: 13, color: 'var(--text-dim)', marginBottom: 16 }}>
          <Trans i18nKey="ord_d_fail_body" components={{ b: <b /> }} />
        </p>
        <label className="alabel">{t('ord_d_reason_opt')}</label>
        <input
          className="afield"
          placeholder={t('ord_d_fail_ph')}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 20, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={failM.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn danger" onClick={apply} disabled={failM.isPending}>
            <Icon name="x" size={15} /> {failM.isPending ? t('ord_d_marking') : t('ord_d_mark_failed')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
