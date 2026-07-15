import { fmtPrice, cn, type Currency } from '@/lib/utils'
import { useCurrencyStore } from '@/stores/currency'

export function Price({
  usd,
  cur,
  strike = false,
  className = '',
}: {
  usd: number
  cur?: Currency
  strike?: boolean
  className?: string
}) {
  const storeCur = useCurrencyStore((s) => s.currency)
  return (
    // nowrap so a formatted price is never split mid-number, even when an
    // ancestor sets overflow-wrap:anywhere for long ids (see .num in tokens.css).
    <span className={cn('num', strike && 'strike', className)} style={{ whiteSpace: 'nowrap' }}>
      {fmtPrice(usd, cur ?? storeCur)}
    </span>
  )
}
