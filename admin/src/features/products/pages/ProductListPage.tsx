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
  Modal,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
  artForCategory,
  ffKey,
  type FfKey,
  type IconName,
} from '@/components'
import { downloadCsv } from '@/lib/utils'
import { useBulk } from '@/hooks/useBulk'
import { useCan, useIsSuperAdmin } from '@/stores/auth'
import { toast } from '@/stores/toast'
import { useCategoryTree } from '@/features/categories/hooks/useCategories'
import { orderedTree, pathLabel, type AdminCategory } from '@/features/categories/api/categories'
import { useProductCategories } from '../hooks/useCategories'
import { categoryLabel } from '../api/categories'
import {
  useProducts,
  useDeleteProduct,
  useRestoreProduct,
  usePurgeProduct,
  useBulkProductAction,
} from '../hooks/useProducts'
import { useSoldByProduct } from '../hooks/useSoldByProduct'
import { listProducts, type BulkAction, type ProductListParams } from '../api/products'
import { productStatus, priceRange } from '../lib/view'
import { bulkActionsFor, bulkConfirmText, bulkDoneText, canSelectAllMatching } from '../lib/trash'
import { amountIgnoredByPrice } from '../lib/unitPricing'
import type { FulfillmentType, Product } from '@/types'

const FF_DOT: Record<FfKey, string> = {
  code: 'var(--ff-code)',
  credit: 'var(--ff-credit)',
  transfer: 'var(--ff-transfer)',
}

// 'deleted' is the trash view: only soft-deleted products (every other filter
// value excludes them).
type StatusFilter = 'all' | 'active' | 'draft' | 'out' | 'deleted'

// Bulk-bar button per action (assign-category opens the category modal instead).
const BULK_BUTTONS: Record<Exclude<BulkAction, 'assign-category'>, { label: string; icon: IconName; cls: string }> = {
  activate: { label: 'Activate', icon: 'check', cls: 'abtn xs ok' },
  deactivate: { label: 'Deactivate', icon: 'eyeoff', cls: 'abtn xs' },
  delete: { label: 'Move to trash', icon: 'trash', cls: 'abtn xs danger' },
  restore: { label: 'Restore', icon: 'refresh', cls: 'abtn xs ok' },
  purge: { label: 'Delete permanently', icon: 'trash', cls: 'abtn xs danger' },
}

