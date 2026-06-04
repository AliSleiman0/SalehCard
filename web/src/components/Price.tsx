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
    <span className={cn('num', strike && 'strike', className)}>
      {fmtPrice(usd, cur ?? storeCur)}
    </span>
  )
}
