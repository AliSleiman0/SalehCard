import { useEffect, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
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
import { apiErrorText } from '@/lib/apiErrorText'
import { useCan } from '@/stores/auth'
import { useKeyedPage } from '@/hooks/useKeyedPage'
import { useKyc, useSetKycStatus } from '../hooks/useKyc'
import { adaptKyc, type KycView } from '../lib/adaptKyc'
import type { KycStatus } from '../api/kyc'

// [status, translation key]
const FILTERS: [KycStatus | '', string][] = [
  ['pending', 'pending'],
  ['approved', 'kyc_filter_approved'],
  ['rejected', 'kyc_filter_rejected'],
  ['', 'all'],
]

function statusFromParam(raw: string | null): KycStatus | '' {
  if (raw === 'all') return ''
  return FILTERS.some(([k]) => k && k === raw) ? (raw as KycStatus) : 'pending'
}

export default function KycPage() {
  const { t } = useTranslation()
  const can = useCan()
  const canManage = can('kyc.manage')
  // The status filter lives in the URL (?status=pending|approved|rejected|all)
  // so dashboard tiles can deep-link to it and a refresh keeps it. Absent or
  // unknown → pending (the work queue); "all" → every status.
  const [searchParams, setSearchParams] = useSearchParams()
  const status = statusFromParam(searchParams.get('status'))
  const setStatus = (k: KycStatus | '') =>
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        next.set('status', k || 'all')
        return next
      },
      { replace: true },
    )
  const [search, setSearch] = useState(searchParams.get('q') ?? '')
  const [debouncedSearch, setDebouncedSearch] = useState(search)
  const [toReject, setToReject] = useState<KycView | null>(null)

  // Adopt the URL's ?q= — keyed on the value, not the params object, so a
  // status-chip URL write doesn't wipe a search typed into the box.
  const qParam = searchParams.get('q') ?? ''
  useEffect(() => {
    setSearch(qParam)
  }, [qParam])


  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  // A status change (chip, dashboard deep link, sidebar) or a new search starts
  // at page 1 — in the same render, so no request goes out for a stale page.
  const [page, setPage] = useKeyedPage(`${status}|${debouncedSearch.trim()}`)

  const { data, isLoading, isError, refetch } = useKyc({
    page,
    status: status || undefined,
    q: debouncedSearch.trim(),
  })

  // Adapted on every render (cheap) so labels and dates follow a language switch.
  const rows = (data?.data ?? []).map(adaptKyc)
  const meta = data?.meta

  const setStatusM = useSetKycStatus()
  const approve = (id: string) => setStatusM.mutate({ id, status: 'approved' })

  return (
    <div className="page page-wide">
      <PageHead
        back="/"
        crumbs={[t('grp_operations'), t('nav_kyc')]}
        title={t('kyc_title')}
        sub={meta ? t('kyc_sub', { n: meta.total.toLocaleString() }) : ''}
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> {t('exp_refresh')}
        </button>
      </PageHead>

      <div className="acard">
        <div className="toolbar">
          <div className="chiprow">
            {FILTERS.map(([k, l]) => (
              <Chip
                key={k || 'all'}
                on={status === k}
                onClick={() => setStatus(k)}
              >
                {t(l)}
              </Chip>
            ))}
          </div>
          <div className="tb-spacer" />
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input
              placeholder={t('kyc_search_ph')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
        </div>

        {isLoading ? (
          <LoadingSpinner />
        ) : isError && !data ? (
          <ErrorState message={t('kyc_load_error')} onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title={t('kyc_empty')} />
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
                          {k.contact} · {t('kyc_submitted', { date: k.submitted })}
                        </span>
                        <span style={{ marginInlineStart: 'auto' }}>
                          <StatusBadge s={k.status} />
                        </span>
                      </div>
                      <div style={{ fontSize: 13, color: 'var(--text-dim)', lineHeight: 1.6 }}>
                        <span>{t('kyc_born', { date: k.dateOfBirth, place: k.placeOfBirth })}</span>
                        {' · '}
                        <span>{t('kyc_lives_in', { place: k.placeOfResidence })}</span>
                        {' · '}
                        <span>
                          {k.documentLabel} <b className="mono">{k.documentNumber}</b>
                        </span>
                      </div>
                      {(k.documentFrontUrl || k.documentBackUrl) && (
                        <div style={{ display: 'flex', gap: 10, marginTop: 10 }}>
                          {k.documentFrontUrl && <DocThumb url={k.documentFrontUrl} label={t('kyc_front')} />}
                          {k.documentBackUrl && <DocThumb url={k.documentBackUrl} label={t('kyc_back')} />}
                        </div>
                      )}
                      {k.status === 'rejected' && k.rejectionReason && (
                        <div style={{ fontSize: 12.5, color: 'var(--danger)', marginTop: 6 }}>
                          {t('kyc_rejected_reason', { reason: k.rejectionReason })}
                        </div>
                      )}
                      <div style={{ display: 'flex', gap: 8, marginTop: 10, alignItems: 'center' }}>
                        {canManage && k.status !== 'approved' && (
                          <button
                            className="abtn xs ok"
                            disabled={setStatusM.isPending}
                            onClick={() => approve(k.id)}
                          >
                            <Icon name="check" size={13} /> {t('approve')}
                          </button>
                        )}
                        {canManage && k.status !== 'rejected' && (
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
              label={t('pg_submissions')}
              onPage={setPage}
            />
          </>
        )}
      </div>

      {toReject && <RejectModal kyc={toReject} onClose={() => setToReject(null)} />}
    </div>
  )
}

/** Labeled document-photo thumbnail; clicking opens the full image in a new tab. */
function DocThumb({ url, label }: { url: string; label: string }) {
  const { t } = useTranslation()
  return (
    <a href={url} target="_blank" rel="noreferrer" style={{ textDecoration: 'none' }}>
      <img
        src={url}
        alt={t('kyc_doc_alt', { side: label })}
        style={{
          width: 72,
          height: 48,
          objectFit: 'cover',
          borderRadius: 6,
          border: '1px solid var(--border)',
          display: 'block',
        }}
      />
      <div className="faint" style={{ fontSize: 11, textAlign: 'center', marginTop: 2 }}>
        {label}
      </div>
    </a>
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
      setError(t('kyc_err_reason'))
      return
    }
    setError('')
    setStatusM.mutate(
      { id: kyc.id, status: 'rejected', reason: reason.trim() },
      {
        onSuccess: onClose,
        onError: (e) => setError(apiErrorText(e, 'kyc_reject_failed')),
      },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={440}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>{t('kyc_reject_title')}</h3>
        <p style={{ fontSize: 13.5, color: 'var(--text-dim)', marginBottom: 16 }}>
          <Trans i18nKey="kyc_reject_body" values={{ name: kyc.fullName }} components={{ b: <b /> }} />
        </p>
        <label className="alabel">{t('usr_reason_required_label')}</label>
        <input
          className="afield"
          placeholder={t('kyc_reason_ph')}
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
