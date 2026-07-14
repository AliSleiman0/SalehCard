import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listCategories,
  createCategory,
  updateCategory,
  deleteCategory,
  type CategoryCreateInput,
  type CategoryUpdateInput,
} from '../api/categories'

const KEY = ['admin', 'categories']

export function useCategoryTree() {
  return useQuery({ queryKey: KEY, queryFn: () => listCategories() })
}

export function useCreateCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CategoryCreateInput) => createCategory(input),
    onSuccess: () => void qc.invalidateQueries({ queryKey: KEY }),
  })
}

export function useUpdateCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: CategoryUpdateInput }) => updateCategory(id, input),
    onSuccess: () => void qc.invalidateQueries({ queryKey: KEY }),
  })
}

export function useDeleteCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteCategory(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: KEY }),
  })
}
