import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Button, Card, ErrorState, Icon, LoadingSpinner } from '@/components'
import { statusBadgeClass, type ComplaintMessage } from '../api/complaints'
import { useComplaint, useReplyComplaint } from '../hooks/useComplaints'

export default function ComplaintDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id = '' } = useParams()
  const [draft, setDraft] = useState('')
  const [error, setError] = useState('')
  const query = useComplaint(id, draft.trim() === '')
  const reply = useReplyComplaint(id)

  if (query.isLoading) return <LoadingSpinner />
  if (!query.data) {
    return (
      <ErrorState
        title={t('no_results')}
        sub={t('no_results_sub')}
        onRetry={() => query.refetch()}
        retryLabel={t('retry')}
      />
    )
  }

  const c = query.data
  const closed = c.status === 'closed'
  const send = () => {
    const body = draft.trim()
    if (!body) return
    setError('')
    reply.mutate(
      { body },
      {
        onSuccess: () => setDraft(''),
        onError: (e) => setError(e instanceof Error ? e.message : t('complaint_send_failed')),
      },
    )
  }

  return (
    <div className="wrap" style={{ padding: '26px 0 50px', maxWidth: 720 }}>
      <a
        className="small clickable faint"
        onClick={() => navigate('/complaints')}
        style={{ display: 'inline-block', marginBottom: 16 }}
      >
        ← {t('complaints')}
      </a>
      <Card style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div className="col" style={{ gap: 6 }}>
          <div className="row" style={{ gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
            <span className="small faint" style={{ fontWeight: 700 }}>
              {c.ref}
            </span>
            <span className={statusBadgeClass(c.status)}>{t('complaint_status_' + c.status)}</span>
            <span className="badge badge-soft">{t('complaint_cat_' + c.category)}</span>
          </div>
          <h1 className="h2" style={{ margin: 0 }}>
            {c.subject}
          </h1>
          {c.orderId && (
            <a className="small clickable" onClick={() => navigate(`/orders/${c.orderId}`)}>
              {t('complaint_linked_order')}: #{c.orderId.slice(-8)}
            </a>
          )}
        </div>
        <hr className="divider" />
        <div className="col" style={{ gap: 12 }}>
          {(c.messages ?? []).map((m) => (
            <Bubble key={m.id} m={m} />
          ))}
        </div>
        {c.status === 'resolved' && <p className="small muted">{t('complaint_resolved_hint')}</p>}
        {closed ? (
          <p className="small muted">{t('complaint_closed_hint')}</p>
        ) : (
          <div className="col" style={{ gap: 10 }}>
            <textarea
              className="field"
              rows={3}
              maxLength={2000}
              value={draft}
              placeholder={t('complaint_reply_ph')}
              onChange={(e) => setDraft(e.target.value)}
              style={{ resize: 'vertical' }}
            />
            {error && (
              <div className="small" style={{ color: 'var(--danger)' }}>
                {error}
              </div>
            )}
            <Button variant="primary" loading={reply.isPending} disabled={!draft.trim() || reply.isPending} onClick={send}>
              <Icon name="arrow" size={16} />
              {t('complaint_send')}
            </Button>
          </div>
        )}
      </Card>
    </div>
  )
}

/** Chat bubble: the customer's own messages on the end side, support on the start side. */
function Bubble({ m }: { m: ComplaintMessage }) {
  const { t } = useTranslation()
  const mine = m.author === 'customer'
  return (
    <div style={{ display: 'flex', justifyContent: mine ? 'flex-end' : 'flex-start' }}>
      <div
        style={{
          maxWidth: '82%',
          padding: '10px 13px',
          borderRadius: 14,
          background: mine ? 'rgba(59,91,255,.16)' : 'var(--surface-2)',
          border: '1px solid var(--border)',
        }}
      >
        <div className="tiny faint" style={{ marginBottom: 4, fontWeight: 700 }}>
          {mine ? t('complaint_you') : t('complaint_support')} · {new Date(m.createdAt).toLocaleString()}
        </div>
        <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.5 }}>{m.body}</div>
        {m.photoUrl && (
          <a href={m.photoUrl} target="_blank" rel="noreferrer" style={{ display: 'inline-block', marginTop: 8 }}>
            <img
              src={m.photoUrl}
              alt=""
              style={{ width: 140, height: 140, objectFit: 'cover', borderRadius: 10, border: '1px solid var(--border)' }}
            />
          </a>
        )}
      </div>
    </div>
  )
}
