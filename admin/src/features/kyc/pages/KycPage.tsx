import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import {
  Icon,
  PageHead,
  StatusBadge,
  Chip,
  Modal,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { ApiError } from '@/lib/api-client'
import { useKyc, useSetKycStatus } from '../hooks/useKyc'
import { adaptKyc, type KycView } from '../lib/adaptKyc'
import type { KycStatus } from '../api/kyc'

const FILTERS: [KycStatus | '', string][] = [
  ['pending', 'Pending'],
  ['approved', 'Verified'],
  ['rejected', 'Rejected'],
  ['', 'All'],
]

export default function KycPage() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()

  const [status, setStatus] = useState<KycStatus | ''>('pending')
  const [search, setSearch] = useState(searchParams.get('q') ?? '')
  const [debouncedSearch, setDebouncedSearch] = useState(search)
  const [page, setPage] = useState(1)
  const [toReject, setToReject] = useState<KycView | null>(null)

  useEffect(() => {
    setSearch(searchParams.get('q') ?? '')
  }, [searchParams])

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  const { data, isLoading, isError, refetch } = useKyc({
    page,
    status: status || undefined,
    q: debouncedSearch.trim(),
  })

  const rows = useMemo(() => (data?.data ?? []).map(adaptKyc), [data])
  const meta = data?.meta

  const setStatusM = useSetKycStatus()
  const approve = (id: string) => setStatusM.mutate({ id, status: 'approved' })

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_operations'), t('nav_kyc')]}
        title="KYC verification"
        sub={meta ? `${meta.total.toLocaleString()} submissions` : ''}
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
          <div className="tb-spacer" />
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input
              placeholder="Search name or document…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
        </div>

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message="Couldn't load KYC submissions." onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title="No submissions found" />
        ) : (
          <>
            <div>
              {rows.map((k) => (
                <div key={k.id} style={{ borderBottom: '1px solid var(--border)' }}>
                  <div style={{ display: 'flex', gap: 14, padding: '14px 18px', alignItems: 'flex-start' }}>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 4, flexWrap: 'wrap' }}>
                        <b style={{ fontSize: 14 }}>{k.fullName}</b>
                        <span className="faint" style={{ fontSize: 12 }}>
                          {k.contact} · submitted {k.submitted}
                        </span>
                        <span style={{ marginInlineStart: 'auto' }}>
                          <StatusBadge s={k.status} />
                        </span>
                      </div>
                      <div style={{ fontSize: 13, color: 'var(--text-dim)', lineHeight: 1.6 }}>
                        <span>Born {k.dateOfBirth} · {k.placeOfBirth}</span>
                        {' · '}
                        <span>Lives in {k.placeOfResidence}</span>
                        {' · '}
                        <span>
                          {k.documentLabel} <b className="mono">{k.documentNumber}</b>
                        </span>
                      </div>
                      {k.status === 'rejected' && k.rejectionReason && (
                        <div style={{ fontSize: 12.5, color: 'var(--danger)', marginTop: 6 }}>
                          Rejected: {k.rejectionReason}
                        </div>
                      )}
                      <div style={{ display: 'flex', gap: 8, marginTop: 10, alignItems: 'center' }}>
                        {k.status !== 'approved' && (
                          <button
                            className="abtn xs ok"
                            disabled={setStatusM.isPending}
                            onClick={() => approve(k.id)}
                          >
                            <Icon name="check" size={13} /> {t('approve')}
                          </button>
                        )}
                        {k.status !== 'rejected' && (
                          <button
                            className="abtn xs danger"
                            disabled={setStatusM.isPending}
                            onClick={() => setToReject(k)}
                          >
                            <Icon name="x" size={13} /> {t('reject')}
                          </button>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
            <Pagination
              page={meta?.page ?? 1}
              pages={meta?.pages ?? 1}
              total={meta?.total ?? rows.length}
              shown={rows.length}
              limit={meta?.limit}
              label="submissions"
              onPage={setPage}
            />
          </>
        )}
      </div>

      {toReject && <RejectModal kyc={toReject} onClose={() => setToReject(null)} />}
    </div>
  )
}

/** Confirmation dialog for rejecting a submission with a required reason. */
function RejectModal({ kyc, onClose }: { kyc: KycView; onClose: () => void }) {
  const { t } = useTranslation()
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const setStatusM = useSetKycStatus()

  const apply = () => {
    if (!reason.trim()) {
      setError('A rejection reason is required.')
      return
    }
    setError('')
    setStatusM.mutate(
      { id: kyc.id, status: 'rejected', reason: reason.trim() },
      {
        onSuccess: onClose,
        onError: (e) => setError(e instanceof ApiError ? e.message : 'Rejection failed.'),
      },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={440}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>Reject KYC submission</h3>
        <p style={{ fontSize: 13.5, color: 'var(--text-dim)', marginBottom: 16 }}>
          Reject <b>{kyc.fullName}</b>'s verification. The reason is shown to the customer.
        </p>
        <label className="alabel">Reason (required)</label>
        <input
          className="afield"
          placeholder="e.g. Date of birth doesn't match the document…"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 18, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={setStatusM.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn danger" onClick={apply} disabled={setStatusM.isPending}>
            <Icon name="x" size={15} /> {t('reject')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
