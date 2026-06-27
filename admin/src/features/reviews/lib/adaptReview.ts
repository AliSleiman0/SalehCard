import { relativeTime } from '@/lib/utils'
import type { AdminReview } from '../api/reviews'

/** The flat shape the moderation list renders. */
export interface ReviewView {
  id: string
  product: string
  art: string
  user: string
  rating: number
  body: string
  /** Short relative time, e.g. "2m ago", "Yesterday". */
  date: string
  status: AdminReview['status']
  verified: boolean
  /** 1-star reviews are surfaced as possible spam. */
  flagged: boolean
  raw: AdminReview
}

/** Map an admin review to the flat view the list renders. */
export function adaptReview(r: AdminReview): ReviewView {
  return {
    id: r.id,
    product: r.productName || 'Unknown product',
    art: r.art,
    user: r.userName || 'Unknown',
    rating: r.rating,
    body: r.body,
    date: relativeTime(r.createdAt),
    status: r.status,
    verified: r.verifiedPurchase,
    flagged: r.rating <= 1,
    raw: r,
  }
}
