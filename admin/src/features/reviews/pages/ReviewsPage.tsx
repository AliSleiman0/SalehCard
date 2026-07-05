import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import {
  Icon,
  PageHead,
  Art,
  Stars,
  StatusBadge,
  Checkbox,
  Chip,
  Modal,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { useBulk } from '@/hooks/useBulk'
import { useReviews, useSetReviewStatus, useDeleteReview } from '../hooks/useReviews'
import { adaptReview, type ReviewView } from '../lib/adaptReview'
import type { ReviewStatus } from '../api/reviews'

const FILTERS: [ReviewStatus | '', string][] = [
  ['pending', 'Pending'],
  ['approved', 'Approved'],
  ['rejected', 'Rejected'],
  ['', 'All'],
]

export default function ReviewsPage() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()

  const [status, setStatus] = useState<ReviewStatus | ''>('pending')
  const [search, setSearch] = useState(searchParams.get('q') ?? '')
  const [debouncedSearch, setDebouncedSearch] = useState(search)
  const [page, setPage] = useState(1)
  const [toDelete, setToDelete] = useState<ReviewView | null>(null)

  // Adopt the URL's ?q= (e.g. the top-bar global search navigates to /reviews?q=…).
  useEffect(() => {
    setSearch(searchParams.get('q') ?? '')
  }, [searchParams])

  // Debounce the search term so we issue one request per pause, not per keystroke.
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  const { data, isLoading, isError, refetch } = useReviews({
    page,
    status: status || undefined,
    q: debouncedSearch.trim(),
  })

  const rows = useMemo(() => (data?.data ?? []).map(adaptReview), [data])
  const meta = data?.meta

  const setStatusM = useSetReviewStatus()
  const bulk = useBulk(rows.map((r) => r.id))

  const moderate = (id: string, s: ReviewStatus) => setStatusM.mutate({ id, status: s })
  const bulkModerate = (s: ReviewStatus) => {
    Promise.all(bulk.sel.map((id) => setStatusM.mutateAsync({ id, status: s }))).finally(() => bulk.clear())
  }

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_content'), t('nav_reviews')]}
        title="Review moderation"
        sub={meta ? `${meta.total.toLocaleString()} reviews` : ''}
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
            <input placeholder="Search reviews…" value={search} onChange={(e) => setSearch(e.target.value)} />
          </div>
        </div>

        {bulk.some && (
          <div className="bulkbar">
            <Checkbox on onClick={bulk.clear} />
            <span>
              {bulk.sel.length} {t('selected')}
            </span>
            <div className="ba-act">
              <button className="abtn xs ok" disabled={setStatusM.isPending} onClick={() => bulkModerate('approved')}>
                <Icon name="check" size={13} /> {t('approve')}
              </button>
              <button className="abtn xs danger" disabled={setStatusM.isPending} onClick={() => bulkModerate('rejected')}>
                <Icon name="x" size={13} /> {t('reject')}
              </button>
            </div>
          </div>
        )}

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message="Couldn't load reviews." onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title="No reviews found" />
        ) : (
          <>
            <div>
              {rows.map((r) => (
                <div key={r.id} style={{ borderBottom: '1px solid var(--border)' }}>
                  <div style={{ display: 'flex', gap: 14, padding: '14px 18px', alignItems: 'flex-start' }}>
                    <div style={{ paddingTop: 3 }}>
                      <Checkbox on={bulk.sel.includes(r.id)} onClick={() => bulk.toggle(r.id)} />
                    </div>
                    <Art art={r.art} size={42} radius={10} />
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 4, flexWrap: 'wrap' }}>
                        <b style={{ fontSize: 13.5 }}>{r.product}</b>
                        <Stars n={r.rating} size={13} />
                        <span className="faint" style={{ fontSize: 12 }}>
                          by {r.user} · {r.date}
                        </span>
                        {r.verified && (
                          <span className="st st-ok" style={{ fontSize: 11 }}>
                            <i className="d" />
                            Verified
                          </span>
                        )}
                        <span style={{ marginInlineStart: 'auto' }}>
                          <StatusBadge s={r.status} />
                        </span>
                      </div>
                      <p style={{ fontSize: 13.5, color: 'var(--text-dim)', lineHeight: 1.5, maxWidth: 760 }}>{r.body}</p>
                      <div style={{ display: 'flex', gap: 8, marginTop: 10, alignItems: 'center' }}>
                        {r.status !== 'approved' && (
                          <button className="abtn xs ok" disabled={setStatusM.isPending} onClick={() => moderate(r.id, 'approved')}>
                            <Icon name="check" size={13} /> {t('approve')}
                          </button>
                        )}
                        {r.status !== 'rejected' && (
                          <button className="abtn xs danger" disabled={setStatusM.isPending} onClick={() => moderate(r.id, 'rejected')}>
                            <Icon name="x" size={13} /> {t('reject')}
                          </button>
                        )}
                        <button className="abtn xs" onClick={() => setToDelete(r)}>
                          <Icon name="trash" size={13} /> {t('delete')}
                        </button>
                        {r.flagged && (
                          <span className="st st-danger" style={{ fontSize: 11 }}>
                            <i className="d" />
                            Flagged · possible spam
                          </span>
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
              label={t('pg_reviews')}
              onPage={setPage}
            />
          </>
        )}
      </div>

      {toDelete && <DeleteModal review={toDelete} onClose={() => setToDelete(null)} />}
    </div>
  )
}

/** Confirmation dialog for deleting a review. */
function DeleteModal({ review, onClose }: { review: ReviewView; onClose: () => void }) {
  const { t } = useTranslation()
  const del = useDeleteReview()

  return (
    <Modal onClose={onClose} maxWidth={420}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 10 }}>Delete review</h3>
        <p style={{ fontSize: 13.5, color: 'var(--text-dim)', marginBottom: 18 }}>
          Delete this review by <b>{review.user}</b> on <b>{review.product}</b>? This cannot be undone.
        </p>
        {del.isError && (
          <div style={{ color: 'var(--danger)', fontSize: 12.5, marginBottom: 12 }}>Couldn't delete this review.</div>
        )}
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={del.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn danger" disabled={del.isPending} onClick={() => del.mutate(review.id, { onSuccess: onClose })}>
            <Icon name="trash" size={15} /> {t('delete')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
