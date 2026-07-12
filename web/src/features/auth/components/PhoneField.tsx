// PhoneField renders a fixed +961 country box next to a digits-only input,
// mirroring the mobile CountryCodeBox. The parent normalizes via toE164Lebanon.
export function PhoneField({
  label,
  value,
  onChange,
  error,
  autoFocus,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  error?: string
  autoFocus?: boolean
}) {
  return (
    <div className="col" style={{ gap: 6 }}>
      <label className="label">{label}</label>
      <div className="row" style={{ gap: 8, alignItems: 'stretch' }}>
        <span
          className="field"
          style={{
            flex: 'none',
            width: 64,
            display: 'grid',
            placeItems: 'center',
            fontWeight: 700,
            color: 'var(--text-dim)',
          }}
        >
          +961
        </span>
        <input
          className="field"
          style={{ flex: 1 }}
          type="tel"
          inputMode="tel"
          autoComplete="tel-national"
          autoFocus={autoFocus}
          placeholder="70 123 456"
          value={value}
          onChange={(e) => onChange(e.target.value)}
        />
      </div>
      {error && (
        <span className="tiny" style={{ color: 'var(--danger)', fontWeight: 600 }}>
          {error}
        </span>
      )}
    </div>
  )
}
