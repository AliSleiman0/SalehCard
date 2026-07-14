import { useState } from 'react'
import { Modal, Icon, LoadingSpinner, EmptyState, Pagination } from '@/components'
import { useCodes, useAddCode, useEditCode, useDeleteCode, useExpireCode } from '../hooks/useInventory'
import type { Code, CodeStatus, InventoryStats } from '@/types'

// Keep a page short enough that the whole modal — add form, list, pagination and
// Close — fits on screen without the footer falling below the fold (the modal has
// no viewport-height cap). The list area also scrolls as a safety net on short screens.
const PAGE_SIZE = 10

// Status → badge tone (`.st st-*`). Available is a healthy green, delivered is
// neutral (it's spent, tied to an order), expired is a retired/danger tone.
const STATUS_BADGE: Record<CodeStatus, string> = {
  available: 'st-ok',
  delivered: 'st-mute',
  expired: 'st-danger',
}

/**
 * Per-product codes management popup. Lists a product's codes (paginated),
 * and — for `inventory.manage` admins — lets them add a single code, edit or
 * expire an available one, and delete a non-delivered one. Delivered codes are
 * tied to an order's fulfillment, so they carry no destructive actions.
 */
export function CodesModal({ product, onClose }: { product: InventoryStats; onClose: () => void }) {
  const productId = product.productId
  const [page, setPage] = useState(1)
  const { data, isLoading } = useCodes(productId, { page, limit: PAGE_SIZE })
  const codes = data?.data ?? []
  const meta = data?.meta

  const add = useAddCode()
  const edit = useEditCode()
  const del = useDeleteCode()
  const expire = useExpireCode()

  // Add form.
  const [newCode, setNewCode] = useState('')
  const [newPin, setNewPin] = useState('')
  const submitAdd = () => {
    if (!newCode.trim()) return
    add.mutate(
      { productId, code: newCode.trim(), pin: newPin.trim() || undefined },
      { onSuccess: () => { setNewCode(''); setNewPin('') } }
    )
  }

  // Inline edit of an available code.
  const [editing, setEditing] = useState<Code | null>(null)
  const [editCodeVal, setEditCodeVal] = useState('')
  const [editPinVal, setEditPinVal] = useState('')
  const startEdit = (c: Code) => {
    setEditing(c)
    setEditCodeVal(c.code)
    setEditPinVal(c.pin ?? '')
  }
  const saveEdit = () => {
    if (!editing || !editCodeVal.trim()) return
    edit.mutate(
      { productId, codeId: editing.id, code: editCodeVal.trim(), pin: editPinVal.trim() || undefined },
      { onSuccess: () => setEditing(null) }
    )
  }

  // Delete confirmation (inline Modal, matching the CategoriesPage pattern).
  const [confirmDel, setConfirmDel] = useState<Code | null>(null)
  const runDelete = () => {
    if (!confirmDel) return
    del.mutate({ productId, codeId: confirmDel.id }, { onSuccess: () => setConfirmDel(null) })
  }

  return (
    <Modal onClose={onClose} maxWidth={760}>
      <div className="pad">
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 4 }}>
          <Icon name="layers" size={18} />
          <h3 style={{ margin: 0 }}>Codes — {product.title}</h3>
        </div>
        <div className="faint" style={{ fontSize: 12.5, marginBottom: 16 }}>
          {product.available} available · {product.uploaded.toLocaleString()} total
        </div>

        {/* Add a single code */}
        <div style={{ display: 'flex', gap: 8, marginBottom: 16, alignItems: 'flex-end', flexWrap: 'wrap' }}>
          <div style={{ flex: '1 1 220px' }}>
            <label className="alabel">New code</label>
            <input
              className="afield"
              style={{ width: '100%', fontFamily: 'ui-monospace' }}
              value={newCode}
              onChange={(e) => setNewCode(e.target.value)}
              placeholder="ABC-123-XYZ"
              onKeyDown={(e) => e.key === 'Enter' && submitAdd()}
            />
          </div>
          <div style={{ flex: '0 1 130px' }}>
            <label className="alabel">PIN (optional)</label>
            <input
              className="afield"
              style={{ width: '100%', fontFamily: 'ui-monospace' }}
              value={newPin}
              onChange={(e) => setNewPin(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && submitAdd()}
            />
          </div>
          <button className="abtn primary" disabled={!newCode.trim() || add.isPending} onClick={submitAdd}>
            <Icon name="plus" size={14} /> Add
          </button>
        </div>

        {/* Code list */}
        {isLoading ? (
          <LoadingSpinner />
        ) : codes.length === 0 ? (
          <EmptyState icon="layers" title="No codes yet" sub="Add a code above or bulk-upload a batch." />
        ) : (
          <div className="tablewrap" style={{ maxHeight: '46vh', overflowY: 'auto' }}>
            <table className="tbl">
              <thead>
                <tr>
                  <th>Code</th>
                  <th>PIN</th>
                  <th>Status</th>
                  <th>Order</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {codes.map((c) => {
                  const isEditing = editing?.id === c.id
                  return (
                    <tr key={c.id}>
                      <td className="mono">
                        {isEditing ? (
                          <input
                            className="afield"
                            style={{ padding: '4px 8px', fontFamily: 'ui-monospace' }}
                            value={editCodeVal}
                            onChange={(e) => setEditCodeVal(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && saveEdit()}
                          />
                        ) : (
                          c.code
                        )}
                      </td>
                      <td className="mono">
                        {isEditing ? (
                          <input
                            className="afield"
                            style={{ padding: '4px 8px', width: 80, fontFamily: 'ui-monospace' }}
                            value={editPinVal}
                            onChange={(e) => setEditPinVal(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && saveEdit()}
                          />
                        ) : (
                          c.pin || '—'
                        )}
                      </td>
                      <td>
                        <span className={'st ' + STATUS_BADGE[c.status]} style={{ textTransform: 'capitalize' }}>
                          <i className="d" />
                          {c.status}
                        </span>
                      </td>
                      <td className="mono faint">{c.orderId ? c.orderId.slice(-8) : '—'}</td>
                      <td>
                        <div className="row-actions">
                          {isEditing ? (
                            <>
                              <button
                                className="abtn xs primary"
                                disabled={!editCodeVal.trim() || edit.isPending}
                                onClick={saveEdit}
                              >
                                Save
                              </button>
                              <button className="abtn xs" onClick={() => setEditing(null)}>
                                Cancel
                              </button>
                            </>
                          ) : c.status === 'available' ? (
                            <>
                              <button className="abtn xs" onClick={() => startEdit(c)}>
                                <Icon name="edit" size={12} /> Edit
                              </button>
                              <button
                                className="abtn xs"
                                disabled={expire.isPending}
                                onClick={() => expire.mutate({ code: c.code, productId })}
                                title="Retire this code from the available pool"
                              >
                                Expire
                              </button>
                              <button className="abtn xs danger" onClick={() => setConfirmDel(c)} title="Delete this code">
                                <Icon name="trash" size={12} />
                              </button>
                            </>
                          ) : c.status === 'expired' ? (
                            <button className="abtn xs danger" onClick={() => setConfirmDel(c)}>
                              <Icon name="trash" size={12} /> Delete
                            </button>
                          ) : (
                            <span className="faint" style={{ fontSize: 11.5 }}>
                              Sold — no actions
                            </span>
                          )}
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}

        <Pagination
          page={meta?.page ?? 1}
          pages={meta?.pages ?? 1}
          total={meta?.total ?? codes.length}
          shown={codes.length}
          limit={meta?.limit}
          onPage={setPage}
        />

        <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 16 }}>
          <button className="abtn" onClick={onClose}>
            Close
          </button>
        </div>
      </div>

      {confirmDel && (
        <Modal onClose={() => setConfirmDel(null)}>
          <div className="pad">
            <h3 style={{ marginTop: 0 }}>Delete this code?</h3>
            <p className="faint" style={{ fontSize: 13 }}>
              <span className="mono">{confirmDel.code}</span> will be permanently removed from the pool. This can't be
              undone.
            </p>
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
              <button className="abtn" onClick={() => setConfirmDel(null)}>
                Cancel
              </button>
              <button className="abtn danger" disabled={del.isPending} onClick={runDelete}>
                <Icon name="trash" size={14} /> Delete
              </button>
            </div>
          </div>
        </Modal>
      )}
    </Modal>
  )
}
