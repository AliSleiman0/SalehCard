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
