import type { Product, UnitPricing } from '@/types'
import type { InputFieldRow } from './fieldRows'

// The product editor's working shape for per-unit ("customer enters amount")
// pricing. blockUnits stays a string while editing (the VariantRow pattern);
// fieldKey may be '' or stale (the admin renamed the field) — driverRow then
// falls back to the first amount-type row.
export interface UnitPricingForm {
  enabled: boolean
  fieldKey: string
  blockUnits: string // '' = unset
}

const num = (s: string): number | undefined => {
  const t = s.trim()
  if (!t) return undefined
  const n = parseFloat(t)
  return Number.isFinite(n) ? n : undefined
}

const fmt = new Intl.NumberFormat('en-US', { maximumFractionDigits: 2 })

/** en-US grouping for unit counts ("45,000"). */
export const formatUnits = (n: number): string => fmt.format(n)

// toForm reads a loaded product's spec; a product without one starts in
// fixed-packages mode with the conventional 1,000-unit block pre-filled.
export function toForm(p?: Pick<Product, 'unitPricing'> | null): UnitPricingForm {
  const u = p?.unitPricing
  if (!u || !(u.blockUnits > 0)) return { enabled: false, fieldKey: '', blockUnits: '1000' }
  return { enabled: true, fieldKey: u.fieldKey, blockUnits: String(u.blockUnits) }
}

/** The amount-type rows that can drive the price. */
export const amountRows = (rows: InputFieldRow[]): InputFieldRow[] =>
  rows.filter((r) => r.type === 'amount')

type AmountCandidate = { key: string; type: string; label?: { en?: string } }

// amountIgnoredByPrice names the amount fields on a fixed-price account-credit
// product: the customer types an amount at checkout but is charged the flat
// package price (the trap the migrated live-app products fell into). Empty when
// the product is per-unit, or not account credit — money-transfer products
// legitimately collect an amount alongside a flat fee.
export function amountIgnoredByPrice(p: {
  fulfillmentType: string
  unitPricing?: Pick<UnitPricing, 'blockUnits'> | null
  inputFields?: AmountCandidate[] | null
}): string[] {
  if (p.fulfillmentType !== 'account_credit') return []
  if (p.unitPricing && p.unitPricing.blockUnits > 0) return []
  return (p.inputFields ?? []).filter((f) => f.type === 'amount').map((f) => f.label?.en?.trim() || f.key)
}

// driverRow resolves the field that drives the amount: the chosen key when it
// still names an amount row, else the first amount row (the default).
export function driverRow(form: UnitPricingForm, rows: InputFieldRow[]): InputFieldRow | undefined {
  const amounts = amountRows(rows)
  return amounts.find((r) => r.key.trim() === form.fieldKey.trim()) ?? amounts[0]
}

// fromForm converts the form to the API spec, or undefined when per-unit is
// off. The caller decides between omit and the clear sentinel (buildInput).
export function fromForm(form: UnitPricingForm, rows: InputFieldRow[]): UnitPricing | undefined {
  if (!form.enabled) return undefined
  return {
    fieldKey: driverRow(form, rows)?.key.trim() ?? '',
    blockUnits: num(form.blockUnits) ?? 0,
  }
}

/** The denomination auto-written on the single per-unit variant ("per 1,000"). */
export function unitDenomination(form: UnitPricingForm): string {
  const b = num(form.blockUnits)
  return b && b > 0 ? `per ${formatUnits(b)}` : 'per unit'
}

// issues reports pre-submit problems with per-unit pricing, mirroring the
// server's validateUnitPricing. Empty when per-unit is off.
export function issues(
  form: UnitPricingForm,
  rows: InputFieldRow[],
  variants: { price: string }[],
): string[] {
  if (!form.enabled) return []
  const out: string[] = []
  const b = num(form.blockUnits)
  if (b === undefined || b <= 0) out.push('Per-unit pricing: set how many units the price is for (above 0).')
  const price = num(variants[0]?.price ?? '')
  if (price === undefined || price <= 0) out.push('Per-unit pricing: set a customer price above 0.')
  const driver = driverRow(form, rows)
  if (!driver) {
    out.push('Per-unit pricing: add an input field of type "amount" for the customer to type the amount.')
  } else {
    const min = num(driver.min)
    const max = num(driver.max)
    if (min === undefined || min <= 0 || max === undefined || max < min)
      out.push(`Per-unit pricing: the amount field "${driver.key.trim() || '?'}" needs a min above 0 and a max ≥ min.`)
  }
  for (const r of rows) {
    if (r.type === 'quantity')
      out.push(`Per-unit pricing: remove the quantity field "${r.key.trim() || '?'}" — per-unit products are bought by amount.`)
  }
  return out
}

