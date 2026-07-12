import { useTranslation } from 'react-i18next'
import { useLocaleStore } from '@/stores/locale'
import type { InputField } from '@/types'

// DynamicField renders one product InputField (text | amount | select; the
// legacy `quantity` type is filtered upstream). Sensitive fields are masked.
// Controlled by the parent — value + onChange. Port of dynamic_input_field.dart.
export function DynamicField({
  field,
  value,
  onChange,
  error,
}: {
  field: InputField
  value: string
  onChange: (v: string) => void
  error?: string
}) {
  const { t } = useTranslation()
  const locale = useLocaleStore((s) => s.locale)
  const label = field.label[locale] || field.label.en || field.key
  const options = field.constraints?.options ?? []

  return (
    <div className="col" style={{ gap: 8 }}>
      <div className="row" style={{ gap: 7, alignItems: 'center' }}>
        <span className="label" style={{ margin: 0 }}>
          {label}
        </span>
        <span className="badge badge-soft" style={{ fontSize: 10 }}>
          {t('required')}
        </span>
      </div>
      {field.type === 'select' ? (
        <select
          className="field"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          style={error ? { borderColor: 'var(--danger)' } : undefined}
        >
          <option value="" disabled>
            {t('enter_value', { label })}
          </option>
          {options.map((opt) => (
            <option key={opt} value={opt}>
              {opt}
            </option>
          ))}
        </select>
      ) : (
        <input
          className="field"
          type={field.sensitive ? 'password' : 'text'}
          inputMode={field.key === 'phone' ? 'tel' : field.type === 'amount' ? 'decimal' : undefined}
          placeholder={t('enter_value', { label })}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          style={error ? { borderColor: 'var(--danger)' } : undefined}
        />
      )}
      {error && (
        <span className="tiny" style={{ color: 'var(--danger)', fontWeight: 600 }}>
          {error}
        </span>
      )}
    </div>
  )
}
