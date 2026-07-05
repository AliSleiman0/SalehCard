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
import { useOffers, useDeleteOffer } from '../hooks/useOffers'
import { adaptOffer, type OfferView } from '../lib/adaptOffer'
import type { OfferStatus } from '../api/offers'

const TYPE_BADGE: Record<string, [string, string]> = {
  percent: ['st-info', 'Percent'],
  fixed: ['st-ok', 'Fixed'],
}

export default function OffersListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const [status, setStatus] = useState<'' | OfferStatus>('')
  const [page, setPage] = useState(1)
  const [toDelete, setToDelete] = useState<OfferView | null>(null)

  const { data, isLoading, isError, refetch } = useOffers({
    page,
    status: status || undefined,
  })

  const rows = useMemo(() => (data?.data ?? []).map(adaptOffer), [data])
  const meta = data?.meta

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_catalog'), t('nav_offers')]}
        title="Offers"
        sub={meta ? `${meta.total.toLocaleString()} deals` : ''}
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> Refresh
        </button>
        <button className="abtn primary" onClick={() => navigate('/offers/new')}>
          <Icon name="plus" size={15} /> New offer
        </button>
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
            <option value="">All statuses</option>
            <option value="active">Active</option>
            <option value="scheduled">Scheduled</option>
            <option value="paused">Paused</option>
            <option value="expired">Expired</option>
          </select>
        </div>

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message="Couldn't load offers." onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title="No offers found" />
        ) : (
          <>
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>Product</th>
                    <th>Discount</th>
                    <th>Price</th>
                    <th>Valid period</th>
                    <th>{t('status')}</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((o) => {
                    const [cls, label] = TYPE_BADGE[o.discountType] ?? ['st-mute', o.discountType]
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
                            <span className="iact danger" onClick={() => setToDelete(o)}>
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
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 10 }}>Delete offer</h3>
        <p style={{ fontSize: 13.5, color: 'var(--text-dim)', marginBottom: 18 }}>
          Delete the offer on <b>{offer.productName}</b>? This cannot be undone.
        </p>
        {del.isError && (
          <div style={{ color: 'var(--danger)', fontSize: 12.5, marginBottom: 12 }}>Couldn't delete this offer.</div>
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
