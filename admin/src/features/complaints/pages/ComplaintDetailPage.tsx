import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ErrorState, Icon, LoadingSpinner, PageHead, Toggle } from '@/components'
import { ApiError } from '@/lib/api-client'
import { money } from '@/lib/utils'
import { useCan } from '@/stores/auth'
import {
  CATEGORY_LABEL,
  COMPLAINT_STATUSES,
  STATUS_CLASS,
  STATUS_LABEL,
  contactLabel,
  whatsappLink,
  type ComplaintMessage,
  type ComplaintStatus,
} from '../api/complaints'
import {
  useAssignees,
  useComplaint,
  useReplyComplaint,
  useSetComplaintAssignee,
  useSetComplaintStatus,
} from '../hooks/useComplaints'

const MAX_BODY = 2000

export default function ComplaintDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id = '' } = useParams<{ id: string }>()
  const can = useCan()
  const canManage = can('complaints.manage')
  const [draft, setDraft] = useState('')
  const [internal, setInternal] = useState(false)
  const [error, setError] = useState('')

  // No polling while a draft is in the box (a refetch must not disturb typing).
  const { data, isLoading, isError, refetch } = useComplaint(id, draft.trim() === '')
  const { data: adminsRes } = useAssignees()
  const admins = adminsRes?.data ?? []
  const replyM = useReplyComplaint(id)
  const statusM = useSetComplaintStatus(id)
  const assignM = useSetComplaintAssignee(id)

  if (isLoading) return <div className="page"><LoadingSpinner /></div>
  if ((isError && !data) || !data?.data) {
    return (
      <div className="page">
        <ErrorState message="Couldn't load this complaint." onRetry={() => refetch()} />
      </div>
    )
  }

  const c = data.data
  const cust = c.customer
  const wa = whatsappLink(cust.phone)
  const closed = c.status === 'closed'
  const canSend = draft.trim().length > 0 && draft.length <= MAX_BODY && !replyM.isPending && (internal || !closed)

  const send = () => {
    setError('')
    replyM.mutate(
      { body: draft.trim(), internal },
      {
        onSuccess: () => {
          setDraft('')
          setInternal(false)
        },
        onError: (e) => setError(e instanceof ApiError ? e.message : 'Could not send.'),
      },
    )
  }

  return (
    <div className="page">
      <PageHead
        crumbs={[t('nav_complaints'), c.ref]}
        title={
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 12 }}>
            {c.ref}
            <span className={STATUS_CLASS[c.status]}>
              <i className="d" />
              {STATUS_LABEL[c.status]}
            </span>
          </span>
        }
        sub={`${CATEGORY_LABEL[c.category]} · ${c.subject}`}
      >
        <button className="abtn" onClick={() => navigate('/complaints')}>
          <Icon name="chevleft" size={15} /> {t('back')}
        </button>
      </PageHead>

      <div className="formgrid">
        <div className="fieldset">
          <div className="acard">
            <div className="panelhead">
              <Icon name="send" size={17} />
              <h3>Conversation</h3>
              {c.awaitingReply && (
                <div className="ph-act">
                  <span className="st st-warn">
                    <i className="d" />
                    Awaiting reply
                  </span>
                </div>
              )}
            </div>
            <div style={{ padding: 18, display: 'flex', flexDirection: 'column', gap: 12 }}>
              {c.messages.map((m) => (
                <Bubble key={m.id} m={m} />
              ))}
            </div>
            {canManage && (
              <div style={{ borderTop: '1px solid var(--border)', padding: 18 }}>
                <textarea
                  className="afield"
                  rows={4}
                  maxLength={MAX_BODY}
                  placeholder={
                    internal
                      ? 'Internal note — only staff can see this'
                      : closed
                        ? 'This complaint is closed. Reopen it to reply, or add an internal note.'
                        : 'Reply to the customer (they get an in-app notification)'
                  }
                  value={draft}
                  onChange={(e) => setDraft(e.target.value)}
                  style={internal ? { borderColor: 'var(--warn)', background: 'rgba(255,176,46,.06)' } : undefined}
                />
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginTop: 10, flexWrap: 'wrap' }}>
                  <label style={{ display: 'inline-flex', alignItems: 'center', gap: 8, fontSize: 13, cursor: 'pointer' }}>
                    <Toggle on={internal} onClick={() => setInternal((v) => !v)} />
                    Internal note
                  </label>
                  <span className="faint" style={{ fontSize: 12 }}>
                    {draft.length}/{MAX_BODY}
                  </span>
                  <button className="abtn primary" style={{ marginInlineStart: 'auto' }} disabled={!canSend} onClick={send}>
                    <Icon name="send" size={14} /> {replyM.isPending ? '…' : internal ? 'Add note' : 'Send reply'}
                  </button>
                </div>
                {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 8 }}>{error}</div>}
              </div>
            )}
          </div>
        </div>

        <div className="fieldset">
          <div className="acard">
            <div className="panelhead">
              <Icon name="users" size={17} />
              <h3>Customer</h3>
            </div>
            <div style={{ padding: '6px 18px 14px' }}>
              <div className="deflist">
                <div className="defrow">
                  <span className="dk">Name</span>
                  <span className="dv">{contactLabel(cust)}</span>
                </div>
                <div className="defrow">
                  <span className="dk">Phone</span>
                  <span className="dv">{cust.phone ? <a href={`tel:${cust.phone}`}>{cust.phone}</a> : '—'}</span>
                </div>
                <div className="defrow">
                  <span className="dk">Email</span>
                  <span className="dv">{cust.email ? <a href={`mailto:${cust.email}`}>{cust.email}</a> : '—'}</span>
                </div>
              </div>
              <div style={{ display: 'flex', gap: 8, marginTop: 12, flexWrap: 'wrap' }}>
                {wa && (
                  <a className="abtn xs ok" href={wa} target="_blank" rel="noreferrer">
                    <Icon name="send" size={13} /> WhatsApp
                  </a>
                )}
                {can('users.view') && (
                  <Link className="abtn xs" to={`/users/${c.userId}`}>
                    <Icon name="users" size={13} /> Open user
                  </Link>
                )}
              </div>
            </div>
          </div>

          {c.order && (
            <div className="acard">
              <div className="panelhead">
                <Icon name="bag" size={17} />
                <h3>Linked order</h3>
                <div className="ph-act">
                  <span className="bdg">{c.order.status}</span>
                </div>
              </div>
              <div style={{ padding: '6px 18px 14px' }}>
                <div className="deflist">
                  <div className="defrow">
                    <span className="dk">Order</span>
                    <span className="dv mono">{c.order.id.slice(-8)}</span>
                  </div>
                  {c.order.items.map((it, i) => (
                    <div className="defrow" key={i}>
                      <span className="dk">{it.title.en}</span>
                      <span className="dv">
                        {it.denomination} ×{it.qty}
                      </span>
                    </div>
                  ))}
                  <div className="defrow">
                    <span className="dk">Total</span>
                    <span className="dv">{money(c.order.total, c.order.currency === 'TRY' ? 'TRY' : 'USD')}</span>
                  </div>
                  <div className="defrow">
                    <span className="dk">Placed</span>
                    <span className="dv">{new Date(c.order.createdAt).toLocaleString()}</span>
                  </div>
                </div>
                {can('orders.view') && (
                  <Link className="abtn xs" style={{ marginTop: 12 }} to={`/orders/${c.order.id}`}>
                    <Icon name="bag" size={13} /> Open order
                  </Link>
                )}
              </div>
            </div>
          )}

          <div className="acard">
            <div className="panelhead">
              <Icon name="settings" size={17} />
              <h3>Handling</h3>
            </div>
            <div style={{ padding: 18, display: 'flex', flexDirection: 'column', gap: 14 }}>
              <div>
                <label className="alabel">{t('status')}</label>
                <select
                  className="select"
                  style={{ width: '100%' }}
                  value={c.status}
                  disabled={!canManage || statusM.isPending}
                  onChange={(e) => statusM.mutate(e.target.value as ComplaintStatus)}
                >
                  {COMPLAINT_STATUSES.map((s) => (
                    <option key={s} value={s}>
                      {STATUS_LABEL[s]}
                    </option>
                  ))}
                </select>
                <div className="faint" style={{ fontSize: 12, marginTop: 6 }}>
                  Resolved or closed notifies the customer. A customer reply reopens a resolved complaint.
                </div>
              </div>
              <div>
                <label className="alabel">Assignee</label>
                <select
                  className="select"
                  style={{ width: '100%' }}
                  value={c.assigneeId ?? ''}
                  disabled={!canManage || assignM.isPending}
                  onChange={(e) => assignM.mutate(e.target.value || null)}
                >
                  <option value="">Unassigned</option>
                  {admins.map((a) => (
                    <option key={a.id} value={a.id}>
                      {contactLabel(a)}
                    </option>
                  ))}
                </select>
              </div>
              <div className="faint" style={{ fontSize: 12 }}>
                Opened {new Date(c.createdAt).toLocaleString()}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