// Prefers the compressed thumbnail, falling back to the display image, then to
// the gradient placeholder — both up front (no thumbnail/image) and on a
// broken URL (onError). images is optional-chained: legacy-import products can
// come back with images: null.
export function ProductThumb({ p }: { p: Product }) {
  const [failed, setFailed] = useState(false)
  const src = p.thumbnail || p.images?.[0]
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
  const can = useCan()
  const canManage = can('products.manage')
  // The Sold column reads the orders domain — hide it (and skip its fetch + CSV
  // column) when the admin's role lacks orders.view.
  const canViewSold = can('orders.view')
  const [ff, setFf] = useState<'all' | FfKey>('all')
  const [cat, setCat] = useState('all')
  const [status, setStatus] = useState<StatusFilter>('all')
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [exporting, setExporting] = useState(false)
  const [assignOpen, setAssignOpen] = useState(false)

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
  const restore = useRestoreProduct()
  const purge = usePurgeProduct()
  const bulkAction = useBulkProductAction()
  const isSuperAdmin = useIsSuperAdmin()
  const trashView = status === 'deleted'
  // "Select all N matching": the bulk action then targets every product matching
  // the filters (sent as a filter, not ids). Only meaningful while the whole page
  // is selected; any filter/page change drops it.
  const [matchAllRaw, setMatchAll] = useState(false)
  useEffect(() => setMatchAll(false), [cat, ff, status, debouncedSearch, page])
  const { data: catsRes } = useProductCategories()
  const { data: soldRes } = useSoldByProduct({ enabled: canViewSold })
  const cats = catsRes?.data ?? []
  const sold = soldRes?.data ?? {}

  const rows: Product[] = data?.data ?? []
  const meta = data?.meta
  const ids = rows.map((r) => r.id)
  const bulk = useBulk(ids)
  const matchAll = matchAllRaw && bulk.all
  const total = meta?.total ?? rows.length
  const selCount = matchAll ? total : bulk.sel.length
  const clearSelection = () => {
    bulk.clear()
    setMatchAll(false)
  }

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

  // Runs a bulk action on the selected rows — or, with "all matching", on every
  // product matching the current filters (purge is never offered for that).
  const runBulk = (action: BulkAction, categoryId?: string, onDone?: () => void) => {
    const confirmText = bulkConfirmText(action, selCount, matchAll)
    if (confirmText && !window.confirm(confirmText)) return
    bulkAction.mutate(
      { ids: matchAll ? [] : bulk.sel, action, categoryId, filter: matchAll ? filters : undefined },
      {
        onSuccess: (res) => {
          const n = res.data?.modified ?? 0
          toast.success(
            action === 'assign-category'
              ? categoryId
                ? `Moved ${n} product(s).`
                : `Unassigned ${n} product(s).`
              : bulkDoneText(action, n),
          )
          clearSelection()
          onDone?.()
        },
        onError: () => toast.error('Could not update the products.'),
      },
    )
  }

  const trashOne = (p: Product) => {
    if (!window.confirm(`Move "${p.title.en}" to the trash? You can restore it from the Deleted view.`)) return
    del.mutate(p.id, {
      onSuccess: () => toast.success('Moved to the trash.'),
      onError: () => toast.error('Could not move the product to the trash.'),
    })
  }
  const restoreOne = (p: Product) =>
    restore.mutate(p.id, {
      onSuccess: () => toast.success(`Restored "${p.title.en}".`),
      onError: () => toast.error('Could not restore the product.'),
    })
  const purgeOne = (p: Product) => {
    if (!window.confirm(`Permanently delete "${p.title.en}"? This cannot be undone.`)) return
    purge.mutate(p.id, {
      onSuccess: () => toast.success('Permanently deleted.'),
      onError: () => toast.error('Could not delete the product permanently.'),
    })
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

      // Omit the Sold column entirely without orders.view, rather than exporting a
      // misleading 0 for every product (the sold data would have 403'd).
      const header = ['Name', 'ID', 'Category', 'Type', 'Variants', 'Stock', 'Price min', 'Price max',
        ...(canViewSold ? ['Sold'] : []), 'Status']
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
          ...(canViewSold ? [sold[pr.id] ?? 0] : []),
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
        back="/"
        crumbs={[t('grp_catalog'), t('nav_products')]}
        title={t('nav_products')}
        sub={`${(meta?.total ?? 0).toLocaleString()} ${trashView ? 'products in the trash' : 'products'}${
          cat !== 'all' || ff !== 'all' || (status !== 'all' && !trashView) || debouncedSearch ? ' (filtered)' : ''
        }`}
      >
        <button className="abtn" onClick={handleExport} disabled={exporting}>
          <Icon name="download" size={15} /> {exporting ? '…' : t('export')}
        </button>
        {canManage && (
          <button className="abtn primary" onClick={() => navigate('/products/new')}>
            <Icon name="plus" size={15} /> {t('new_product')}
          </button>
        )}
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
            onChange={(e) => {
              // Switching into/out of the trash must never carry a selection
              // across — the bulk actions differ between the two views.
              clearSelection()
              onFilter(setStatus)(e.target.value as StatusFilter)
            }}
          >
            <option value="all">{t('all')} {t('status').toLowerCase()}</option>
            <option value="active">{t('active')}</option>
            <option value="draft">{t('draft')}</option>
            <option value="out">{t('out_of_stock')}</option>
            <option value="deleted">Deleted (trash)</option>
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

        {canManage && bulk.some && (
          <div className="bulkbar">
            <Checkbox on onClick={clearSelection} />
            <span>
              {matchAll ? `All ${total.toLocaleString()} matching` : bulk.sel.length} {t('selected')}
            </span>
            {/* "Select all matching" reaches past this page — the owner's
                one-click way to trash (or restore / move) the whole filtered catalog. */}
            {canSelectAllMatching(bulk.all, rows.length, total) &&
              (matchAll ? (
                <button className="abtn xs ghost" onClick={() => setMatchAll(false)}>
                  Only this page
                </button>
              ) : (
                <button className="abtn xs ghost" onClick={() => setMatchAll(true)}>
                  Select all {total.toLocaleString()} matching
                </button>
              ))}
            <div className="ba-act">
              {bulkActionsFor(trashView, isSuperAdmin, matchAll).map((action) =>
                action === 'assign-category' ? (
                  <button key={action} className="abtn xs" onClick={() => setAssignOpen(true)}>
                    <Icon name="layers" size={13} /> Move to category
                  </button>
                ) : (
                  <button
                    key={action}
                    className={BULK_BUTTONS[action].cls}
                    disabled={bulkAction.isPending}
                    onClick={() => runBulk(action)}
                  >
                    <Icon name={BULK_BUTTONS[action].icon} size={13} /> {BULK_BUTTONS[action].label}
                  </button>
                ),
              )}
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
                  {canViewSold && <th>Sold</th>}
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
                          {amountIgnoredByPrice(p).length > 0 && (
                            <span
                              className="st st-warn"
                              style={{ fontSize: 11, marginTop: 2, width: 'fit-content' }}
                              title={`Customers type "${amountIgnoredByPrice(p).join('", "')}" but pay the fixed package price — open the product and switch to Per unit, or change the field type.`}
                            >
                              <i className="d" />
                              Price ignores amount
                            </span>
                          )}
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
                    {canViewSold && (
                      <td className="num muted">{sold[p.id] ? sold[p.id].toLocaleString() : '—'}</td>
                    )}
                    <td>
                      <StatusBadge s={productStatus(p)} />
                    </td>
                    <td onClick={(e) => e.stopPropagation()}>
                      <div className="row-actions">
                        <span className="iact" onClick={() => navigate(`/products/${p.id}/edit`)}>
                          <Icon name="edit" size={15} />
                        </span>
                        {canManage && !p.deletedAt && (
                          <span className="iact danger" title="Move to trash" onClick={() => trashOne(p)}>
                            <Icon name="trash" size={15} />
                          </span>
                        )}
                        {canManage && p.deletedAt && (
                          <span className="iact" title="Restore" onClick={() => restoreOne(p)}>
                            <Icon name="refresh" size={15} />
                          </span>
                        )}
                        {isSuperAdmin && p.deletedAt && (
                          <span className="iact danger" title="Delete permanently" onClick={() => purgeOne(p)}>
                            <Icon name="xc" size={15} />
                          </span>
                        )}
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

      {assignOpen && (
        <AssignCategoryModal
          count={selCount}
          pending={bulkAction.isPending}
          onClose={() => setAssignOpen(false)}
          onConfirm={(categoryId) => runBulk('assign-category', categoryId, () => setAssignOpen(false))}
        />
      )}
    </div>
  )
}

/** Bulk "Move to category": pick a taxonomy node (or None to unassign) for the
 *  selected products. Options are the full tree, indented + path-labeled. */
function AssignCategoryModal({
  count,
  pending,
  onClose,
  onConfirm,
}: {
  count: number
  pending: boolean
  onClose: () => void
  onConfirm: (categoryId: string) => void
}) {
  const { t } = useTranslation()
  const { data } = useCategoryTree()
  const nodes: AdminCategory[] = data?.data ?? []
  const ordered = orderedTree(nodes)
  const byId = new Map(nodes.map((n) => [n.id, n]))
  const [categoryId, setCategoryId] = useState('')

  const target = categoryId ? pathLabel(byId.get(categoryId)!, byId) : 'no category (unassign)'

  return (
    <Modal onClose={onClose} maxWidth={480}>
      <div className="pad">
        <h3 style={{ marginTop: 0 }}>Move to category</h3>
        <p className="muted" style={{ fontSize: 13, lineHeight: 1.6 }}>
          Reassign {count} selected product{count === 1 ? '' : 's'} to{' '}
          <b>{target}</b>.
        </p>
        <label className="alabel">Category</label>
        <select
          className="select"
          style={{ width: '100%', marginBottom: 14 }}
          value={categoryId}
          onChange={(e) => setCategoryId(e.target.value)}
        >
          <option value="">— None (unassign) —</option>
          {ordered.map((n) => (
            <option key={n.id} value={n.id}>
              {pathLabel(n, byId)}
            </option>
          ))}
        </select>
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose}>
            {t('cancel')}
          </button>
          <button className="abtn primary" disabled={pending} onClick={() => onConfirm(categoryId)}>
            <Icon name="layers" size={14} /> {pending ? '…' : 'Move'}
          </button>
        </div>
      </div>
    </Modal>
  )
}
