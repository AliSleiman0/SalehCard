import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, ImageArt, Price, Badge, Button } from '@/components'
import { useCurrencyStore } from '@/stores/currency'
import type { OrderView } from '@/features/orders/types'

export function OrderRow({ o }: { o: OrderView }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const cur = useCurrencyStore((s) => s.currency)
  const done = ['delivered', 'completed'].includes(o.status ?? '')
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
          {o.id} · {o.date}
        </span>
      </div>
      <Badge variant={done ? 'instant' : 'soft'}>
        {done ? <Icon name="check" size={11} /> : null}
        {t(o.status ?? '')}
      </Badge>
      <Price usd={o.total} cur={cur} className="num small" />
      <Button variant="ghost" size="sm" onClick={() => navigate('/orders/' + o.id)}>
        <Icon name="chevron" size={15} />
      </Button>
    </div>
  )
}
