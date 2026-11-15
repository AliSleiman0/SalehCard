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
import { useCan } from '@/stores/auth'
import { usePromos, useDeletePromo } from '../hooks/usePromos'
import { adaptPromo, type PromoView } from '../lib/adaptPromo'
import type { PromoStatus, PromoType } from '../api/promos'

const TYPE_CLASS: Record<string, string> = {
  percent: 'st-info',
  fixed: 'st-ok',
  cashback: 'st-warn',
}

const TYPES: PromoType[] = ['percent', 'fixed', 'cashback']
const STATUSES: PromoStatus[] = ['active', 'paused', 'expired', 'depleted']

export default function PromoListPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const can = useCan()
  const canManage = can('promos.manage')
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

  // The labels (period, value) follow the language, so re-adapt when it changes.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const rows = useMemo(() => (data?.data ?? []).map(adaptPromo), [data, i18n.language])
  const meta = data?.meta

  return (
    <div className="page page-wide">
      <PageHead
        back="/"
        crumbs={[t('grp_finance'), t('nav_promos')]}
        title={t('nav_promos')}
        sub={meta ? t('promo_campaigns', { n: meta.total.toLocaleString() }) : ''}
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> {t('exp_refresh')}
        </button>
        {canManage && (
          <button className="abtn primary" onClick={() => navigate('/promos/new')}>
            <Icon name="plus" size={15} /> {t('promo_new')}
          </button>
        )}
      </PageHead>

      <div className="acard">
        <div className="toolbar">
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input placeholder={t('promo_search_ph')} value={search} onChange={(e) => setSearch(e.target.value)} />
          </div>
          <select
            className="select"
            value={type}
            onChange={(e) => {
              setType(e.target.value as '' | PromoType)
              setPage(1)
            }}
          >
            <option value="">{t('promo_all_types')}</option>
            {TYPES.map((k) => (
              <option key={k} value={k}>
                {t(`promo_type_${k}`)}
              </option>
            ))}
          </select>
          <select
            className="select"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value as '' | PromoStatus)
              setPage(1)
            }}
          >
            <option value="">{t('promo_all_statuses')}</option>
            {STATUSES.map((k) => (
              <option key={k} value={k}>
                {t(`promo_status_${k}`)}
              </option>
            ))}
          </select>
        </div>

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message={t('promo_load_error')} onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title={t('promo_empty')} />
        ) : (
          <>
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>{t('promo_col_code')}</th>
                    <th>{t('col_type')}</th>
                    <th>{t('promo_col_value')}</th>
                    <th>{t('promo_col_usage')}</th>
                    <th>{t('promo_col_period')}</th>
                    <th>{t('status')}</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((p) => {
                    const cls = TYPE_CLASS[p.type] ?? 'st-mute'
                    const label = t(`promo_type_${p.type}`, { defaultValue: p.type })
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
                            {canManage && (
                              <span className="iact danger" onClick={() => setToDelete(p)}>
                                <Icon name="trash" size={15} />
                              </span>
                            )}
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
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 10 }}>{t('promo_delete_title')}</h3>
        <p style={{ fontSize: 13.5, color: 'var(--text-dim)', marginBottom: 18 }}>
          {t('promo_delete_pre')} <b className="mono">{promo.code}</b>
          {t('promo_delete_post')}
        </p>
        {del.isError && (
          <div style={{ color: 'var(--danger)', fontSize: 12.5, marginBottom: 12 }}>{t('promo_delete_error')}</div>
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
