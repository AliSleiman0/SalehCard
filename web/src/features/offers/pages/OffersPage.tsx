import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { LoadingSpinner, ErrorState, EmptyState } from '@/components'
import { useLocaleStore } from '@/stores/locale'
import { useOffers } from '../hooks/useOffers'
import { adaptOffer } from '../lib/adaptOffer'
import { OfferCard } from '../components/OfferCard'

export default function OffersPage() {
  const { t } = useTranslation()
  const locale = useLocaleStore((s) => s.locale)
  const query = useOffers()
  const offers = useMemo(
    () => (query.data ?? []).map((o) => adaptOffer(o, locale)),
    [query.data, locale],
  )

  return (
    <div className="wrap" style={{ paddingTop: 18, paddingBottom: 40 }}>
      <h1 className="h1" style={{ marginBottom: 6 }}>
        {t('offers_title')}
      </h1>
      <p className="muted small" style={{ marginBottom: 22 }}>
        {t('offers_sub')}
      </p>
      {query.isLoading ? (
        <LoadingSpinner />
      ) : query.isError ? (
        <ErrorState title={t('retry')} onRetry={() => query.refetch()} retryLabel={t('retry')} />
      ) : offers.length === 0 ? (
        <EmptyState icon="star" title={t('offers_empty')} sub={t('offers_empty_sub')} />
      ) : (
        <div className="prodgrid">
          {offers.map((o) => (
            <OfferCard key={o.id} o={o} />
          ))}
        </div>
      )}
    </div>
  )
}
