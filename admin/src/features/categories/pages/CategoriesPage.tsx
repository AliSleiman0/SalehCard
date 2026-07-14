import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon, PageHead, Modal, Toggle, LoadingSpinner, ErrorState } from '@/components'
import { ApiError } from '@/lib/api-client'
import { useUploadProductImage } from '@/features/products/hooks/useProducts'
import { useCategoryTree, useCreateCategory, useUpdateCategory, useDeleteCategory } from '../hooks/useCategories'
import { orderedTree, pathLabel, subtreeIds, type AdminCategory } from '../api/categories'

const MAX_DEPTH = 2 // Collection(0) → Category(1) → Subcategory(2)

// Category tree manager: the catalog taxonomy the app browses (Collections →
// Categories → Subcategories) and products are assigned to. A super admin or a
// "categories.manage" role can add/edit/reorder/hide nodes; deleting a node that
// still has children or assigned products is blocked server-side.
export default function CategoriesPage() {
  const { t } = useTranslation()
  const { data, isLoading, isError, refetch } = useCategoryTree()
  const del = useDeleteCategory()

  const nodes = useMemo(() => data?.data ?? [], [data])
  const ordered = useMemo(() => orderedTree(nodes), [nodes])
  const byId = useMemo(() => new Map(nodes.map((n) => [n.id, n])), [nodes])

  // { mode:'new', parentId } to create, or { mode:'edit', node } to edit.
  const [editing, setEditing] = useState<{ mode: 'new'; parentId?: string } | { mode: 'edit'; node: AdminCategory } | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<AdminCategory | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const update = useUpdateCategory()

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
                {ordered.map((n) => (
                  <tr key={n.id}>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', paddingLeft: n.depth * 22 }}>
                        {n.depth > 0 && <Icon name="chevright" size={13} />}
                        <div style={{ marginLeft: n.depth > 0 ? 4 : 0 }}>
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
                    <td className="muted" style={{ fontSize: 13 }}>
                      {n.productCount ?? 0}
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
                ))}
                {ordered.length === 0 && (
                  <tr>
                    <td colSpan={4} className="muted" style={{ textAlign: 'center', padding: 20 }}>
                      No categories yet — create a Collection to get started.
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
