import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Icon, PageHead, Art, Avatar, FfBadge, StatusBadge, PayChip, Modal, ComingSoonNote } from '@/components'
import { money } from '@/lib/utils'
import { orders } from '@/lib/mock/demo'

export default function OrderDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  // TODO: wire to GET /api/admin/orders/:id (mock; backend route stubbed 501).
  const o = orders.find((x) => x.id === id) || orders[0]
  const [masked, setMasked] = useState(true)
  const [refundOpen, setRefundOpen] = useState(false)
  const [tStatus, setTStatus] = useState<'processing' | 'completed'>('processing')

  return (
    <div className="page">
      <PageHead
        crumbs={[t('nav_orders'), o.id]}
        title={
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 12 }}>
            Order {o.id} <StatusBadge s={o.status} />
          </span>
        }
        sub={`Placed ${o.date} · ${o.customer}`}
      >
        <button className="abtn" onClick={() => navigate('/orders')}>
          <Icon name="chevleft" size={15} /> {t('back')}
        </button>
        <button className="abtn">
          <Icon name="download" size={15} /> Invoice
        </button>
        {o.status !== 'refunded' && o.status !== 'failed' && (
          <button className="abtn danger" onClick={() => setRefundOpen(true)}>
            <Icon name="refresh" size={15} /> {t('refund')}
          </button>
        )}
      </PageHead>

      <ComingSoonNote mock />

      <div className="formgrid">
        <div className="fieldset">
          <div className="acard">
            <div className="panelhead">
              <Icon name="bag" size={17} />
              <h3>Items</h3>
              <div className="ph-act">
                <FfBadge ff={o.ff} />
              </div>
            </div>
            <div style={{ padding: 18 }}>
              <div style={{ display: 'flex', gap: 14, alignItems: 'center' }}>
                <Art art={o.art} size={56} radius={12} />
                <div style={{ flex: 1 }}>
                  <div style={{ fontWeight: 800, fontSize: 15 }}>{o.product}</div>
                  <div className="faint" style={{ fontSize: 12.5 }}>
                    Qty {o.qty} · {o.cur}
                  </div>
                </div>
                <div className="num strong" style={{ fontSize: 16 }}>
                  {money(o.amount, o.cur)}
                </div>
              </div>
            </div>
          </div>

          {o.ff === 'code' && (
            <div className="acard" style={{ borderColor: 'var(--ff-code-bd)' }}>
              <div className="panelhead">
                <span className="bdg ff-code">
                  <i className="d" />
                  Code delivery
                </span>
                <div className="ph-act">
                  <span className="st st-ok">
                    <i className="d" />
                    Delivered
                  </span>
                </div>
              </div>
              <div style={{ padding: 18 }}>
                <label className="alabel">Delivered code (admin view)</label>
                <div className={'avault' + (masked ? ' masked' : '')}>
                  <span className="vc">X3K9 — 7F2P — QW41 — 8N6R</span>
                  <div style={{ display: 'flex', gap: 6 }}>
                    <button className="abtn xs" onClick={() => setMasked(!masked)}>
                      <Icon name={masked ? 'eye' : 'eyeoff'} size={13} /> {masked ? 'Reveal' : 'Hide'}
                    </button>
                    <button className="abtn xs">
                      <Icon name="copy" size={13} /> Copy
                    </button>
                  </div>
                </div>
                <div className="deflist" style={{ marginTop: 8 }}>
                  <div className="defrow">
                    <span className="dk">Code ID</span>
                    <span className="dv mono">CODE-88241</span>
                  </div>
                  <div className="defrow">
                    <span className="dk">Delivered at</span>
                    <span className="dv">Jun 3, 2026 · 14:08</span>
                  </div>
                  <div className="defrow">
                    <span className="dk">Delivery channel</span>
                    <span className="dv">In-app vault + email</span>
                  </div>
                </div>
              </div>
            </div>
          )}

          {o.ff === 'credit' && (
            <div className="acard" style={{ borderColor: 'var(--ff-credit-bd)' }}>
              <div className="panelhead">
                <span className="bdg ff-credit">
                  <i className="d" />
                  Account credit
                </span>
                <div className="ph-act">
                  <span className="st st-ok">
                    <i className="d" />
                    Credited
                  </span>
                </div>
              </div>
              <div style={{ padding: 18 }}>
                <div className="deflist">
                  <div className="defrow">
                    <span className="dk">Player / Account ID</span>
                    <span className="dv mono">5521 8842 1190</span>
                  </div>
                  <div className="defrow">
                    <span className="dk">Provider</span>
                    <span className="dv">Direct API · Midasbuy</span>
                  </div>
                  <div className="defrow">
                    <span className="dk">Amount credited</span>
                    <span className="dv">660 UC</span>
                  </div>
                  <div className="defrow">
                    <span className="dk">Confirmation</span>
                    <span className="dv">
                      <span className="st st-ok">
                        <i className="d" />
                        Confirmed · ref MB-44128
                      </span>
                    </span>
                  </div>
                  <div className="defrow">
                    <span className="dk">Credited at</span>
                    <span className="dv">Jun 3, 2026 · 14:08</span>
                  </div>
                </div>
                <div
                  style={{
                    marginTop: 14,
                    padding: 12,
                    background: 'var(--ff-credit-bg)',
                    borderRadius: 'var(--ar-sm)',
                    fontSize: 12.5,
                    color: 'var(--text-dim)',
                    display: 'flex',
                    gap: 8,
                  }}
                >
                  <Icon name="bolt" size={15} /> No code needed — balance credited directly to the player account.
                </div>
              </div>
            </div>
          )}

          {o.ff === 'transfer' && (
            <div className="acard" style={{ borderColor: 'var(--ff-transfer-bd)' }}>
              <div className="panelhead">
                <span className="bdg ff-transfer">
                  <i className="d" />
                  Money transfer
                </span>
                <div className="ph-act">
                  <span className="st st-warn">
                    <i className="d" />
                    Processing
                  </span>
                </div>
              </div>
              <div style={{ padding: 18 }}>
                <div className="g2" style={{ alignItems: 'start' }}>
                  <div className="timeline">
                    {(
                      [
                        ['Submitted', 'Jun 5 · 09:02', 'done'],
                        ['Processing', 'Jun 5 · 09:14', tStatus === 'processing' ? 'active' : 'done'],
                        ['Completed', tStatus === 'completed' ? 'Just now' : 'Pending', tStatus === 'completed' ? 'done' : 'todo'],
                      ] as [string, string, string][]
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
                  <div>
                    <div className="deflist">
                      <div className="defrow">
                        <span className="dk">Reference</span>
                        <span className="dv mono">WU-7741-2290</span>
                      </div>
                      <div className="defrow">
                        <span className="dk">Recipient</span>
                        <span className="dv">Ahmad Saleh</span>
                      </div>
                      <div className="defrow">
                        <span className="dk">Country</span>
                        <span className="dv">Jordan 🇯🇴</span>
                      </div>
                    </div>
                  </div>
                </div>
                <div style={{ marginTop: 16, paddingTop: 16, borderTop: '1px solid var(--border)' }}>
                  <label className="alabel">Manually update transfer status</label>
                  <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
                    <div className="aseg">
                      {(
                        [
                          ['processing', 'Processing'],
                          ['completed', 'Completed'],
                        ] as ['processing' | 'completed', string][]
                      ).map(([k, l]) => (
                        <button key={k} className={tStatus === k ? 'on' : ''} onClick={() => setTStatus(k)}>
                          {l}
                        </button>
                      ))}
                    </div>
                    <button className="abtn sm primary">
                      <Icon name="check" size={14} /> Update &amp; notify customer
                    </button>
                  </div>
                  <div className="ahint">Transfer is the one fulfillment type that may need manual intervention.</div>
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
                {(
                  [
                    ['Order placed', o.date, 'done', 'bag'],
                    ['Payment captured', o.date, 'done', 'card'],
                    [
                      o.status === 'refunded' ? 'Refund issued' : 'Fulfilled',
                      o.date,
                      o.status === 'failed' ? 'todo' : 'done',
                      o.status === 'refunded' ? 'refresh' : 'checkc',
                    ],
                  ] as [string, string, string, 'bag' | 'card' | 'refresh' | 'checkc'][]
                ).map(([title, sub, st, ic], i) => (
                  <div className={'tlitem ' + st} key={i}>
                    <div className="tldot">
                      <Icon name={ic} size={14} />
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
                <Avatar name={o.customer} />
              </div>
              <div style={{ marginInlineStart: 8 }}>
                <div style={{ fontWeight: 800 }}>{o.customer}</div>
                <div className="faint" style={{ fontSize: 12.5 }}>
                  {o.email}
                </div>
              </div>
            </div>
            <button className="abtn sm" style={{ width: '100%' }} onClick={() => navigate('/users/U-9001')}>
              <Icon name="users" size={14} /> View customer profile
            </button>
          </div>

          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Payment summary</h3>
            <div className="deflist">
              <div className="defrow">
                <span className="dk">Subtotal</span>
                <span className="dv num">{money(o.amount, o.cur)}</span>
              </div>
              <div className="defrow">
                <span className="dk">Discount</span>
                <span className="dv num" style={{ color: 'var(--ok)' }}>
                  −{money(0, o.cur)}
                </span>
              </div>
              <div className="defrow">
                <span className="dk">Method</span>
                <span className="dv">
                  <PayChip p={o.pay} />
                </span>
              </div>
              <div className="defrow">
                <span className="dk strong" style={{ color: 'var(--text)', fontWeight: 800 }}>
                  Total
                </span>
                <span className="dv num" style={{ fontSize: 17 }}>
                  {money(o.amount, o.cur)}
                </span>
              </div>
            </div>
          </div>

          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>Quick actions</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              <button className="abtn sm" style={{ justifyContent: 'flex-start' }}>
                <Icon name="send" size={14} /> Resend delivery email
              </button>
              <button className="abtn sm" style={{ justifyContent: 'flex-start' }}>
                <Icon name="copy" size={14} /> Copy order link
              </button>
              {o.status !== 'refunded' && (
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
                <h3 style={{ fontSize: 17, fontWeight: 800 }}>Refund order {o.id}</h3>
                <div className="faint" style={{ fontSize: 12.5 }}>
                  {money(o.amount, o.cur)} · {o.customer}
                </div>
              </div>
            </div>
            <label className="alabel" style={{ marginTop: 16 }}>
              Refund to
            </label>
            <div className="g2">
              <div className="chip on" style={{ justifyContent: 'center', padding: '11px' }}>
                Store wallet
              </div>
              <div className="chip" style={{ justifyContent: 'center', padding: '11px' }}>
                Original method
              </div>
            </div>
            <label className="alabel" style={{ marginTop: 14 }}>
              Amount
            </label>
            <input className="afield" defaultValue={Math.abs(o.amount).toFixed(2)} />
            <label className="alabel" style={{ marginTop: 14 }}>
              Reason
            </label>
            <select className="select" style={{ width: '100%' }}>
              <option>Customer request</option>
              <option>Failed delivery</option>
              <option>Duplicate order</option>
              <option>Fraud / chargeback</option>
            </select>
            <div style={{ display: 'flex', gap: 10, marginTop: 20, justifyContent: 'flex-end' }}>
              <button className="abtn" onClick={() => setRefundOpen(false)}>
                {t('cancel')}
              </button>
              <button className="abtn danger" onClick={() => setRefundOpen(false)}>
                <Icon name="check" size={15} /> Confirm refund
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}