// enableRows prepares the input fields for switching to per-unit: drops every
// quantity-type field (reporting their keys so the editor can say so) and
// appends a blank-bounded `amount` field when there is no amount field yet.
// `stash` records exactly what changed so disableRows can undo it if the admin
// switches back to fixed packages before saving.
export interface UnitRowsStash {
  /** removed quantity rows with their original positions */
  removed: { index: number; row: InputFieldRow }[]
  /** the auto-added amount row (by reference — editing it replaces the object) */
  added?: InputFieldRow
}

export function enableRows(rows: InputFieldRow[]): {
  rows: InputFieldRow[]
  removed: string[]
  stash: UnitRowsStash
} {
  const removedRows = rows
    .map((row, index) => ({ index, row }))
    .filter((x) => x.row.type === 'quantity')
  const removed = removedRows.map((x) => x.row.key.trim() || '(unnamed)')
  const next = rows.filter((r) => r.type !== 'quantity')
  let added: InputFieldRow | undefined
  if (amountRows(next).length === 0) {
    added = {
      key: next.some((r) => r.key.trim() === 'amount') ? 'units' : 'amount',
      label: { en: 'Enter Amount', ar: 'ادخل العدد المطلوب' },
      type: 'amount',
      sensitive: false,
      min: '',
      max: '',
      options: [],
    }
    next.push(added)
  }
  return { rows: next, removed, stash: { removed: removedRows, added } }
}

// disableRows undoes enableRows when the admin goes back to fixed packages
// before saving: the auto-added amount row is dropped if still untouched, and
// the removed quantity rows go back at their original positions.
export function disableRows(rows: InputFieldRow[], stash: UnitRowsStash | null): InputFieldRow[] {
  if (!stash) return rows
  const next = stash.added ? rows.filter((r) => r !== stash.added) : [...rows]
  for (const { index, row } of [...stash.removed].sort((a, b) => a.index - b.index)) {
    next.splice(Math.min(index, next.length), 0, row)
  }
  return next
}

// droppedPackages lists the extra packages (rows 2..N that would otherwise be
// saved) that a save in per-unit mode deletes — per-unit keeps one price row.
export function droppedPackages(variants: { denomination: string; price: string }[]): string[] {
  return variants
    .slice(1)
    .filter((v) => v.denomination.trim() && v.price.trim())
    .map((v) => v.denomination.trim())
}

// fixedDenomination is the package label to put back when switching to fixed
// packages: a still-auto "per N" label becomes the plain block size ("1,000",
// i.e. the package the price buys); any admin-typed label is kept.
export function fixedDenomination(denomination: string, form: UnitPricingForm): string {
  if (denomination.trim() !== unitDenomination(form)) return denomination
  const b = num(form.blockUnits)
  return b && b > 0 ? formatUnits(b) : ''
}

// preview renders the editor's live example line, e.g.
// "Customer enters 100 – 200,000 · e.g. 100 = $1.75" (sample amount = min,
// priced like the server's UnitLinePrice). '' until the inputs are usable.
export function preview(form: UnitPricingForm, rows: InputFieldRow[], price: string): string {
  const driver = driverRow(form, rows)
  const b = num(form.blockUnits)
  const p = num(price)
  const min = driver ? num(driver.min) : undefined
  const max = driver ? num(driver.max) : undefined
  if (!b || b <= 0 || !p || p <= 0 || !min || min <= 0) return ''
  const line = Math.round(((p * min) / b) * 100) / 100
  const range = max !== undefined && max >= min ? `${formatUnits(min)} – ${formatUnits(max)}` : `at least ${formatUnits(min)}`
  return `Customer enters ${range} · e.g. ${formatUnits(min)} = $${line.toFixed(2)}`
}
