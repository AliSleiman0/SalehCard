export const RATES = { USD: 1, TRY: 34.2 } as const
export const SYM = { USD: '$', TRY: '₺' } as const

export type Currency = 'USD' | 'TRY'

export function fmtPrice(usd: number, cur: Currency): string {
  const v = usd * (RATES[cur] || 1)
  const s = SYM[cur] || '$'
  const grouped = Number(v).toLocaleString('en-US', {
    minimumFractionDigits: v >= 100 ? 0 : 2,
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
