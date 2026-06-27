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
import { useOrder } from '../hooks/useOrders'
import { adaptOrder } from '../lib/adaptOrder'

export default function OrderDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const { data, isLoading, isError, refetch } = useOrder(id)
  const [masked, setMasked] = useState(true)
  const [refundOpen, setRefundOpen] = useState(false)

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
  const refundable = o.status !== 'refunded' && o.status !== 'failed'

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
                  <span className={o.fulfillment.deliveredCode ? 'st st-ok' : 'st st-warn'}>
                    <i className="d" />
                    {o.fulfillment.deliveredCode ? 'Delivered' : 'Pending'}
                  </span>
                </div>
              </div>
              <div style={{ padding: 18 }}>
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
                  <span className={o.status === 'completed' ? 'st st-ok' : 'st st-warn'}>
                    <i className="d" />
                    {o.status === 'completed' ? 'Credited' : 'Processing'}
                  </span>
                </div>
              </div>
              <div style={{ padding: 18 }}>
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
                <div style={{ marginTop: 16, paddingTop: 16, borderTop: '1px solid var(--border)' }}>
                  <label className="alabel">Manually update transfer status</label>
                  <div className="ahint">
                    <Icon name="clock" size={13} /> Manual status updates are coming soon (not yet implemented).
                  </div>
                </div>
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
              {refundable && (
                <button className="abtn sm danger" style={{ justifyContent: 'flex-start' }} onClick={() => setRefundOpen(true)}>
                  <Icon name="refresh" size={14} /> Initiate refund
                </button>
              )}
            </div>
          </div>
        </div>
      </div>

      {refundOpen && (
        <Modal onClose={() => setRefundOpen(false)} maxWidth={440}>
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
                <h3 style={{ fontSize: 17, fontWeight: 800 }}>Refund order {v.id.slice(-8)}</h3>
                <div className="faint" style={{ fontSize: 12.5 }}>
                  {money(o.total, v.cur)} · {v.customer}
                </div>
              </div>
            </div>
            <div className="ahint" style={{ marginTop: 16 }}>
              <Icon name="clock" size={13} /> Refunds aren't wired yet — this action is coming soon.
            </div>
            <div style={{ display: 'flex', gap: 10, marginTop: 20, justifyContent: 'flex-end' }}>
              <button className="abtn" onClick={() => setRefundOpen(false)}>
                {t('cancel')}
              </button>
              <button className="abtn danger" disabled title="Coming soon">
                <Icon name="check" size={15} /> Confirm refund
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}
