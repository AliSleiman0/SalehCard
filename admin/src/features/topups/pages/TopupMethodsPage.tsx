import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Icon,
  PageHead,
  Modal,
  Toggle,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { apiErrorText } from '@/lib/apiErrorText'
import { useCan } from '@/stores/auth'
import {
  useTopupMethods,
  useCreateMethod,
  useUpdateMethod,
  useDeleteMethod,
} from '../hooks/useTopupMethods'
import { METHOD_KINDS } from '../api/methods'
import type { MethodField, MethodFieldType, MethodInput, MethodKind, TopUpMethod } from '../api/methods'

const FIELD_TYPES: MethodFieldType[] = ['text', 'number', 'select', 'file']

/** Translation key for a method kind ('' = not classified yet). */
const kindKey = (k: string) => `top_kind_${k || 'none'}_name`

/** slugify turns a label into a stable field key (lowercase, alnum + underscore). */
function slugify(s: string): string {
  return s
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
}

export default function TopupMethodsPage() {
  const { t } = useTranslation()
  const canManage = useCan()('topups.manage')
  const { data, isLoading, isError, refetch } = useTopupMethods()
  const methods = data?.data ?? []
  const [editing, setEditing] = useState<TopUpMethod | 'new' | null>(null)
  const delM = useDeleteMethod()

  return (
    <div className="page page-wide">
      <PageHead
        back="/topups"
        crumbs={[t('grp_finance'), t('nav_topup_methods')]}
        title={t('top_methods_title')}
        sub={t('top_methods_sub')}
      >
        {canManage && (
          <button className="abtn primary" onClick={() => setEditing('new')}>
            <Icon name="plus" size={15} /> {t('top_new_method')}
          </button>
        )}
      </PageHead>

      <div className="acard">
        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message={t('top_methods_load_error')} onRetry={() => refetch()} />
        ) : methods.length === 0 ? (
          <EmptyState title={t('top_methods_empty')} sub={t('top_methods_empty_sub')} />
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
                      {t('top_enabled')}
                    </span>
                  ) : (
                    <span className="bdg">{t('top_disabled')}</span>
                  )}
                  <span className="bdg">
                    {METHOD_KINDS.some((k) => k.value === (m.kind ?? '')) ? t(kindKey(m.kind ?? '')) : m.kind}
                  </span>
                  <span className="faint" style={{ fontSize: 12.5 }}>
                    {t('top_inputs', { count: m.fields.length })}
                  </span>
                  {canManage && (
                    <div style={{ marginInlineStart: 'auto', display: 'flex', gap: 8 }}>
                      <button className="abtn xs" onClick={() => setEditing(m)}>
                        <Icon name="edit" size={13} /> {t('edit')}
                      </button>
                      <button
                        className="abtn xs danger"
                        disabled={delM.isPending}
                        onClick={() => {
                          if (confirm(t('top_method_delete_confirm', { name: m.name }))) {
                            delM.mutate(m.id)
                          }
                        }}
                      >
                        <Icon name="trash" size={13} /> {t('delete')}
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
  const { t } = useTranslation()
  const [name, setName] = useState(method?.name ?? '')
  const [kind, setKind] = useState<MethodKind>(method?.kind ?? '')
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
      setError(t('top_err_name_required'))
      return
    }
    // Derive missing keys from labels and enforce uniqueness client-side (the
    // server re-validates).
    const seen = new Set<string>()
    const cleaned: MethodField[] = []
    for (const f of fields) {
      const label = f.label.trim()
      if (!label) {
        setError(t('top_err_label_required'))
        return
      }
      const key = slugify(f.key || label)
      if (!key) {
        setError(t('top_err_no_key', { label }))
        return
      }
      if (seen.has(key)) {
        setError(t('top_err_dup_key', { key }))
        return
      }
      seen.add(key)
      const options =
        f.type === 'select' ? (f.options ?? []).map((o) => o.trim()).filter(Boolean) : undefined
      if (f.type === 'select' && (!options || options.length === 0)) {
        setError(t('top_err_no_options', { label }))
        return
      }
      cleaned.push({ key, label, type: f.type, required: f.required, options })
    }

    const input: MethodInput = {
      name: name.trim(),
      kind,
      instructions: instructions.trim(),
      enabled,
      sortOrder: Number(sortOrder) || 0,
      fields: cleaned,
    }
    setError('')
    const onErr = (e: unknown) => setError(apiErrorText(e, 'top_method_save_failed'))
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
          {method ? t('top_edit_method') : t('top_new_method_title')}
        </h3>

        <label className="alabel">{t('top_name')}</label>
        <input
          className="afield"
          placeholder={t('top_name_ph')}
          value={name}
          onChange={(e) => setName(e.target.value)}
        />

        <label className="alabel" style={{ marginTop: 12 }}>
          {t('col_type')}
        </label>
        <select className="afield" value={kind} onChange={(e) => setKind(e.target.value as MethodKind)}>
          {METHOD_KINDS.map((k) => (
            <option key={k.value} value={k.value}>
              {t(kindKey(k.value), { defaultValue: k.label })}
            </option>
          ))}
        </select>
        <div className="faint" style={{ fontSize: 12, marginTop: 4 }}>
          {t('top_kind_hint')}
        </div>

        <label className="alabel" style={{ marginTop: 12 }}>
          {t('top_instructions')}
        </label>
        <textarea
          className="afield"
          rows={3}
          placeholder={t('top_instructions_ph')}
          value={instructions}
          onChange={(e) => setInstructions(e.target.value)}
        />

        <div style={{ display: 'flex', gap: 20, alignItems: 'center', marginTop: 14 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <Toggle on={enabled} onClick={() => setEnabled((v) => !v)} />
            <span style={{ fontSize: 13.5 }}>{t('top_shown_to_customers')}</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span className="alabel" style={{ margin: 0 }}>
              {t('top_sort_order')}
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
          <b style={{ fontSize: 14 }}>{t('top_customer_inputs')}</b>
          <button
            className="abtn xs"
            style={{ marginInlineStart: 'auto' }}
            onClick={() => setFields((fs) => [...fs, blankField()])}
          >
            <Icon name="plus" size={13} /> {t('top_add_input')}
          </button>
        </div>

        {fields.length === 0 && (
          <div className="faint" style={{ fontSize: 13, padding: '4px 0 8px' }}>
            {t('top_no_inputs')}
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
                <label className="alabel">{t('top_label')}</label>
                <input
                  className="afield"
                  placeholder={t('top_label_ph')}
                  value={f.label}
                  onChange={(e) => setField(i, { label: e.target.value })}
                />
              </div>
              <div style={{ flex: '0 1 160px' }}>
                <label className="alabel">{t('col_type')}</label>
                <select
                  className="afield"
                  value={f.type}
                  onChange={(e) => setField(i, { type: e.target.value as MethodFieldType })}
                >
                  {FIELD_TYPES.map((v) => (
                    <option key={v} value={v}>
                      {t(`top_ft_${v}`)}
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
                {t('top_required')}
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
                <label className="alabel">{t('top_options')}</label>
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
            {t('cancel')}
          </button>
          <button className="abtn primary" onClick={save} disabled={saving}>
            <Icon name="check" size={15} /> {method ? t('top_save') : t('top_create')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
