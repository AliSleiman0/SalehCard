import { Fragment, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  Icon,
  PageHead,
  Modal,
  Toggle,
  StatusBadge,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { ApiError } from '@/lib/api-client'
import {
  useProducts,
  useUploadProductImage,
  useCategoryProductOrder,
  useSaveCategoryProductOrder,
} from '@/features/products/hooks/useProducts'
import { useCan } from '@/stores/auth'
import type { Product } from '@/types'
import { ProductThumb } from '@/features/products/pages/ProductListPage'
import { priceRange, productStatus } from '@/features/products/lib/view'
import { useCategoryTree, useCreateCategory, useUpdateCategory, useDeleteCategory } from '../hooks/useCategories'
import { orderedTree, pathLabel, subtreeIds, type AdminCategory } from '../api/categories'
import { moveItem } from '../lib/moveItem'

const MAX_DEPTH = 2 // Collection(0) → Category(1) → Subcategory(2)

// Category tree manager: the catalog taxonomy the app browses (Collections →
// Categories → Subcategories) and products are assigned to. A super admin or a
// "categories.manage" role can add/edit/reorder/hide nodes; deleting a node that
// still has children or assigned products is blocked server-side.
export default function CategoriesPage() {
  const { t } = useTranslation()
  const { data, isLoading, isError, refetch } = useCategoryTree()
  const del = useDeleteCategory()
  const update = useUpdateCategory()

  const nodes = useMemo(() => data?.data ?? [], [data])
  const ordered = useMemo(() => orderedTree(nodes), [nodes])
  const byId = useMemo(() => new Map(nodes.map((n) => [n.id, n])), [nodes])

  // { mode:'new', parentId } to create, or { mode:'edit', node } to edit.
  const [editing, setEditing] = useState<{ mode: 'new'; parentId?: string } | { mode: 'edit'; node: AdminCategory } | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<AdminCategory | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  // collapsed = nodes whose children are hidden; openProducts = nodes showing
  // their inline product panel; search filters the tree.
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set())
  const [openProducts, setOpenProducts] = useState<Set<string>>(new Set())
  const [collapseInited, setCollapseInited] = useState(false)
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')

  // Open collapsed-to-collections once the tree loads — a fully-expanded 78-node
  // tree is a wall of rows. Only nodes that have children are collapsible.
  useEffect(() => {
    if (!collapseInited && nodes.length > 0) {
      setCollapsed(new Set(nodes.filter((n) => n.hasChildren).map((n) => n.id)))
      setCollapseInited(true)
    }
  }, [nodes, collapseInited])

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 300)
    return () => clearTimeout(timer)
  }, [search])

  const q = debouncedSearch.trim().toLowerCase()

  const ancestorsOf = useCallback(
    (n: AdminCategory): string[] => {
      const out: string[] = []
      const seen = new Set<string>()
      let cur = n.parentId ? byId.get(n.parentId) : undefined
      while (cur && !seen.has(cur.id)) {
        seen.add(cur.id)
        out.push(cur.id)
        cur = cur.parentId ? byId.get(cur.parentId) : undefined
      }
      return out
    },
    [byId],
  )

  // While searching, show every match plus its ancestors (so hits keep context);
  // null = not searching.
  const matchIds = useMemo(() => {
    if (!q) return null
    const keep = new Set<string>()
    for (const n of nodes) {
      const hay = `${n.name.en} ${n.name.ar} ${n.name.tr} ${n.slug}`.toLowerCase()
      if (hay.includes(q)) {
        keep.add(n.id)
        for (const a of ancestorsOf(n)) keep.add(a)
      }
    }
    return keep
  }, [q, nodes, ancestorsOf])

  // A node is visible unless an ancestor is collapsed; search overrides collapse.
  const visible = useMemo(
    () =>
      ordered.filter((n) => {
        if (matchIds) return matchIds.has(n.id)
        return !ancestorsOf(n).some((a) => collapsed.has(a))
      }),
    [ordered, matchIds, collapsed, ancestorsOf],
  )

  const toggleSet = (setter: React.Dispatch<React.SetStateAction<Set<string>>>, id: string) =>
    setter((s) => {
      const next = new Set(s)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  const collapseAll = () => setCollapsed(new Set(nodes.filter((n) => n.hasChildren).map((n) => n.id)))
  const expandAll = () => setCollapsed(new Set())

  function runDelete(node: AdminCategory) {
    setDeleteError(null)
    del.mutate(node.id, {
      onSuccess: () => setConfirmDelete(null),
      onError: (e) => setDeleteError(e instanceof ApiError ? e.message : 'Could not delete the category.'),
    })
  }

  return (
    <div className="page">
      <PageHead
        back="/"
        crumbs={[t('grp_catalog'), t('nav_categories')]}
        title={t('nav_categories')}
        sub="Collections, categories and subcategories the app browses"
      >
        <button className="abtn primary" onClick={() => setEditing({ mode: 'new' })}>
          <Icon name="plus" size={15} /> New collection
        </button>
      </PageHead>

      <div className="acard">
        <div className="panelhead">
          <Icon name="layers" size={17} />
          <h3>Catalog taxonomy</h3>
        </div>
        {!isLoading && !isError && (
          <div className="toolbar">
            <div className="fsearch">
              <Icon name="search" size={15} />
              <input
                placeholder="Search collections & categories…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            <div style={{ marginLeft: 'auto', display: 'flex', gap: 8 }}>
              <button className="abtn sm" onClick={expandAll}>
                <Icon name="chevdown" size={13} /> Expand all
              </button>
              <button className="abtn sm" onClick={collapseAll}>
                <Icon name="chevright" size={13} /> Collapse all
              </button>
            </div>
          </div>
        )}
        {isLoading && <LoadingSpinner />}
        {isError && <ErrorState message="Couldn't load categories." onRetry={() => void refetch()} />}
        {!isLoading && !isError && (
          <div className="tablewrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Name</th>
                  <th style={{ width: 120 }}>Products</th>
                  <th style={{ width: 90, textAlign: 'center' }}>Visible</th>
                  <th style={{ width: 200 }}></th>
                </tr>
              </thead>
              <tbody>
                {visible.map((n) => {
                  const isCollapsed = !matchIds && collapsed.has(n.id)
                  const count = n.productCount ?? 0
                  const productsOpen = openProducts.has(n.id)
                  return (
                    <Fragment key={n.id}>
                      <tr>
                        <td>
                          <div style={{ display: 'flex', alignItems: 'center', paddingLeft: n.depth * 22 }}>
                            {n.hasChildren ? (
                              <button
                                type="button"
                                className="treecaret"
                                aria-label={isCollapsed ? 'Expand' : 'Collapse'}
                                onClick={() => toggleSet(setCollapsed, n.id)}
                                style={{ background: 'none', border: 0, padding: 0, cursor: 'pointer', color: 'inherit', display: 'inline-flex' }}
                              >
                                <Icon name={isCollapsed ? 'chevright' : 'chevdown'} size={14} />
                              </button>
                            ) : (
                              <span style={{ display: 'inline-block', width: 14 }} />
                            )}
                            <div style={{ marginLeft: 6 }}>
                              <b>{n.name.en || n.slug}</b>
                              <div className="muted" style={{ fontSize: 12 }}>
                                {n.slug}
                                {n.depth === 0 && ' · Collection'}
                                {n.depth === 1 && ' · Category'}
                                {n.depth === 2 && ' · Subcategory'}
                              </div>
                            </div>
                          </div>
                        </td>
                        <td style={{ fontSize: 13 }}>
                          {count > 0 ? (
                            <button
                              type="button"
                              onClick={() => toggleSet(setOpenProducts, n.id)}
                              style={{ background: 'none', border: 0, padding: 0, cursor: 'pointer', color: 'var(--brand)', fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: 4 }}
                            >
                              <Icon name={productsOpen ? 'chevdown' : 'chevright'} size={12} />
                              {count}
                            </button>
                          ) : (
                            <span className="muted">0</span>
                          )}
                        </td>
                        <td style={{ textAlign: 'center' }}>
                          <div style={{ display: 'inline-flex' }}>
                            <Toggle
                              on={n.visible}
                              onClick={() => update.mutate({ id: n.id, input: { visible: !n.visible } })}
                            />
                          </div>
                        </td>
                        <td>
                          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
                            {n.depth < MAX_DEPTH && (
                              <button className="abtn sm" onClick={() => setEditing({ mode: 'new', parentId: n.id })}>
                                <Icon name="plus" size={13} /> Child
                              </button>
                            )}
                            <button className="abtn sm" onClick={() => setEditing({ mode: 'edit', node: n })}>
                              <Icon name="edit" size={14} /> {t('edit')}
                            </button>
                            <button
                              className="abtn sm danger"
                              onClick={() => {
                                setDeleteError(null)
                                setConfirmDelete(n)
                              }}
                            >
                              <Icon name="trash" size={14} />
                            </button>
                          </div>
                        </td>
                      </tr>
                      {productsOpen && (
                        <tr>
                          <td colSpan={4} style={{ background: 'var(--surface-2)', padding: 0 }}>
                            <CategoryProductsPanel node={n} />
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  )
                })}
                {visible.length === 0 && (
                  <tr>
                    <td colSpan={4} className="muted" style={{ textAlign: 'center', padding: 20 }}>
                      {q
                        ? 'No collections or categories match your search.'
                        : 'No categories yet — create a Collection to get started.'}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {editing && (
        <CategoryEditorModal
          key={editing.mode === 'edit' ? editing.node.id : 'new-' + (editing.parentId ?? 'root')}
          editing={editing}
          nodes={nodes}
          byId={byId}
          onClose={() => setEditing(null)}
        />
      )}

      {confirmDelete && (
        <Modal onClose={() => setConfirmDelete(null)}>
          <div className="pad">
            <h3 style={{ marginTop: 0 }}>Delete “{confirmDelete.name.en || confirmDelete.slug}”?</h3>
            <p className="muted" style={{ fontSize: 13, lineHeight: 1.6 }}>
              A category with subcategories or assigned products can't be deleted — move or reassign
              them first.
            </p>
            {deleteError && (
              <div style={{ color: 'var(--danger)', fontSize: 12.5, fontWeight: 600, marginBottom: 12 }}>
                {deleteError}
              </div>
            )}
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
              <button className="abtn" onClick={() => setConfirmDelete(null)}>
                {t('cancel')}
              </button>
              <button className="abtn danger" disabled={del.isPending} onClick={() => runDelete(confirmDelete)}>
                <Icon name="trash" size={14} /> {t('delete')}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}

/** Create / edit modal: name (en/ar/tr), parent, image, order, visibility. */
function CategoryEditorModal({
  editing,
  nodes,
  byId,
  onClose,
}: {
  editing: { mode: 'new'; parentId?: string } | { mode: 'edit'; node: AdminCategory }
  nodes: AdminCategory[]
  byId: Map<string, AdminCategory>
  onClose: () => void
}) {
  const { t } = useTranslation()
  const create = useCreateCategory()
  const update = useUpdateCategory()
  const uploadImage = useUploadProductImage()
  const pending = create.isPending || update.isPending || uploadImage.isPending

  const node = editing.mode === 'edit' ? editing.node : null
  const [name, setName] = useState({ en: node?.name.en ?? '', ar: node?.name.ar ?? '', tr: node?.name.tr ?? '' })
  const [parentId, setParentId] = useState(node?.parentId ?? (editing.mode === 'new' ? editing.parentId ?? '' : ''))
  const [image, setImage] = useState(node?.image ?? '')
  const [sortOrder, setSortOrder] = useState(String(node?.sortOrder ?? 0))
  const [visible, setVisible] = useState(node?.visible ?? true)
  const [error, setError] = useState<string | null>(null)

  const initialParent = node?.parentId ?? ''

  // Valid parents: nodes that can still take a child (depth < MAX) and — when
  // editing — are not the node itself or one of its descendants (no cycles).
  const excluded = node ? subtreeIds(node.id, nodes) : new Set<string>()
  const parentOptions = nodes.filter((n) => n.depth < MAX_DEPTH && !excluded.has(n.id))

  function save() {
    setError(null)
    if (!name.en.trim()) {
      setError('An English name is required.')
      return
    }
    const order = Number(sortOrder) || 0
    const opts = {
      onSuccess: onClose,
      onError: (e: Error) => setError(e instanceof ApiError ? e.message : 'Could not save the category.'),
    }
    if (node) {
      const input: Parameters<typeof update.mutate>[0]['input'] = {
        name: { en: name.en.trim(), ar: name.ar.trim(), tr: name.tr.trim() },
        image: image.trim(),
        sortOrder: order,
        visible,
      }
      // Move only when the parent actually changed to another node (the backend
      // does not support re-parenting to the top level).
      if (parentId && parentId !== initialParent) input.parentId = parentId
      update.mutate({ id: node.id, input }, opts)
    } else {
      create.mutate(
        {
          parentId: parentId || undefined,
          name: { en: name.en.trim(), ar: name.ar.trim(), tr: name.tr.trim() },
          image: image.trim(),
          sortOrder: order,
          visible,
        },
        opts,
      )
    }
  }

  function onFile(file: File) {
    setError(null)
    if (file.size > 10 * 1024 * 1024) {
      setError('Image must be 10 MB or smaller.')
      return
    }
    uploadImage.mutate(file, {
      onSuccess: (res) => res.data && setImage(res.data.imageUrl),
      onError: (e) => setError((e as Error).message),
    })
  }

  const title = node
    ? `Edit “${node.name.en || node.slug}”`
    : editing.mode === 'new' && editing.parentId
      ? 'New subcategory'
      : 'New collection'

  return (
    <Modal onClose={onClose} maxWidth={560}>
      <div className="pad">
        <h3 style={{ marginTop: 0 }}>{title}</h3>

        <label className="alabel">Parent</label>
        <select
          className="select"
          style={{ width: '100%', marginBottom: 14 }}
          value={parentId}
          onChange={(e) => setParentId(e.target.value)}
        >
          <option value="">— Top-level Collection —</option>
          {parentOptions.map((p) => (
            <option key={p.id} value={p.id}>
              {pathLabel(p, byId)}
            </option>
          ))}
        </select>

        <div className="g2" style={{ marginBottom: 14 }}>
          <div>
            <label className="alabel">Name (English)</label>
            <input className="afield" value={name.en} onChange={(e) => setName({ ...name, en: e.target.value })} />
          </div>
          <div>
            <label className="alabel">Name (العربية)</label>
            <input className="afield" value={name.ar} onChange={(e) => setName({ ...name, ar: e.target.value })} />
          </div>
          <div>
            <label className="alabel">Name (Türkçe)</label>
            <input className="afield" value={name.tr} onChange={(e) => setName({ ...name, tr: e.target.value })} />
          </div>
          <div>
            <label className="alabel">Sort order</label>
            <input
              className="afield"
              type="number"
              value={sortOrder}
              onChange={(e) => setSortOrder(e.target.value)}
            />
          </div>
        </div>

        <label className="alabel">Image</label>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 14 }}>
          {image ? (
            <img src={image} alt="" style={{ width: 48, height: 48, borderRadius: 8, objectFit: 'cover' }} />
          ) : (
            <div style={{ width: 48, height: 48, borderRadius: 8, background: 'var(--surface-2)', display: 'grid', placeItems: 'center' }}>
              <Icon name="box" size={18} />
            </div>
          )}
          <label className="abtn sm" style={{ cursor: 'pointer' }}>
            <Icon name="upload" size={13} /> {uploadImage.isPending ? 'Uploading…' : 'Choose image'}
            <input
              type="file"
              accept="image/*"
              style={{ display: 'none' }}
              onChange={(e) => {
                const f = e.target.files?.[0]
                if (f) onFile(f)
              }}
            />
          </label>
          {image && (
            <button className="abtn sm danger" onClick={() => setImage('')}>
              <Icon name="x" size={13} /> Remove
            </button>
          )}
        </div>

        <label className="alabel" style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <Toggle on={visible} onClick={() => setVisible(!visible)} /> Visible to customers
        </label>

        {error && (
          <div style={{ color: 'var(--danger)', fontSize: 12.5, fontWeight: 600, marginTop: 12 }}>{error}</div>
        )}
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 16 }}>
          <button className="abtn" onClick={onClose}>
            {t('cancel')}
          </button>
          <button className="abtn primary" disabled={pending} onClick={save}>
            <Icon name="check" size={15} /> {t('save')}
          </button>
        </div>
      </div>
    </Modal>
  )
}

/** Inline product list under an expanded category row. Tree-aware — lists the
 *  node's own + descendants' products (matching the roll-up count) in the order
 *  customers see them, with its own search + pagination. Rows link to the
 *  product editor. "Arrange order" switches to the reorder list. */
function CategoryProductsPanel({ node }: { node: AdminCategory }) {
  const navigate = useNavigate()
  const can = useCan()
  const canArrange = can('products.manage')
  const [arranging, setArranging] = useState(false)
  const [search, setSearch] = useState('')
  const [debounced, setDebounced] = useState('')
  const [page, setPage] = useState(1)

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebounced(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  const term = debounced.trim()
  // Browsing uses the customer catalog order. A search drops it (newest
  // first): every distinct term would otherwise load + cache a whole sorted
  // set on the API, crowding out the cached customer listings.
  const { data, isLoading, isError, refetch } = useProducts(
    {
      categoryId: node.id,
      search: term || undefined,
      sort: term ? undefined : 'name',
      page,
    },
    { enabled: !arranging },
  )
  const rows = data?.data ?? []
  const meta = data?.meta

  if (arranging) {
    return <ArrangeProductsPanel node={node} onDone={() => setArranging(false)} />
  }

  return (
    <div style={{ padding: '12px 16px' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8, flexWrap: 'wrap' }}>
        <div className="fsearch" style={{ maxWidth: 340, flex: '1 1 220px' }}>
          <Icon name="search" size={14} />
          <input
            placeholder="Search products in this category…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        {canArrange && (
          <button className="abtn sm" style={{ marginLeft: 'auto' }} onClick={() => setArranging(true)}>
            <Icon name="sort" size={13} /> Arrange order
          </button>
        )}
      </div>
      <div className="ahint" style={{ marginTop: 0, marginBottom: 10 }}>
        {node.hasChildren
          ? 'Includes subcategories, listed A to Z. “Arrange order” sets the order of the products placed directly in this category; arrange each subcategory on its own row.'
          : 'Listed in the order customers see.'}{' '}
        The count reflects available products; drafts are listed here too. Assign products from the
        Products page (select rows → Move to category).
      </div>
      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message="Could not load products." onRetry={() => void refetch()} />
      ) : rows.length === 0 ? (
        <EmptyState
          title="No products here yet."
          sub={debounced ? 'Try a different search.' : 'Assign products from the Products page.'}
        />
      ) : (
        <>
          <div className="tablewrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Product</th>
                  <th>Category</th>
                  <th>Price (USD)</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((p) => (
                  <tr key={p.id} className="clickable" onClick={() => navigate(`/products/${p.id}/edit`)}>
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
                    <td className="num strong">
                      {priceRange(p) === '—' ? <span className="faint">—</span> : '$' + priceRange(p)}
                    </td>
                    <td>
                      <StatusBadge s={productStatus(p)} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Pagination
            page={meta?.page ?? 1}
            pages={meta?.pages ?? 1}
            total={meta?.total ?? rows.length}
            shown={rows.length}
            limit={meta?.limit}
            onPage={setPage}
          />
        </>
      )}
    </div>
  )
}

/** Reorder list for the products assigned DIRECTLY to one node: drag a row by
 *  its handle, or use the arrow buttons (keyboard-friendly), then Save. The
 *  order is per node — products in subcategories are arranged on their own
 *  subcategory. Products never arranged sit after the arranged ones, A to Z. */
function ArrangeProductsPanel({ node, onDone }: { node: AdminCategory; onDone: () => void }) {
  const { data, isLoading, isError, refetch, isFetching } = useCategoryProductOrder(node.id)
  const save = useSaveCategoryProductOrder()
  // null = untouched: follow the server order (also after a reload).
  const [order, setOrder] = useState<Product[] | null>(null)
  // The drag source lives in a ref (read synchronously by dragover/drop, which
  // can fire before React re-renders after dragstart); the state mirrors it
  // only for styling.
  const dragRef = useRef<number | null>(null)
  const [dragFrom, setDragFrom] = useState<number | null>(null)
  const [dragOver, setDragOver] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [stale, setStale] = useState(false)

  const loaded = useMemo(() => data?.data ?? [], [data])
  const list = order ?? loaded
  const dirty = list.some((p, i) => p.id !== loaded[i]?.id)
  const name = node.name.en || node.slug

  function move(from: number, to: number) {
    setError(null)
    setOrder(moveItem(list, from, to))
  }

  function endDrag() {
    dragRef.current = null
    setDragFrom(null)
    setDragOver(null)
  }

  function runSave() {
    setError(null)
    setStale(false)
    save.mutate(
      { categoryId: node.id, productIds: list.map((p) => p.id) },
      {
        onSuccess: onDone,
        onError: (e) => {
          if (e instanceof ApiError && e.code === 'STALE_ORDER') setStale(true)
          setError(e instanceof ApiError ? e.message : 'Could not save the order.')
        },
      },
    )
  }

  function reload() {
    setError(null)
    setStale(false)
    setOrder(null)
    void refetch()
  }

  return (
    <div style={{ padding: '12px 16px' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8, flexWrap: 'wrap' }}>
        <b style={{ fontSize: 13.5 }}>Arrange products in “{name}”</b>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: 8 }}>
          <button className="abtn sm" onClick={onDone} disabled={save.isPending}>
            {dirty ? 'Cancel' : 'Done'}
          </button>
          <button
            className="abtn sm primary"
            onClick={runSave}
            disabled={!dirty || save.isPending || isFetching || list.length === 0}
          >
            <Icon name="check" size={13} /> {save.isPending ? 'Saving…' : 'Save order'}
          </button>
        </div>
      </div>
      <div className="ahint" style={{ marginTop: 0, marginBottom: 10 }}>
        Customers see these products in this order, in the app and on the web. Drag a row by its
        handle or use the arrows. This arranges only the products placed directly in “{name}” —
        products in its subcategories are arranged on each subcategory. Products added later appear
        after the arranged ones, A to Z.
      </div>
      {error && (
        <div
          role="alert"
          style={{ color: 'var(--danger)', fontSize: 12.5, fontWeight: 600, marginBottom: 10, display: 'flex', gap: 10, alignItems: 'center' }}
        >
          {error}
          {stale && (
            <button className="abtn sm" onClick={reload}>
              <Icon name="refresh" size={13} /> Reload
            </button>
          )}
        </div>
      )}
      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message="Could not load products." onRetry={() => void refetch()} />
      ) : list.length === 0 ? (
        <EmptyState
          title="No products placed directly in this category."
          sub={
            node.hasChildren
              ? 'Open a subcategory and arrange its products there.'
              : 'Assign products from the Products page.'
          }
        />
      ) : (
        <div className="tablewrap">
          <table className="tbl">
            <thead>
              <tr>
                <th style={{ width: 44 }}>#</th>
                <th>Product</th>
                <th>Status</th>
                <th style={{ width: 170, textAlign: 'right' }}>Move</th>
              </tr>
            </thead>
            <tbody>
              {list.map((p, i) => {
                const label = p.title.en || p.id.slice(-8)
                const showDrop = dragOver === i && dragFrom !== null && dragFrom !== i
                return (
                  <tr
                    key={p.id}
                    onDragOver={(e) => {
                      if (dragRef.current === null) return
                      e.preventDefault()
                      e.dataTransfer.dropEffect = 'move'
                      if (dragOver !== i) setDragOver(i)
                    }}
                    onDrop={(e) => {
                      e.preventDefault()
                      const from = dragRef.current
                      if (from !== null) move(from, i)
                      endDrag()
                    }}
                    style={{
                      opacity: dragFrom === i ? 0.45 : 1,
                      boxShadow: showDrop ? `inset 0 ${dragFrom! < i ? -2 : 2}px 0 var(--brand)` : undefined,
                    }}
                  >
                    <td className="num muted">{i + 1}</td>
                    <td>
                      <div className="cellprod">
                        <span
                          draggable
                          title="Drag to reorder"
                          aria-hidden="true"
                          onDragStart={(e) => {
                            dragRef.current = i
                            setDragFrom(i)
                            e.dataTransfer.effectAllowed = 'move'
                            e.dataTransfer.setData('text/plain', p.id)
                          }}
                          onDragEnd={endDrag}
                          style={{ cursor: 'grab', display: 'inline-flex', padding: '0 2px' }}
                          className="faint"
                        >
                          <Icon name="dots" size={16} />
                        </span>
                        <ProductThumb p={p} />
                        <div className="pn">
                          <b>{p.title.en}</b>
                          <span className="mono">{p.id.slice(-8)}</span>
                        </div>
                      </div>
                    </td>
                    <td>
                      <StatusBadge s={productStatus(p)} />
                    </td>
                    <td>
                      <div style={{ display: 'flex', gap: 4, justifyContent: 'flex-end' }}>
                        <button className="abtn sm" aria-label={`Move ${label} to top`} title="Move to top" disabled={i === 0} onClick={() => move(i, 0)}>
                          <Icon name="arrowup" size={13} />
                        </button>
                        <button className="abtn sm" aria-label={`Move ${label} up`} title="Move up" disabled={i === 0} onClick={() => move(i, i - 1)}>
                          <Icon name="chevup" size={13} />
                        </button>
                        <button className="abtn sm" aria-label={`Move ${label} down`} title="Move down" disabled={i === list.length - 1} onClick={() => move(i, i + 1)}>
                          <Icon name="chevdown" size={13} />
                        </button>
                        <button className="abtn sm" aria-label={`Move ${label} to bottom`} title="Move to bottom" disabled={i === list.length - 1} onClick={() => move(i, list.length - 1)}>
                          <Icon name="arrowdown" size={13} />
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
