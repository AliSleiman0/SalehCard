import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon } from './Icon'
import { cn } from '@/lib/utils'

export interface Sort {
  k: string
  dir: 'asc' | 'desc'
}

/** Sortable table header cell (`.sortable`). */
export function SortTh({
  label,
  k,
  sort,
  setSort,
  className,
}: {
  label: React.ReactNode
  k: string
  sort: Sort | null
  setSort: (s: Sort) => void
  className?: string
}) {
  const active = sort && sort.k === k
  return (
    <th
      className={cn('sortable', className)}
      onClick={() => setSort({ k, dir: active && sort!.dir === 'asc' ? 'desc' : 'asc' })}
    >
      {label}
      <span className="car">
        <Icon name={active ? (sort!.dir === 'asc' ? 'chevup' : 'chevdown') : 'sort'} size={13} />
      </span>
    </th>
  )
}

/** Footer pagination control. Page state is local (display-only until wired). */
export function Pagination({
  page = 1,
  pages = 1,
  total = 0,
  shown,
  label,
  limit,
  onPage,
}: {
  page?: number
  pages?: number
  total?: number
  shown?: number
  label?: React.ReactNode
  /** Page size — used to compute the "Showing X–Y" range on pages after the
   *  first. Falls back to the shown count (correct only on page 1). */
  limit?: number
  /** When provided, the component is controlled: clicks call onPage instead of
   *  tracking page internally (so the caller can refetch). */
  onPage?: (page: number) => void
}) {
  const { t } = useTranslation()
  const [internal, setInternal] = useState(page)
  const p = onPage ? page : internal
  const go = (n: number) => {
    const clamped = Math.max(1, Math.min(pages, n))
    if (onPage) onPage(clamped)
    else setInternal(clamped)
  }
  const count = shown ?? Math.min(total, 14)
  const from = total === 0 ? 0 : (p - 1) * (limit ?? count) + 1
  const to = total === 0 ? 0 : from + count - 1
  return (
    <div className="pagination">
      <span>
        {t('showing')} <b>{from}–{to}</b> {t('of')} <b className="num">{total.toLocaleString()}</b>{' '}
        {label ?? t('results')}
      </span>
      <div className="sp" style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
        <div className="pg" onClick={() => go(p - 1)}>
          <Icon name="chevleft" size={15} />
        </div>
        {Array.from({ length: pages }, (_, i) => i + 1).map((n) => (
          <div key={n} className={cn('pg', n === p && 'on')} onClick={() => go(n)}>
            {n}
          </div>
        ))}
        <div className="pg" onClick={() => go(p + 1)}>
          <Icon name="chevright" size={15} />
        </div>
      </div>
    </div>
  )
}
