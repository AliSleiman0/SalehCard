import { useEffect, useState } from 'react'
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
import { downloadCsv } from '@/lib/utils'
import { useBulk } from '@/hooks/useBulk'
import { useProductCategories } from '../hooks/useCategories'
import { categoryLabel } from '../api/categories'
import { useProducts, useDeleteProduct, useBulkProductAction } from '../hooks/useProducts'
import { useSoldByProduct } from '../hooks/useSoldByProduct'
import { listProducts, type ProductListParams } from '../api/products'
import { productStatus, priceRange } from '../lib/view'
import type { FulfillmentType, Product } from '@/types'

const FF_DOT: Record<FfKey, string> = {
  code: 'var(--ff-code)',
  credit: 'var(--ff-credit)',
  transfer: 'var(--ff-transfer)',
}

type StatusFilter = 'all' | 'active' | 'draft' | 'out'

// Prefers the compressed thumbnail, falling back to the display image, then to
// the gradient placeholder — both up front (no thumbnail/image) and on a
// broken URL (onError).
function ProductThumb({ p }: { p: Product }) {
  const [failed, setFailed] = useState(false)
  const src = p.thumbnail || p.images[0]
  if (!src || failed) return <Art art={artForCategory(p.category)} size={36} />
  return (
    <img
      src={src}
      alt=""
      onError={() => setFailed(true)}
      style={{ width: 36, height: 36, borderRadius: 8, objectFit: 'cover', flexShrink: 0 }}
    />
  )
}

function toFulfillment(ff: FfKey): FulfillmentType {
  return ff === 'credit' ? 'account_credit' : ff === 'transfer' ? 'transfer' : 'code'
}

export default function ProductListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [ff, setFf] = useState<'all' | FfKey>('all')
  const [cat, setCat] = useState('all')
  const [status, setStatus] = useState<StatusFilter>('all')
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [exporting, setExporting] = useState(false)

  // Debounce search so we issue one request per pause, not per keystroke; reset
  // to page 1 on each new term.
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  // Filters shared by the list query and the CSV export (page is added per call).
  const filters: ProductListParams = {
    category: cat === 'all' ? undefined : cat,
    fulfillmentType: ff === 'all' ? undefined : toFulfillment(ff),
    status: status === 'all' ? undefined : status,
    search: debouncedSearch.trim() || undefined,
  }

  const { data, isLoading, isError, refetch } = useProducts({ ...filters, page })
  const del = useDeleteProduct()
  const bulkAction = useBulkProductAction()
  const { data: catsRes } = useProductCategories()
  const { data: soldRes } = useSoldByProduct()
  const cats = catsRes?.data ?? []
  const sold = soldRes?.data ?? {}

  const rows: Product[] = data?.data ?? []
  const meta = data?.meta
  const ids = rows.map((r) => r.id)
  const bulk = useBulk(ids)

  // Reset to page 1 whenever a non-search filter changes (search resets via its debounce).
  const onFilter =
    <T,>(setter: (v: T) => void) =>
    (v: T) => {
      setter(v)
      setPage(1)
    }

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

  // Export every product matching the current filters (across all pages — the
  // backend caps limit at 100, so page through to total). CSV mirrors the table.
  const handleExport = async () => {
    if (exporting) return
    setExporting(true)
    try {
      const all: Product[] = []
      let p = 1
      let pages = 1
      do {
        const res = await listProducts({ ...filters, page: p, limit: 100 })
        all.push(...(res.data ?? []))
        pages = res.meta?.pages ?? 1
        p++
      } while (p <= pages)

      const header = ['Name', 'ID', 'Category', 'Type', 'Variants', 'Stock', 'Price min', 'Price max', 'Sold', 'Status']
      const csvRows = all.map((pr) => {
        const prices = pr.variants.map((v) => v.price)
        const min = prices.length ? Math.min(...prices) : 0
        const max = prices.length ? Math.max(...prices) : 0
        return [
          pr.title.en,
          pr.id,
          pr.category,
          ffKey(pr.fulfillmentType),
          pr.variants.length,
          pr.stock,
          min.toFixed(2),
          max.toFixed(2),
          sold[pr.id] ?? 0,
          productStatus(pr),
        ]
      })
      downloadCsv(`products-${new Date().toISOString().slice(0, 10)}.csv`, [header, ...csvRows])
    } finally {
      setExporting(false)
    }
  }

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_catalog'), t('nav_products')]}
        title={t('nav_products')}
        sub={`${(meta?.total ?? 0).toLocaleString()} products${
          cat !== 'all' || ff !== 'all' || status !== 'all' || debouncedSearch ? ' (filtered)' : ''
        }`}
      >
        <button className="abtn" onClick={handleExport} disabled={exporting}>
          <Icon name="download" size={15} /> {exporting ? '…' : t('export')}
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
          <select className="select" value={cat} onChange={(e) => onFilter(setCat)(e.target.value)}>
            <option value="all">All categories</option>
            {cats.map((c) => (
              <option key={c.value} value={c.value}>
                {categoryLabel(c.value)}
              </option>
            ))}
          </select>
          <select
            className="select"
            value={status}
            onChange={(e) => onFilter(setStatus)(e.target.value as StatusFilter)}
          >
            <option value="all">{t('all')} {t('status').toLowerCase()}</option>
            <option value="active">{t('active')}</option>
            <option value="draft">{t('draft')}</option>
            <option value="out">{t('out_of_stock')}</option>
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
                onClick={() => onFilter(setFf)(k as 'all' | FfKey)}
                dotColor={k !== 'all' ? FF_DOT[k as FfKey] : undefined}
              >
                {l}
              </Chip>
            ))}
          </div>
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
                        <ProductThumb p={p} />
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
                    <td className="num muted">{sold[p.id] ? sold[p.id].toLocaleString() : '—'}</td>
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
        <Pagination
          page={meta?.page ?? 1}
          pages={meta?.pages ?? 1}
          total={meta?.total ?? rows.length}
          shown={rows.length}
          limit={meta?.limit}
          label={t('pg_products')}
          onPage={setPage}
        />
      </div>
    </div>
  )
}