/** One thread entry: customer on the right, support on the left, internal notes flagged. */
function Bubble({ m }: { m: ComplaintMessage }) {
  const mine = m.author === 'customer'
  const bg = m.internal ? 'rgba(255,176,46,.10)' : mine ? 'var(--surface-2)' : 'rgba(91,139,255,.12)'
  return (
    <div style={{ display: 'flex', justifyContent: mine ? 'flex-end' : 'flex-start' }}>
      <div
        style={{
          maxWidth: '80%',
          background: bg,
          border: `1px ${m.internal ? 'dashed var(--warn)' : 'solid var(--border)'}`,
          borderRadius: 12,
          padding: '10px 12px',
        }}
      >
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', fontSize: 11.5, marginBottom: 4 }}>
          <b>{mine ? 'Customer' : m.adminLabel || 'Support'}</b>
          {m.internal && (
            <span className="st st-warn" style={{ padding: '1px 6px', fontSize: 10.5 }}>
              Internal
            </span>
          )}
          <span className="faint">{new Date(m.createdAt).toLocaleString()}</span>
        </div>
        <div style={{ whiteSpace: 'pre-wrap', fontSize: 13.5, lineHeight: 1.5 }}>{m.body}</div>
        {m.photoUrl && (
          <a href={m.photoUrl} target="_blank" rel="noreferrer" style={{ display: 'inline-block', marginTop: 8 }}>
            <img
              src={m.photoUrl}
              alt="Attachment"
              style={{ width: 120, height: 120, objectFit: 'cover', borderRadius: 8, border: '1px solid var(--border)' }}
            />
          </a>
        )}
      </div>
    </div>
  )
}
