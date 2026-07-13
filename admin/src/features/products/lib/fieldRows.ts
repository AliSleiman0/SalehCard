import type { InputField, InputFieldType } from '@/types'

// The product editor's working shape for one input-field row. Numeric bounds
// are kept as strings while editing (the VariantRow pattern — typing "1." must
// never jank a controlled number input); fromRows parses them on save.
export interface InputFieldRow {
  key: string
  label: { en: string; ar: string }
  legacyName?: string
  type: InputFieldType
  sensitive?: boolean
  min: string // '' = unset
  max: string // '' = unset
  options: string[] // select only
}

const num = (s: string): number | undefined => {
  const t = s.trim()
  if (!t) return undefined
  const n = parseFloat(t)
  return Number.isFinite(n) ? n : undefined
}

// toRows converts stored API fields into editable rows. The legacy-corrupt
// {min:0,max:0} import shape means "unbounded", so it renders as empty boxes —
// and because fromRows omits empty boxes, a round-trip repairs it rather than
// writing 0/0 back.
export function toRows(fields: InputField[]): InputFieldRow[] {
  return fields.map((f) => {
    const c = f.constraints
    const corrupt = c?.min === 0 && c?.max === 0
    return {
      key: f.key,
      label: { en: f.label?.en ?? '', ar: f.label?.ar ?? '' },
      ...(f.legacyName ? { legacyName: f.legacyName } : {}),
      type: f.type,
      sensitive: f.sensitive,
      min: !corrupt && c?.min != null ? String(c.min) : '',
      max: !corrupt && c?.max != null ? String(c.max) : '',
      options: c?.options ?? [],
    }
  })
}

// fromRows converts edited rows back to the API shape, emitting only the
// constraints the row's current type supports: min/max for quantity/amount
// (each only when it parses to a finite number — empty boxes are omitted, so
// {min:0,max:0} can never be produced), options for select, nothing for text.
export function fromRows(rows: InputFieldRow[]): InputField[] {
  return rows.map((r) => {
    const base: InputField = {
      key: r.key.trim(),
      label: r.label,
      ...(r.legacyName ? { legacyName: r.legacyName } : {}),
      type: r.type,
      sensitive: !!r.sensitive,
    }
    if (r.type === 'quantity' || r.type === 'amount') {
      const min = num(r.min)
      const max = num(r.max)
      if (min !== undefined || max !== undefined) {
        base.constraints = {
          ...(min !== undefined ? { min } : {}),
          ...(max !== undefined ? { max } : {}),
        }
      }
    } else if (r.type === 'select' && r.options.length > 0) {
      base.constraints = { options: r.options }
    }
    return base
  })
}

// rowIssues reports pre-submit problems the admin must fix in the table before
// the product can be saved. Mirrors the server's sanitizeInputFields rejections,
// plus one stricter rule: the editor never produces an optionless select (the
// server tolerates those only for legacy imports).
export function rowIssues(rows: InputFieldRow[]): string[] {
  const issues: string[] = []
  const seen = new Set<string>()
  rows.forEach((r, i) => {
    const key = r.key.trim()
    if (!key) {
      issues.push(`Input field #${i + 1} needs a key.`)
    } else if (seen.has(key)) {
      issues.push(`Duplicate field key "${key}" — keys must be unique.`)
    } else {
      seen.add(key)
    }
    const name = key || `#${i + 1}`
    if (r.type === 'quantity' || r.type === 'amount') {
      const min = num(r.min)
      const max = num(r.max)
      if (r.min.trim() && min === undefined) issues.push(`Field "${name}": min is not a number.`)
      if (r.max.trim() && max === undefined) issues.push(`Field "${name}": max is not a number.`)
      if ((min !== undefined && min < 0) || (max !== undefined && max < 0))
        issues.push(`Field "${name}": min/max cannot be negative.`)
      if (min !== undefined && max !== undefined && min > max)
        issues.push(`Field "${name}": min cannot exceed max.`)
    }
    if (r.type === 'select' && r.options.length === 0)
      issues.push(`Field "${name}": a select needs at least one option.`)
  })
  return issues
}
