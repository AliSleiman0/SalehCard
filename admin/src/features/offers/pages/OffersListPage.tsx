import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
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
import { useOffers, useDeleteOffer } from '../hooks/useOffers'
import { adaptOffer, type OfferView } from '../lib/adaptOffer'
import type { OfferStatus } from '../api/offers'

const TYPE_CLASS: Record<string, string> = {
  percent: 'st-info',
  fixed: 'st-ok',
}

const STATUS_LABEL: Record<OfferStatus, string> = {
  active: 'promo_status_active',
  scheduled: 'off_status_scheduled',
  paused: 'promo_status_paused',
  expired: 'promo_status_expired',
}

export default function OffersListPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const can = useCan()
  const canManage = can('offers.manage')

  const [status, setStatus] = useState<'' | OfferStatus>('')
  const [page, setPage] = useState(1)
  const [toDelete, setToDelete] = useState<OfferView | null>(null)

  const { data, isLoading, isError, refetch } = useOffers({
    page,
    status: status || undefined,
  })

  // The valid-period label follows the language, so re-adapt when it changes.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const rows = useMemo(() => (data?.data ?? []).map(adaptOffer), [data, i18n.language])
  const meta = data?.meta

  return (
    <div className="page page-wide">
      <PageHead
        back="/"
        crumbs={[t('grp_catalog'), t('nav_offers')]}
        title={t('nav_offers')}
        sub={meta ? t('off_deals', { n: meta.total.toLocaleString() }) : ''}
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> {t('exp_refresh')}
        </button>
        {canManage && (
          <button className="abtn primary" onClick={() => navigate('/offers/new')}>
            <Icon name="plus" size={15} /> {t('off_new')}
          </button>
        )}
      </PageHead>

      <div className="acard">
        <div className="toolbar">
          <select
            className="select"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value as '' | OfferStatus)
              setPage(1)
            }}
          >
            <option value="">{t('promo_all_statuses')}</option>
            {(Object.keys(STATUS_LABEL) as OfferStatus[]).map((k) => (
              <option key={k} value={k}>
                {t(STATUS_LABEL[k])}
              </option>
            ))}
          </select>
        </div>

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message={t('off_load_error')} onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title={t('off_empty')} />
        ) : (
          <>
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>{t('col_product')}</th>
                    <th>{t('off_col_discount')}</th>
                    <th>{t('off_col_price')}</th>
                    <th>{t('promo_col_period')}</th>
                    <th>{t('status')}</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((o) => {
                    const cls = TYPE_CLASS[o.discountType] ?? 'st-mute'
                    const label = t(`promo_type_${o.discountType}`, { defaultValue: o.discountType })
                    return (
                      <tr key={o.id} className="clickable" onClick={() => navigate(`/offers/${o.id}/edit`)}>
                        <td>
                          <div className="strong">{o.productName}</div>
                          {o.category && (
                            <div className="muted" style={{ fontSize: 12 }}>
                              {o.category}
                            </div>
                          )}
                        </td>
                        <td>
                          <span className={'st ' + cls}>
                            <i className="d" />
                            {label} {o.valueLabel}
                          </span>
                        </td>
                        <td className="num strong">{o.priceLabel}</td>
                        <td className="muted" style={{ fontSize: 12.5 }}>
                          {o.period}
                        </td>
                        <td>
                          <StatusBadge s={o.status} />
                        </td>
                        <td onClick={(e) => e.stopPropagation()}>
                          <div className="row-actions">
                            <span className="iact" onClick={() => navigate(`/offers/${o.id}/edit`)}>
                              <Icon name="edit" size={15} />
                            </span>
                            {canManage && (
                              <span className="iact danger" onClick={() => setToDelete(o)}>
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
              label={t('pg_offers')}
              onPage={setPage}
            />
          </>
        )}
      </div>

      {toDelete && <DeleteModal offer={toDelete} onClose={() => setToDelete(null)} />}
    </div>
  )
}

/** Confirmation dialog for deleting an offer. */
function DeleteModal({ offer, onClose }: { offer: OfferView; onClose: () => void }) {
  const { t } = useTranslation()
  const del = useDeleteOffer()

  return (
    <Modal onClose={onClose} maxWidth={420}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 10 }}>{t('off_delete_title')}</h3>
        <p style={{ fontSize: 13.5, color: 'var(--text-dim)', marginBottom: 18 }}>
          {t('off_delete_pre')} <b>{offer.productName}</b>
          {t('promo_delete_post')}
        </p>
        {del.isError && (
          <div style={{ color: 'var(--danger)', fontSize: 12.5, marginBottom: 12 }}>{t('off_delete_error')}</div>
        )}
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={del.isPending}>
            {t('cancel')}
          </button>
          <button
            className="abtn danger"
            disabled={del.isPending}
            onClick={() => del.mutate(offer.id, { onSuccess: onClose })}
          >
            <Icon name="trash" size={15} /> {t('delete')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
