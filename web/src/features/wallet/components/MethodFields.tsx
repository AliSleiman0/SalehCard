import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { uploadTopUpDocument, type TopUpMethodField } from '../api/wallet'
import type { MethodValues } from '../lib/methodFields'

/**
 * Renders the inputs an admin defined on a manual top-up method (text, number,
 * select, file). File fields upload immediately and store the returned URL, so
 * the submit is a plain JSON request (same flow as the app's top-up screen).
 */
export function MethodFields({
  fields,
  values,
  onChange,
  onUploadingChange,
}: {
  fields: TopUpMethodField[]
  values: MethodValues
  onChange: (key: string, value: string) => void
  onUploadingChange?: (uploading: boolean) => void
}) {
  const { t } = useTranslation()
  const [uploading, setUploading] = useState<Record<string, boolean>>({})
  const [errors, setErrors] = useState<Record<string, string>>({})

  const setBusy = (key: string, busy: boolean) => {
    setUploading((u) => {
      const next = { ...u, [key]: busy }
      onUploadingChange?.(Object.values(next).some(Boolean))
      return next
    })
  }

  const upload = async (f: TopUpMethodField, file: File | undefined) => {
    if (!file) return
    setErrors((e) => ({ ...e, [f.key]: '' }))
    setBusy(f.key, true)
    try {
      const res = await uploadTopUpDocument(file)
      if (res.success && res.data?.url) onChange(f.key, res.data.url)
      else setErrors((e) => ({ ...e, [f.key]: res.error?.message ?? t('failed_title') }))
    } catch {
      setErrors((e) => ({ ...e, [f.key]: t('failed_title') }))
    } finally {
      setBusy(f.key, false)
    }
  }

  return (
    <div className="col" style={{ gap: 12, marginBottom: 12 }}>
      {fields.map((f) => {
        const label = (
          <div className="label">
            {f.label || f.key}
            {f.required && <span style={{ color: 'var(--danger, #e5484d)' }}> *</span>}
          </div>
        )
        const v = values[f.key] ?? ''
        if (f.type === 'select') {
          return (
            <div key={f.key}>
              {label}
              <select className="field" value={v} onChange={(e) => onChange(f.key, e.target.value)}>
                <option value="">{f.placeholder || '—'}</option>
                {(f.options ?? []).map((o) => (
                  <option key={o} value={o}>
                    {o}
                  </option>
                ))}
              </select>
            </div>
          )
        }
        if (f.type === 'file') {
          return (
            <div key={f.key}>
              {label}
              <input
                className="field"
                type="file"
                accept="image/*"
                aria-label={f.label || f.key}
                disabled={!!uploading[f.key]}
                onChange={(e) => void upload(f, e.target.files?.[0])}
              />
              {uploading[f.key] && <span className="tiny faint">{t('processing')}</span>}
              {!uploading[f.key] && v && (
                <a className="tiny" href={v} target="_blank" rel="noreferrer" style={{ color: 'var(--ok)' }}>
                  ✓ {t('uploaded', { defaultValue: 'Uploaded' })}
                </a>
              )}
              {errors[f.key] && (
                <span className="tiny" style={{ color: 'var(--danger, #e5484d)' }}>
                  {errors[f.key]}
                </span>
              )}
            </div>
          )
        }
        return (
          <div key={f.key}>
            {label}
            <input
              className={'field' + (f.type === 'number' ? ' num' : '')}
              inputMode={f.type === 'number' ? 'decimal' : undefined}
              placeholder={f.placeholder}
              aria-label={f.label || f.key}
              value={v}
              onChange={(e) => onChange(f.key, e.target.value)}
            />
          </div>
        )
      })}
    </div>
  )
}
