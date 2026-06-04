import { useState } from 'react'
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
  label = 'results',
}: {
  page?: number
  pages?: number
  total?: number
  shown?: number
  label?: string
}) {
  const [p, setP] = useState(page)
  const upTo = shown ?? Math.min(total, 14)
  return (
    <div className="pagination">
      <span>
        Showing <b>1–{upTo}</b> of <b className="num">{total.toLocaleString()}</b> {label}
      </span>
      <div className="sp" style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
        <div className="pg" onClick={() => setP(Math.max(1, p - 1))}>
          <Icon name="chevleft" size={15} />
        </div>
        {Array.from({ length: pages }, (_, i) => i + 1).map((n) => (
          <div key={n} className={cn('pg', n === p && 'on')} onClick={() => setP(n)}>
            {n}
          </div>
        ))}
        <div className="pg" onClick={() => setP(Math.min(pages, p + 1))}>
          <Icon name="chevright" size={15} />
        </div>
      </div>
    </div>
  )
}
