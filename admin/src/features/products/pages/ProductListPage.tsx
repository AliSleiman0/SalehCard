import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  Icon,
  PageHead,
  Art,
  FfBadge,
  StatusBadge,
  Checkbox,
  Chip,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
  artForCategory,
  ffKey,
  type FfKey,
} from '@/components'
import { useBulk } from '@/hooks/useBulk'
import { CATS } from '@/lib/mock/demo'
import { useProducts, useDeleteProduct, useBulkProductAction } from '../hooks/useProducts'
import { productStatus, priceRange } from '../lib/view'
import type { FulfillmentType, Product } from '@/types'

const FF_DOT: Record<FfKey, string> = {
  code: 'var(--ff-code)',
  credit: 'var(--ff-credit)',
  transfer: 'var(--ff-transfer)',
}

function toFulfillment(ff: FfKey): FulfillmentType {
  return ff === 'credit' ? 'account_credit' : ff === 'transfer' ? 'transfer' : 'code'
}

export default function ProductListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [ff, setFf] = useState<'all' | FfKey>('all')
  const [cat, setCat] = useState('all')
  const [search, setSearch] = useState('')

  const { data, isLoading, isError, refetch } = useProducts({
    category: cat === 'all' ? undefined : cat,
    fulfillmentType: ff === 'all' ? undefined : toFulfillment(ff),
    search: search.trim() || undefined,
    limit: 100,
  })
  const del = useDeleteProduct()
  const bulkAction = useBulkProductAction()

  const rows: Product[] = data?.data ?? []
  const ids = rows.map((r) => r.id)
  const bulk = useBulk(ids)

  const stockCell = (p: Product) => {
    const ff = ffKey(p.fulfillmentType)
    if (ff === 'transfer') return <span className="faint">— service —</span>
    if (ff === 'credit')
      return (
        <span className="st st-ok" style={{ fontSize: 11 }}>
          <i className="d" />
          Unlimited
        </span>
      )
    const lvl = p.stock === 0 ? 'lo' : p.stock < 50 ? 'mid' : 'hi'
    const pct = Math.min(100, (p.stock / 250) * 100)
    return (
      <span className={'stock ' + lvl}>
        <div className="bar">
          <i style={{ width: (p.stock === 0 ? 4 : pct) + '%' }} />
        </div>
        <span className="num">{p.stock}</span>
      </span>
    )
  }

  const runBulk = (action: 'activate' | 'deactivate' | 'delete') => {
    if (action === 'delete' && !window.confirm(`Delete ${bulk.sel.length} product(s)? This cannot be undone.`)) return
    bulkAction.mutate({ ids: bulk.sel, action }, { onSuccess: () => bulk.clear() })
  }

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_catalog'), t('nav_products')]}
        title={t('nav_products')}
        sub={`${rows.length} products${cat !== 'all' || ff !== 'all' || search ? ' (filtered)' : ''}`}
      >
        <button className="abtn">
          <Icon name="download" size={15} /> {t('export')}
        </button>
        <button className="abtn primary" onClick={() => navigate('/products/new')}>
          <Icon name="plus" size={15} /> {t('new_product')}
        </button>
      </PageHead>

      <div className="acard">
        <div className="toolbar">
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input
              placeholder="Search products…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <select className="select" value={cat} onChange={(e) => setCat(e.target.value)}>
            <option value="all">All categories</option>
            {CATS.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
          <div className="chiprow">
            {(
              [
                ['all', t('all')],
                ['code', t('ff_code')],
                ['credit', t('ff_credit')],
                ['transfer', t('ff_transfer')],
              ] as [string, string][]
            ).map(([k, l]) => (
              <Chip
                key={k}
                on={ff === k}
                onClick={() => setFf(k as 'all' | FfKey)}
                dotColor={k !== 'all' ? FF_DOT[k as FfKey] : undefined}
              >
                {l}
              </Chip>
            ))}
          </div>
          <div className="tb-spacer" />
          <button className="abtn sm">
            <Icon name="filter" size={14} /> {t('filter')}
          </button>
        </div>

        {bulk.some && (
          <div className="bulkbar">
            <Checkbox on onClick={bulk.clear} />
            <span>
              {bulk.sel.length} {t('selected')}
            </span>
            <div className="ba-act">
              <button className="abtn xs ok" onClick={() => runBulk('activate')}>
                <Icon name="check" size={13} /> {t('activate')}
              </button>
              <button className="abtn xs" onClick={() => runBulk('deactivate')}>
                <Icon name="eyeoff" size={13} /> {t('deactivate')}
              </button>
              <button className="abtn xs danger" onClick={() => runBulk('delete')}>
                <Icon name="trash" size={13} /> {t('delete')}
              </button>
            </div>
          </div>
        )}

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message="Could not load products" onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title="No products found" sub="Try clearing filters or create a new product." />
        ) : (
          <div className="tablewrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th style={{ width: 36 }}>
                    <Checkbox on={bulk.all} onClick={bulk.toggleAll} />
                  </th>
                  <th>Product</th>
                  <th>Category</th>
                  <th>Type</th>
                  <th>Variants</th>
                  <th>Stock / codes</th>
                  <th>Price (USD)</th>
                  <th>Sold</th>
                  <th>{t('status')}</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {rows.map((p) => (
                  <tr key={p.id} className="clickable" onClick={() => navigate(`/products/${p.id}/edit`)}>
                    <td onClick={(e) => e.stopPropagation()}>
                      <Checkbox on={bulk.sel.includes(p.id)} onClick={() => bulk.toggle(p.id)} />
                    </td>
                    <td>
                      <div className="cellprod">
                        <Art art={artForCategory(p.category)} size={36} />
                        <div className="pn">
                          <b>{p.title.en}</b>
                          <span className="mono">{p.id.slice(-8)}</span>
                        </div>
                      </div>
                    </td>
                    <td className="muted">{p.category}</td>
                    <td>
                      <FfBadge ff={ffKey(p.fulfillmentType)} />
                    </td>
                    <td className="num">{p.variants.length}</td>
                    <td>{stockCell(p)}</td>
                    <td className="num strong">{priceRange(p) === '—' ? <span className="faint">—</span> : '$' + priceRange(p)}</td>
                    <td className="num muted">{p.ratings.count ? p.ratings.count.toLocaleString() : '—'}</td>
                    <td>
                      <StatusBadge s={productStatus(p)} />
                    </td>
                    <td onClick={(e) => e.stopPropagation()}>
                      <div className="row-actions">
                        <span className="iact" onClick={() => navigate(`/products/${p.id}/edit`)}>
                          <Icon name="edit" size={15} />
                        </span>
                        <span
                          className="iact danger"
                          onClick={() => {
                            if (window.confirm(`Delete "${p.title.en}"?`)) del.mutate(p.id)
                          }}
                        >
                          <Icon name="trash" size={15} />
                        </span>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <Pagination total={data?.meta?.total ?? rows.length} pages={data?.meta?.pages ?? 1} shown={rows.length} label="products" />
      </div>
    </div>
  )
}
