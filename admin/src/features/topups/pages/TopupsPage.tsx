import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Icon,
  PageHead,
  Chip,
  Modal,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
  RoleBadge,
} from '@/components'
import { money } from '@/lib/utils'
import { ApiError } from '@/lib/api-client'
import { useCan } from '@/stores/auth'
import { useTopUps, useApproveTopUp, useRejectTopUp } from '../hooks/useTopups'
import type { AdminTopUp, TopUpStatus } from '../api/topups'

const FILTERS: [TopUpStatus | '', string][] = [
  ['pending', 'Pending'],
  ['approved', 'Approved'],
  ['rejected', 'Rejected'],
  ['', 'All'],
]

const STATUS_CLASS: Record<TopUpStatus, string> = {
  pending: 'st st-warn',
  approved: 'st st-ok',
  rejected: 'st st-danger',
}

/** isDocUrl detects an uploaded top-up document (re-encoded JPEG in our keyspace). */
function isDocUrl(v: string): boolean {
  return /^https?:\/\//.test(v) && v.includes('/topups/') && v.endsWith('.jpg')
}

export default function TopupsPage() {
  const { t } = useTranslation()
  const can = useCan()
  const canManage = can('topups.manage')
  const [status, setStatus] = useState<TopUpStatus | ''>('pending')
  const [page, setPage] = useState(1)
  const [toReject, setToReject] = useState<AdminTopUp | null>(null)

  const { data, isLoading, isError, refetch } = useTopUps({ page, status: status || undefined })
  const rows = useMemo(() => data?.data ?? [], [data])
  const meta = data?.meta

  const approveM = useApproveTopUp()
  const [approveError, setApproveError] = useState('')
  const approve = (id: string) => {
    setApproveError('')
    approveM.mutate(id, {
      onError: (e) => setApproveError(e instanceof ApiError ? e.message : 'Approval failed.'),
    })
  }

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_finance'), 'Top-up requests']}
        title="Top-up requests"
        sub={meta ? `${meta.total.toLocaleString()} requests` : ''}
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> Refresh
        </button>
      </PageHead>

      <div className="acard">
        <div className="toolbar">
          <div className="chiprow">
            {FILTERS.map(([k, l]) => (
              <Chip
                key={k || 'all'}
                on={status === k}
                onClick={() => {
                  setStatus(k)
                  setPage(1)
                }}
              >
                {l}
              </Chip>
            ))}
          </div>
        </div>

        {approveError && (
          <div style={{ padding: '10px 18px', color: 'var(--danger)', fontSize: 13 }}>{approveError}</div>
        )}

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message="Couldn't load top-up requests." onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState
            title="No top-up requests"
            sub="Customer wallet-funding requests appear here for approval after you receive their payment."
          />
        ) : (
          <>
            <div>
              {rows.map((r) => (
                <div key={r.id} style={{ borderBottom: '1px solid var(--border)', padding: '14px 18px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                    <b style={{ fontSize: 15 }}>{money(r.amount)}</b>
                    <span className="bdg">{(r.methodName || r.channel).toUpperCase()}</span>
                    <span className="faint" style={{ fontSize: 12.5 }}>
                      {r.customerEmail || r.customerPhone || r.userId.slice(-8)}
                    </span>
                    {r.role && <RoleBadge role={r.role} />}
                    <span className="faint" style={{ fontSize: 12 }}>
                      · {new Date(r.createdAt).toLocaleString()}
                    </span>
                    <span style={{ marginInlineStart: 'auto' }} className={STATUS_CLASS[r.status]}>
                      <i className="d" />
                      {r.status}
                    </span>
                  </div>
                  {r.fields && r.fields.length > 0 && (
                    <div
                      style={{
                        display: 'flex',
                        flexWrap: 'wrap',
                        gap: 16,
                        marginTop: 10,
                        alignItems: 'flex-start',
                      }}
                    >
                      {r.fields.map((f) => (
                        <div key={f.key} style={{ fontSize: 12.5 }}>
                          <div className="faint" style={{ marginBottom: 3 }}>
                            {f.label}
                          </div>
                          {isDocUrl(f.value) ? (
                            <a href={f.value} target="_blank" rel="noreferrer">
                              <img
                                src={f.value}
                                alt={f.label}
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
                            <b>{f.value}</b>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                  {r.note && (
                    <div style={{ fontSize: 13, color: 'var(--text-dim)', marginTop: 6 }}>
                      Note: {r.note}
                    </div>
                  )}
                  {r.status === 'rejected' && r.decisionReason && (
                    <div style={{ fontSize: 12.5, color: 'var(--danger)', marginTop: 6 }}>
                      Rejected: {r.decisionReason}
                    </div>
                  )}
                  {r.status !== 'pending' && r.decidedBy && (
                    <div className="faint" style={{ fontSize: 12, marginTop: 4 }}>
                      Decided by {r.decidedBy}
                      {r.decidedAt ? ` · ${new Date(r.decidedAt).toLocaleString()}` : ''}
                    </div>
                  )}
                  {canManage && r.status === 'pending' && (
                    <div style={{ display: 'flex', gap: 8, marginTop: 10 }}>
                      <button
                        className="abtn xs ok"
                        disabled={approveM.isPending}
                        onClick={() => approve(r.id)}
                      >
                        <Icon name="check" size={13} /> Approve & credit
                      </button>
                      <button
                        className="abtn xs danger"
                        disabled={approveM.isPending}
                        onClick={() => setToReject(r)}
                      >
                        <Icon name="x" size={13} /> {t('reject')}
                      </button>
                    </div>
                  )}
                </div>
              ))}
            </div>
            <Pagination
              page={meta?.page ?? 1}
              pages={meta?.pages ?? 1}
              total={meta?.total ?? rows.length}
              shown={rows.length}
              limit={meta?.limit}
              label={t('pg_requests')}
              onPage={setPage}
            />
          </>
        )}
      </div>

      {toReject && <RejectModal req={toReject} onClose={() => setToReject(null)} />}
    </div>
  )
}

/** Confirmation dialog for rejecting a request with a required reason. */
function RejectModal({ req, onClose }: { req: AdminTopUp; onClose: () => void }) {
  const { t } = useTranslation()
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const rejectM = useRejectTopUp()

  const apply = () => {
    if (!reason.trim()) {
      setError('A rejection reason is required.')
      return
    }
    setError('')
    rejectM.mutate(
      { id: req.id, reason: reason.trim() },
      {
        onSuccess: onClose,
        onError: (e) => setError(e instanceof ApiError ? e.message : 'Rejection failed.'),
      },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={440}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>Reject top-up request</h3>
        <p style={{ fontSize: 13.5, color: 'var(--text-dim)', marginBottom: 16 }}>
          Reject the <b>{money(req.amount)}</b> request from <b>{req.customerEmail || 'customer'}</b>.
          The reason is shown to the customer.
        </p>
        <label className="alabel">Reason (required)</label>
        <input
          className="afield"
          placeholder="e.g. no matching payment received…"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 18, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={rejectM.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn danger" onClick={apply} disabled={rejectM.isPending}>
            <Icon name="x" size={15} /> {t('reject')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
