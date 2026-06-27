import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta } from '@/types'

/** Preset expense categories. "other" carries a free-text label. */
export type ExpenseCategory =
  | 'salary'
  | 'rent'
  | 'utilities'
  | 'inventory'
  | 'marketing'
  | 'fees'
  | 'other'

/** An expense as returned by the admin API (matches the Go model). */
export interface AdminExpense {
  id: string
  amount: number
  currency: string
  category: ExpenseCategory
  categoryOther?: string
  note?: string
  incurredAt: string
  createdAt: string
  updatedAt: string
}

export interface ExpenseListParams {
  page?: number
  limit?: number
  category?: ExpenseCategory | ''
  currency?: string
  /** ISO date (inclusive lower bound on incurredAt). */
  from?: string
  /** ISO date (inclusive upper bound on incurredAt). */
  to?: string
  /** Note substring search. */
  q?: string
}

/** Filters for the totals summary (same shape as the list filters). */
export type ExpenseSummaryParams = Omit<ExpenseListParams, 'page' | 'limit'>

/** Create/update payload. */
export interface ExpenseInput {
  amount: number
  currency: string
  category: ExpenseCategory
  categoryOther?: string
  note?: string
  /** ISO timestamp for the date the expense applies to. */
  incurredAt: string
}

/** Grand total of spend in one currency. */
export interface CurrencyTotal {
  currency: string
  total: number
  count: number
}

/** Spend within a (currency, category) bucket. */
export interface CategoryTotal {
  category: ExpenseCategory
  currency: string
  total: number
  count: number
}

/** Aggregate spend for a filter (computed server-side over all matching rows). */
export interface ExpenseSummary {
  totals: CurrencyTotal[]
  byCategory: CategoryTotal[]
}

const ADMIN = '/api/admin/expenses'

function buildQuery(params: ExpenseListParams): string {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.category) q.set('category', params.category)
  if (params.currency) q.set('currency', params.currency)
  if (params.from) q.set('from', params.from)
  if (params.to) q.set('to', params.to)
  if (params.q) q.set('q', params.q)
  return q.toString()
}

export function listExpenses(
  params: ExpenseListParams,
): Promise<ApiResponse<AdminExpense[]> & { meta?: PaginationMeta }> {
  const qs = buildQuery(params)
  return apiClient.get<AdminExpense[]>(qs ? `${ADMIN}?${qs}` : ADMIN)
}

export function getExpenseSummary(params: ExpenseSummaryParams): Promise<ApiResponse<ExpenseSummary>> {
  const qs = buildQuery(params)
  return apiClient.get<ExpenseSummary>(qs ? `${ADMIN}/summary?${qs}` : `${ADMIN}/summary`)
}

export function getExpense(id: string): Promise<ApiResponse<AdminExpense>> {
  return apiClient.get<AdminExpense>(`${ADMIN}/${id}`)
}

export function createExpense(input: ExpenseInput): Promise<ApiResponse<AdminExpense>> {
  return apiClient.post<AdminExpense>(ADMIN, input)
}

export function updateExpense(id: string, input: ExpenseInput): Promise<ApiResponse<AdminExpense>> {
  return apiClient.put<AdminExpense>(`${ADMIN}/${id}`, input)
}

export function deleteExpense(id: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiClient.delete<{ deleted: boolean }>(`${ADMIN}/${id}`)
}
