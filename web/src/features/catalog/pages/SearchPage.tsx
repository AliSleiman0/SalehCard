import { useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Icon, LoadingSpinner, ErrorState, EmptyState } from '@/components'
import { useLocaleStore } from '@/stores/locale'
import { useDebouncedValue } from '@/lib/useDebouncedValue'
import { useProducts } from '../hooks/useProducts'
import { adaptProduct } from '../lib/adaptProduct'
import { ProductGrid } from '../components/ProductGrid'

export default function SearchPage() {
  const { t } = useTranslation()
  const locale = useLocaleStore((s) => s.locale)
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''
  const debounced = useDebouncedValue(q.trim(), 350)

  const query = useProducts(
    { q: debounced, limit: 50 },
    { enabled: debounced.length > 0 },
  )
  const items = useMemo(
    () => (query.data?.data ?? []).map((p) => adaptProduct(p, locale)),
    [query.data, locale],
  )

  return (
    <div className="wrap" style={{ paddingTop: 18, paddingBottom: 40 }}>
      <div className="searchbar" style={{ maxWidth: 560, marginBottom: 22 }}>
        <Icon name="search" size={18} />
        <input
          autoFocus
          placeholder={t('search_ph')}
          value={q}
          onChange={(e) => setParams(e.target.value ? { q: e.target.value } : {}, { replace: true })}
        />
      </div>

      {debounced.length === 0 ? (
        <EmptyState icon="search" title={t('search_prompt')} sub={t('search_prompt_sub')} />
      ) : query.isLoading ? (
        <LoadingSpinner />
      ) : query.isError ? (
        <ErrorState title={t('retry')} onRetry={() => query.refetch()} retryLabel={t('retry')} />
      ) : items.length === 0 ? (
        <EmptyState icon="search" title={t('no_results')} sub={t('no_results_sub')} />
      ) : (
        <>
          <p className="muted small" style={{ marginBottom: 16 }}>
            {items.length} {t('search_results')}
          </p>
          <ProductGrid products={items} />
        </>
      )}
    </div>
  )
}
