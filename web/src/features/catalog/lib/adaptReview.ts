import type { ApiReview } from '../api/reviews'

// ViewReview is the presentational shape the product-page review card renders.
export interface ViewReview {
  id: string
  initials: string
  name: string
  stars: number
  body: string
  date: string
  verified: boolean
}

function initialsOf(name: string): string {
  const parts = name.trim().split(/[\s._-]+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[1][0]).toUpperCase()
}

function fmtDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

export function adaptReview(r: ApiReview): ViewReview {
  return {
    id: r.id,
    initials: initialsOf(r.userName),
    name: r.userName,
    stars: r.rating,
    body: r.body,
    date: fmtDate(r.createdAt),
    verified: r.verifiedPurchase,
  }
}
