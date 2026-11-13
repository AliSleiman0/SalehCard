export const RATES = { USD: 1, TRY: 34.2 } as const
export const SYM = { USD: '$', TRY: '₺' } as const

export type Currency = 'USD' | 'TRY'

export function fmtPrice(usd: number, cur: Currency): string {
  const v = usd * (RATES[cur] || 1)
  const s = SYM[cur] || '$'
  const grouped = Number(v).toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
  return cur === 'TRY' ? `${grouped} ${s}` : `${s}${grouped}`
}

export function cn(...classes: (string | false | null | undefined)[]): string {
  return classes.filter(Boolean).join(' ')
}

export function isRTL(locale: string): boolean {
  return locale === 'ar'
}

/** Absolute date + time, e.g. "Nov 12, 2026 · 03:05 PM" ('' for a bad date). */
export function fmtDateTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return (
    d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }) +
    ' · ' +
    d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
  )
}
