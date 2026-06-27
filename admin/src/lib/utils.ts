import type { Locale, StockLevel } from '@/types'

export function cn(...classes: (string | false | null | undefined)[]): string {
  return classes.filter(Boolean).join(' ')
}

export function isRTL(locale: string): boolean {
  return locale === 'ar'
}

/** Money formatter matching the admin design's `money()` helper. */
export function money(v: number, cur: 'USD' | 'TRY' = 'USD'): string {
  const sign = v < 0 ? '−' : ''
  const a = Math.abs(v)
  const s = a.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
  return cur === 'TRY' ? `${sign}₺${s}` : `${sign}$${s}`
}

/** Short relative time ("2m ago", "Yesterday", "Jun 3") from an ISO timestamp. */
export function relativeTime(iso: string): string {
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return '—'
  const diff = Math.max(0, Date.now() - then)
  const m = Math.floor(diff / 60_000)
  if (m < 1) return 'Just now'
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  const d = Math.floor(h / 24)
  if (d === 1) return 'Yesterday'
  if (d < 7) return `${d}d ago`
  return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

/** Stock level from available count vs threshold — green / yellow / red. */
export function stockLevel(available: number, threshold: number): StockLevel {
  if (available <= 0 || available < threshold * 0.4) return 'lo'
  if (available < threshold) return 'mid'
  return 'hi'
}

export const LOCALE_NAMES: Record<Locale, string> = {
  en: 'English',
  ar: 'العربية',
  tr: 'Türkçe',
}

export const LOCALE_FLAGS: Record<Locale, string> = {
  en: 'EN',
  ar: 'AR',
  tr: 'TR',
}
