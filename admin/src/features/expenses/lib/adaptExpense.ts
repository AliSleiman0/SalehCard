import type { AdminExpense, ExpenseCategory } from '../api/expenses'

/** The flat shape the expense list renders. */
export interface ExpenseView {
  id: string
  amount: number
  currency: string
  /** Currency-formatted amount, e.g. "$1,200.00". */
  amountLabel: string
  category: ExpenseCategory
  /** Free-text label when category is "other" (empty otherwise). */
  categoryOther: string
  note: string
  /** "Mon D, YYYY" label for incurredAt. */
  dateLabel: string
  raw: AdminExpense
}

/** Format an amount in its currency, falling back to a plain number on error. */
export function amountLabel(amount: number, currency: string): string {
  try {
    return new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(amount)
  } catch {
    return `${amount.toFixed(2)} ${currency}`
  }
}

/** "Mon D, YYYY" label for an ISO date, or '—' when absent/unparseable. */
export function dateLabel(iso?: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

/** Map an admin expense to the flat view the list/table renders. */
export function adaptExpense(e: AdminExpense): ExpenseView {
  return {
    id: e.id,
    amount: e.amount,
    currency: e.currency,
    amountLabel: amountLabel(e.amount, e.currency),
    category: e.category,
    categoryOther: e.categoryOther ?? '',
    note: e.note ?? '',
    dateLabel: dateLabel(e.incurredAt),
    raw: e,
  }
}
