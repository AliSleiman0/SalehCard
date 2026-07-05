import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  Icon,
  PageHead,
  StatusBadge,
  Modal,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { usePromos, useDeletePromo } from '../hooks/usePromos'
import { adaptPromo, type PromoView } from '../lib/adaptPromo'
import type { PromoStatus, PromoType } from '../api/promos'

const TYPE_BADGE: Record<string, [string, string]> = {
  percent: ['st-info', 'Percent'],
  fixed: ['st-ok', 'Fixed'],
  cashback: ['st-warn', 'Cashback'],
}

export default function PromoListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const [type, setType] = useState<'' | PromoType>('')
  const [status, setStatus] = useState<'' | PromoStatus>('')
  const [search, setSearch] = useState(searchParams.get('q') ?? '')
  const [debouncedSearch, setDebouncedSearch] = useState(search)
  const [page, setPage] = useState(1)
  const [toDelete, setToDelete] = useState<PromoView | null>(null)

  // Adopt the URL's ?q= (e.g. the top-bar global search navigates to /promos?q=…).
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

  const { data, isLoading, isError, refetch } = usePromos({
    page,
    type: type || undefined,
    status: status || undefined,
    q: debouncedSearch.trim(),
  })

  const rows = useMemo(() => (data?.data ?? []).map(adaptPromo), [data])
  const meta = data?.meta

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_finance'), t('nav_promos')]}
        title="Promo codes"
        sub={meta ? `${meta.total.toLocaleString()} campaigns` : ''}
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> Refresh
        </button>
        <button className="abtn primary" onClick={() => navigate('/promos/new')}>
          <Icon name="plus" size={15} /> New promo
        </button>
      </PageHead>

      <div className="acard">
        <div className="toolbar">
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input placeholder="Search codes…" value={search} onChange={(e) => setSearch(e.target.value)} />
          </div>
          <select
            className="select"
            value={type}
            onChange={(e) => {
              setType(e.target.value as '' | PromoType)
              setPage(1)
            }}
          >
            <option value="">All types</option>
            <option value="percent">Percent</option>
            <option value="fixed">Fixed</option>
            <option value="cashback">Cashback</option>
          </select>
          <select
            className="select"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value as '' | PromoStatus)
              setPage(1)
            }}
          >
            <option value="">All statuses</option>
            <option value="active">Active</option>
            <option value="paused">Paused</option>
            <option value="expired">Expired</option>
            <option value="depleted">Depleted</option>
          </select>
        </div>

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message="Couldn't load promo codes." onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title="No promo codes found" />
        ) : (
          <>
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>Code</th>
                    <th>Type</th>
                    <th>Value</th>
                    <th>Usage</th>
                    <th>Valid period</th>
                    <th>{t('status')}</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((p) => {
                    const [cls, label] = TYPE_BADGE[p.type] ?? ['st-mute', p.type]
                    return (
                      <tr key={p.id} className="clickable" onClick={() => navigate(`/promos/${p.id}/edit`)}>
                        <td>
                          <span
                            className="mono strong"
                            style={{
                              fontSize: 13.5,
                              padding: '3px 8px',
                              background: 'var(--surface-2)',
                              borderRadius: 6,
                              border: '1px dashed var(--border-strong)',
                            }}
                          >
                            {p.code}
                          </span>
                        </td>
                        <td>
                          <span className={'st ' + cls}>
                            <i className="d" />
                            {label}
                          </span>
                        </td>
                        <td className="num strong">{p.valueLabel}</td>
                        <td>
                          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                            <span className="num muted" style={{ minWidth: 78, fontSize: 12.5 }}>
                              {p.used.toLocaleString()} / {p.unlimited ? '∞' : p.limit.toLocaleString()}
                            </span>
                            <div className="meter" style={{ width: 70 }}>
                              <i
                                style={{
                                  width: p.usagePct + '%',
                                  background: p.usagePct > 90 ? 'var(--danger)' : 'var(--grad)',
                                }}
                              />
                            </div>
                          </div>
                        </td>
                        <td className="muted" style={{ fontSize: 12.5 }}>
                          {p.period}
                        </td>
                        <td>
                          <StatusBadge s={p.status} />
                        </td>
                        <td onClick={(e) => e.stopPropagation()}>
                          <div className="row-actions">
                            <span className="iact" onClick={() => navigate(`/promos/${p.id}/edit`)}>
                              <Icon name="edit" size={15} />
                            </span>
                            <span className="iact danger" onClick={() => setToDelete(p)}>
                              <Icon name="trash" size={15} />
                            </span>
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
            <Pagination
              page={meta?.page ?? 1}
              pages={meta?.pages ?? 1}
              total={meta?.total ?? rows.length}
              shown={rows.length}
              limit={meta?.limit}
              label={t('pg_promos')}
              onPage={setPage}
            />
          </>
        )}
      </div>

      {toDelete && <DeleteModal promo={toDelete} onClose={() => setToDelete(null)} />}
    </div>
  )
}

/** Confirmation dialog for deleting a promo code. */
function DeleteModal({ promo, onClose }: { promo: PromoView; onClose: () => void }) {
  const { t } = useTranslation()
  const del = useDeletePromo()

  return (
    <Modal onClose={onClose} maxWidth={420}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 10 }}>Delete promo</h3>
        <p style={{ fontSize: 13.5, color: 'var(--text-dim)', marginBottom: 18 }}>
          Delete <b className="mono">{promo.code}</b>? This cannot be undone.
        </p>
        {del.isError && (
          <div style={{ color: 'var(--danger)', fontSize: 12.5, marginBottom: 12 }}>Couldn't delete this promo.</div>
        )}
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={del.isPending}>
            {t('cancel')}
          </button>
          <button
            className="abtn danger"
            disabled={del.isPending}
            onClick={() => del.mutate(promo.id, { onSuccess: onClose })}
          >
            <Icon name="trash" size={15} /> {t('delete')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
