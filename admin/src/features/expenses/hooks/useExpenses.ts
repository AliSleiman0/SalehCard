import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listExpenses,
  getExpenseSummary,
  getExpense,
  createExpense,
  updateExpense,
  deleteExpense,
  type ExpenseInput,
  type ExpenseListParams,
  type ExpenseSummaryParams,
} from '../api/expenses'

export function useExpenses(params: ExpenseListParams = {}) {
  return useQuery({
    queryKey: ['admin', 'expenses', params],
    queryFn: () => listExpenses(params),
    placeholderData: keepPreviousData,
  })
}

export function useExpenseSummary(params: ExpenseSummaryParams = {}) {
  return useQuery({
    queryKey: ['admin', 'expenses', 'summary', params],
    queryFn: () => getExpenseSummary(params),
    placeholderData: keepPreviousData,
  })
}

export function useExpense(id: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'expense', id],
    queryFn: () => getExpense(id!),
    enabled: !!id,
  })
}

export function useCreateExpense() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ExpenseInput) => createExpense(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'expenses'] }),
  })
}

export function useUpdateExpense(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ExpenseInput) => updateExpense(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'expenses'] })
      qc.invalidateQueries({ queryKey: ['admin', 'expense', id] })
    },
  })
}

export function useDeleteExpense() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteExpense(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'expenses'] }),
  })
}
