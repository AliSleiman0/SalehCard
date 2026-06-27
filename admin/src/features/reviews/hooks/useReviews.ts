import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listReviews,
  setReviewStatus,
  deleteReview,
  type ReviewListParams,
  type ReviewStatus,
} from '../api/reviews'

export function useReviews(params: ReviewListParams = {}) {
  return useQuery({
    queryKey: ['admin', 'reviews', params],
    queryFn: () => listReviews(params),
    placeholderData: keepPreviousData,
  })
}

export function useSetReviewStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: ReviewStatus }) => setReviewStatus(id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'reviews'] }),
  })
}

export function useDeleteReview() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteReview(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'reviews'] }),
  })
}
