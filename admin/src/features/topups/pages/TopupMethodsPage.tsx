import { useState } from 'react'
import {
  Icon,
  PageHead,
  Modal,
  Toggle,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { ApiError } from '@/lib/api-client'
import { useCan } from '@/stores/auth'
import {
  useTopupMethods,
  useCreateMethod,
  useUpdateMethod,
  useDeleteMethod,
} from '../hooks/useTopupMethods'
import type { MethodField, MethodFieldType, MethodInput, TopUpMethod } from '../api/methods'

const FIELD_TYPES: [MethodFieldType, string][] = [
  ['text', 'Text'],
  ['number', 'Number'],
  ['select', 'Dropdown'],
  ['file', 'File / image upload'],
]

/** slugify turns a label into a stable field key (lowercase, alnum + underscore). */
function slugify(s: string): string {
  return s
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
}

export default function TopupMethodsPage() {
  const canManage = useCan()('topups.manage')
  const { data, isLoading, isError, refetch } = useTopupMethods()
  const methods = data?.data ?? []
  const [editing, setEditing] = useState<TopUpMethod | 'new' | null>(null)
  const delM = useDeleteMethod()

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={['Finance', 'Payment methods']}
        title="Manual payment methods"
        sub="Customer-facing wallet top-up options. Each is informational — a customer submits the inputs you define, then you approve the request in the Top-up queue to credit their wallet."
      >
        {canManage && (
          <button className="abtn primary" onClick={() => setEditing('new')}>
            <Icon name="plus" size={15} /> New method
          </button>
        )}
      </PageHead>

      <div className="acard">
        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message="Couldn't load payment methods." onRetry={() => refetch()} />
        ) : methods.length === 0 ? (
          <EmptyState
            title="No payment methods yet"
            sub="Add one (e.g. Bank Transfer) with instructions and the inputs customers should submit."
          />
        ) : (
          <div>
            {methods.map((m) => (
              <div
                key={m.id}
                style={{ borderBottom: '1px solid var(--border)', padding: '14px 18px' }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                  <b style={{ fontSize: 15 }}>{m.name}</b>
                  {m.enabled ? (
                    <span className="st st-ok">
                      <i className="d" />
                      Enabled
                    </span>
                  ) : (
                    <span className="bdg">Disabled</span>
                  )}
                  <span className="faint" style={{ fontSize: 12.5 }}>
                    {m.fields.length} input{m.fields.length === 1 ? '' : 's'}
                  </span>
                  {canManage && (
                    <div style={{ marginInlineStart: 'auto', display: 'flex', gap: 8 }}>
                      <button className="abtn xs" onClick={() => setEditing(m)}>
                        <Icon name="edit" size={13} /> Edit
                      </button>
                      <button
                        className="abtn xs danger"
                        disabled={delM.isPending}
                        onClick={() => {
                          if (confirm(`Delete "${m.name}"? Existing requests keep their record.`)) {
                            delM.mutate(m.id)
                          }
                        }}
                      >
                        <Icon name="trash" size={13} /> Delete
                      </button>
                    </div>
                  )}
                </div>
                {m.instructions && (
                  <div style={{ fontSize: 13, color: 'var(--text-dim)', marginTop: 6 }}>
                    {m.instructions}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {editing && (
        <MethodEditor
          method={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
    </div>
  )
}

/** blankField returns a new empty input row. */
function blankField(): MethodField {
  return { key: '', label: '', type: 'text', required: false, options: [] }
}

/** MethodEditor is the create/edit modal with a repeatable field builder. */
function MethodEditor({ method, onClose }: { method: TopUpMethod | null; onClose: () => void }) {
  const [name, setName] = useState(method?.name ?? '')
  const [instructions, setInstructions] = useState(method?.instructions ?? '')
  const [enabled, setEnabled] = useState(method?.enabled ?? true)
  const [sortOrder, setSortOrder] = useState(method?.sortOrder ?? 0)
  const [fields, setFields] = useState<MethodField[]>(method?.fields ?? [])
  const [error, setError] = useState('')

  const createM = useCreateMethod()
  const updateM = useUpdateMethod()
  const saving = createM.isPending || updateM.isPending

  const setField = (i: number, patch: Partial<MethodField>) =>
    setFields((fs) => fs.map((f, idx) => (idx === i ? { ...f, ...patch } : f)))

  const save = () => {
    if (!name.trim()) {
      setError('A method name is required.')
      return
    }
    // Derive missing keys from labels and enforce uniqueness client-side (the
    // server re-validates).
    const seen = new Set<string>()
    const cleaned: MethodField[] = []
    for (const f of fields) {
      const label = f.label.trim()
      if (!label) {
        setError('Every input needs a label.')
        return
      }
      const key = slugify(f.key || label)
      if (!key) {
        setError(`Couldn't derive a key for "${label}". Give it a manual key.`)
        return
      }
      if (seen.has(key)) {
        setError(`Duplicate input key "${key}". Keys must be unique.`)
        return
      }
      seen.add(key)
      const options =
        f.type === 'select' ? (f.options ?? []).map((o) => o.trim()).filter(Boolean) : undefined
      if (f.type === 'select' && (!options || options.length === 0)) {
        setError(`Dropdown "${label}" needs at least one option.`)
        return
      }
      cleaned.push({ key, label, type: f.type, required: f.required, options })
    }

    const input: MethodInput = {
      name: name.trim(),
      instructions: instructions.trim(),
      enabled,
      sortOrder: Number(sortOrder) || 0,
      fields: cleaned,
    }
    setError('')
    const onErr = (e: unknown) =>
      setError(e instanceof ApiError ? e.message : 'Could not save the method.')
    if (method) {
      updateM.mutate({ id: method.id, input }, { onSuccess: onClose, onError: onErr })
    } else {
      createM.mutate(input, { onSuccess: onClose, onError: onErr })
    }
  }

  return (
    <Modal onClose={onClose} maxWidth={640}>
      <div style={{ padding: 22, maxHeight: '82vh', overflowY: 'auto' }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 16 }}>
          {method ? 'Edit payment method' : 'New payment method'}
        </h3>

        <label className="alabel">Name</label>
        <input
          className="afield"
          placeholder="e.g. Bank Transfer"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />

        <label className="alabel" style={{ marginTop: 12 }}>
          Instructions (how the customer pays — e.g. the account to send to)
        </label>
        <textarea
          className="afield"
          rows={3}
          placeholder="Transfer to IBAN … then upload your receipt below."
          value={instructions}
          onChange={(e) => setInstructions(e.target.value)}
        />

        <div style={{ display: 'flex', gap: 20, alignItems: 'center', marginTop: 14 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <Toggle on={enabled} onClick={() => setEnabled((v) => !v)} />
            <span style={{ fontSize: 13.5 }}>Shown to customers</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span className="alabel" style={{ margin: 0 }}>
              Sort order
            </span>
            <input
              className="afield"
              type="number"
              style={{ width: 90 }}
              value={sortOrder}
              onChange={(e) => setSortOrder(Number(e.target.value))}
            />
          </div>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', marginTop: 20, marginBottom: 8 }}>
          <b style={{ fontSize: 14 }}>Customer inputs</b>
          <button
            className="abtn xs"
            style={{ marginInlineStart: 'auto' }}
            onClick={() => setFields((fs) => [...fs, blankField()])}
          >
            <Icon name="plus" size={13} /> Add input
          </button>
        </div>

        {fields.length === 0 && (
          <div className="faint" style={{ fontSize: 13, padding: '4px 0 8px' }}>
            No inputs — the customer just sees the instructions and submits the amount.
          </div>
        )}

        {fields.map((f, i) => (
          <div
            key={i}
            style={{
              border: '1px solid var(--border)',
              borderRadius: 10,
              padding: 12,
              marginBottom: 10,
            }}
          >
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'flex-end' }}>
              <div style={{ flex: '1 1 180px' }}>
                <label className="alabel">Label</label>
                <input
                  className="afield"
                  placeholder="e.g. Transfer receipt"
                  value={f.label}
                  onChange={(e) => setField(i, { label: e.target.value })}
                />
              </div>
              <div style={{ flex: '0 1 160px' }}>
                <label className="alabel">Type</label>
                <select
                  className="afield"
                  value={f.type}
                  onChange={(e) => setField(i, { type: e.target.value as MethodFieldType })}
                >
                  {FIELD_TYPES.map(([v, l]) => (
                    <option key={v} value={v}>
                      {l}
                    </option>
                  ))}
                </select>
              </div>
              <label
                style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, paddingBottom: 10 }}
              >
                <input
                  type="checkbox"
                  checked={f.required}
                  onChange={(e) => setField(i, { required: e.target.checked })}
                />
                Required
              </label>
              <button
                className="abtn xs danger"
                style={{ marginBottom: 4 }}
                onClick={() => setFields((fs) => fs.filter((_, idx) => idx !== i))}
              >
                <Icon name="trash" size={13} />
              </button>
            </div>
            {f.type === 'select' && (
              <div style={{ marginTop: 8 }}>
                <label className="alabel">Options (comma-separated)</label>
                <input
                  className="afield"
                  placeholder="BLOM, Bankmed, Fransabank"
                  value={(f.options ?? []).join(', ')}
                  onChange={(e) => setField(i, { options: e.target.value.split(',') })}
                />
              </div>
            )}
          </div>
        ))}

        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}

        <div style={{ display: 'flex', gap: 10, marginTop: 18, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button className="abtn primary" onClick={save} disabled={saving}>
            <Icon name="check" size={15} /> {method ? 'Save' : 'Create'}
          </button>
        </div>
      </div>
    </Modal>
  )
}
