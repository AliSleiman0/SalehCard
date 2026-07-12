import { useTranslation } from 'react-i18next'
import { LoadingSpinner, ErrorState, EmptyState } from '@/components'
import { useLocaleStore } from '@/stores/locale'
import { useCategories } from '../hooks/useCategories'
import { adaptRootCategory } from '../lib/adaptCategory'
import { CategoryTile } from '../components/CategoryTile'

// CategoriesPage is the all-categories index (/categories) — the storefront's
// primary "browse" landing, reached from the Shop pill, the account sidebar's
// "Browse store" link, and the mobile Categories tab.
export default function CategoriesPage() {
  const { t } = useTranslation()
  const locale = useLocaleStore((s) => s.locale)
  const query = useCategories({ depth: 0, withCounts: true })
  const cats = (query.data?.data ?? [])
    .map((c) => adaptRootCategory(c, locale))
    .sort((a, b) => a.order - b.order)

  return (
    <div className="wrap" style={{ paddingTop: 18, paddingBottom: 40 }}>
      <div className="section-head">
        <h1 className="h1">{t('shop_cat')}</h1>
      </div>
      {query.isLoading ? (
        <LoadingSpinner />
      ) : query.isError ? (
        <ErrorState title={t('retry')} onRetry={() => query.refetch()} retryLabel={t('retry')} />
      ) : cats.length === 0 ? (
        <EmptyState icon="grid" title={t('no_results')} sub={t('no_results_sub')} />
      ) : (
        <div className="catgrid">
          {cats.map((c) => (
            <CategoryTile key={c.key} c={c} />
          ))}
        </div>
      )}
    </div>
  )
}
