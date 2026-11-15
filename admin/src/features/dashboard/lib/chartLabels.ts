import type { TFunction } from 'i18next'
import { formatDate } from '@/lib/utils'

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

/** The revenue chart's axis label in the admin's language. The server sends
 *  English labels: "Jan 2" (daily), "W05" (weekly) and "Jan" (monthly).
 *  English, and anything unrecognised, stays as sent. */
export function chartLabel(label: string, t: TFunction, lang: string): string {
  if (lang.startsWith('en')) return label
  const week = /^W(\d+)$/.exec(label)
  if (week) return t('dash_week', { n: week[1] })
  const day = /^([A-Z][a-z]{2})(?: (\d{1,2}))?$/.exec(label)
  const month = day ? MONTHS.indexOf(day[1]) : -1
  if (!day || month < 0) return label
  if (!day[2]) return formatDate(new Date(2000, month, 1), { month: 'short' })
  return formatDate(new Date(2000, month, Number(day[2])), { month: 'short', day: 'numeric' })
}
