import { Fragment, useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { LoadingSpinner, ErrorState, EmptyState } from '@/components'
import { useLocaleStore } from '@/stores/locale'
import { useCategories, useCategoryNode } from '../hooks/useCategories'
import { useNodeProducts } from '../hooks/useNodeProducts'
import { adaptProduct } from '../lib/adaptProduct'
import { crumbLink, crumbName, localizedName } from '../lib/adaptCategory'
import { ProductGrid } from '../components/ProductGrid'
import { SubcategoryGrid } from '../components/SubcategoryGrid'

// CatalogNodePage is the web twin of the app's subcategory drill-down
// (/catalog/:id): breadcrumb → the node's subcategories (if any) → its
// products. Under a node that has subcategories only the products attached to
// the node itself are listed (the rest live in the subcategories); a leaf lists
// all of its products. Keyed by id so state resets between nodes.
export default function CatalogNodePage() {
  const { id } = useParams<{ id: string }>()
  return <NodeView key={id ?? ''} id={id} />
}

function NodeView({ id }: { id: string | undefined }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const locale = useLocaleStore((s) => s.locale)
  const nodeQ = useCategoryNode(id)
  const view = nodeQ.data?.data
  const hasChildren = !!view?.node.hasChildren
  const productsQ = useNodeProducts(view ? id : undefined, hasChildren)
  // Visible subcategories (same query as SubcategoryGrid, so it is shared):
  // hasChildren also counts hidden nodes, so it alone cannot tell whether
  // the page will show any tiles.
  const childrenQ = useCategories({ parentId: view?.node.id, withCounts: true }, { enabled: hasChildren })
  const visibleChildren = hasChildren ? (childrenQ.data?.data?.length ?? 0) : 0

  const products = useMemo(
    () =>
      (productsQ.data?.pages ?? []).flatMap((p) => p.data ?? []).map((p) => adaptProduct(p, locale)),
    [productsQ.data, locale],
  )
  const total = productsQ.data?.pages[0]?.meta?.total ?? products.length

  if (nodeQ.isLoading) return <LoadingSpinner />
  if (nodeQ.isError || !view) {
    return (
      <div className="wrap" style={{ paddingTop: 40, paddingBottom: 40 }}>
        <EmptyState icon="search" title={t('no_results')} sub={t('category_unavailable')} />
      </div>
    )
  }

  const title = localizedName(view.node, locale)

  return (
    <div className="wrap" style={{ paddingTop: 18, paddingBottom: 40 }}>
      <nav className="row wrap-gap" style={{ gap: 10, marginBottom: 16 }} aria-label="breadcrumb">
        <a className="small clickable faint" onClick={() => navigate('/')}>
          {t('nav_home')}
        </a>
        {view.ancestors.map((a) => (
          <Fragment key={a.id}>
            <span className="faint">/</span>
            <a className="small clickable faint" onClick={() => navigate(crumbLink(a))}>
              {crumbName(a, locale)}
            </a>
          </Fragment>
        ))}
        <span className="faint">/</span>
        <span className="small" style={{ fontWeight: 700 }}>
          {title}
        </span>
      </nav>

      <div style={{ marginBottom: 22 }}>
        <h1 className="h1">{title}</h1>
        {/* The rolled-up count matches the list only on a leaf (under a node with
            subcategories the list shows just the node's own products). */}
        {!hasChildren && view.node.productCount !== undefined && (
          <p className="muted small">
            {view.node.productCount} {t('results')}
          </p>
        )}
      </div>

      {hasChildren && <SubcategoryGrid parentId={view.node.id} />}

      {productsQ.isLoading ? (
        <LoadingSpinner />
      ) : productsQ.isError ? (
        <ErrorState title={t('retry')} onRetry={() => productsQ.refetch()} retryLabel={t('retry')} />
      ) : products.length === 0 ? (
        // A node with subcategories and no products of its own shows just the
        // subcategory tiles; an empty leaf says so.
        visibleChildren > 0 || (hasChildren && childrenQ.isLoading) ? null : (
          <EmptyState icon="search" title={t('no_results')} sub={t('no_results_sub')} />
        )
      ) : (
        <section>
          {hasChildren && (
            <h2 className="h3" style={{ marginBottom: 12 }}>
              {t('products_in', { name: title })}
            </h2>
          )}
          <ProductGrid products={products} />
          {productsQ.hasNextPage && (
            <div style={{ display: 'grid', placeItems: 'center', marginTop: 22 }}>
              <button
                className="btn btn-ghost"
                disabled={productsQ.isFetchingNextPage}
                onClick={() => productsQ.fetchNextPage()}
              >
                {productsQ.isFetchingNextPage
                  ? t('loading')
                  : `${t('load_more')} (${products.length}/${total})`}
              </button>
            </div>
          )}
        </section>
      )}
    </div>
  )
}
