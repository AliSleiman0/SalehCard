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

/** Long human date, e.g. "June 28, 2026". */
export function formatLongDate(d: Date = new Date()): string {
  return d.toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })
}

/** Trigger a client-side PDF download of a titled table. jspdf is dynamically
 *  imported so it stays out of the main bundle (loaded on first PDF export). */
export async function downloadPdf(
  filename: string,
  title: string,
  header: string[],
  rows: (string | number)[][],
): Promise<void> {
  const [{ default: jsPDF }, { default: autoTable }] = await Promise.all([
    import('jspdf'),
    import('jspdf-autotable'),
  ])
  const doc = new jsPDF()
  doc.setFontSize(14)
  doc.text(title, 14, 16)
  autoTable(doc, {
    startY: 22,
    head: [header],
    body: rows.map((r) => r.map((c) => String(c))),
    styles: { fontSize: 8 },
    headStyles: { fillColor: [138, 59, 255] },
  })
  doc.save(filename)
}

/** Trigger a client-side CSV download from a 2-D array of rows (first row = header). */
export function downloadCsv(filename: string, rows: (string | number)[][]): void {
  const esc = (c: string | number): string => {
    const s = String(c)
    return /[",\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
  }
  const csv = rows.map((r) => r.map(esc).join(',')).join('\r\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/** Relative "last active" label, or "—" when the timestamp is missing or the Go
 *  zero value (`0001-01-01T00:00:00Z`, which omitempty does not drop for a
 *  time.Time) — i.e. an account that has never been seen. */
export function lastActive(iso?: string): string {
  if (!iso) return '—'
  const t = new Date(iso).getTime()
  if (Number.isNaN(t) || t <= 0) return '—'
  return relativeTime(iso)
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
