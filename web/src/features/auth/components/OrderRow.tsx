import { useLocation, useNavigate } from 'react-router-dom'
import { Icon, ImageArt, Price, Button } from '@/components'
import { useCurrencyStore } from '@/stores/currency'
import { BalanceLine, OrderStatusBadge } from '@/features/orders/components/OrderParts'
import type { OrderView } from '@/features/orders/types'
import { orderNumber } from '@/lib/orderNumber'

export function OrderRow({ o }: { o: OrderView }) {
  const navigate = useNavigate()
  const location = useLocation()
  const cur = useCurrencyStore((s) => s.currency)
  return (
    <div className="lrow">
      <ImageArt
        art={o.art}
        word={o.product.split(' — ')[0].split(' ')[0]}
        h={48}
        wordSize={12}
        radius={10}
        style={{ width: 64, flex: 'none' }}
      />
      <div className="col" style={{ gap: 2, flex: 1, minWidth: 0 }}>
        <span style={{ fontWeight: 700, fontSize: 14 }}>{o.product}</span>
        <span className="tiny faint num">
          {orderNumber(o.id)} · {o.date}
        </span>
        <BalanceLine o={o} />
      </div>
      <OrderStatusBadge status={o.status} />
      <Price usd={o.total} cur={cur} className="num small" />
      <Button variant="ghost" size="sm" onClick={() =>
          // Remember the list URL (with any ?q= search) for the detail page's back link.
          navigate('/orders/' + o.id, { state: { from: location.pathname + location.search } })
        }>
        <Icon name="chevron" size={15} />
      </Button>
    </div>
  )
}
