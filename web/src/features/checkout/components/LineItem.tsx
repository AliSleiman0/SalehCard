import { Icon, ImageArt, Price, Stepper } from '@/components'
import { useCurrencyStore } from '@/stores/currency'
import type { CartItem } from '@/stores/cart'

export function LineItem({
  it,
  onQty,
  onRemove,
}: {
  it: CartItem
  onQty?: (key: string, qty: number) => void
  onRemove?: (key: string) => void
}) {
  const cur = useCurrencyStore((s) => s.currency)
  return (
    <div className="lrow">
      <ImageArt
        art={it.art}
        word={it.brand.split(' ')[0]}
        h={56}
        wordSize={14}
        radius={12}
        style={{ width: 74, flex: 'none' }}
      />
      <div className="col" style={{ gap: 3, flex: 1, minWidth: 0 }}>
        <span style={{ fontWeight: 800 }}>{it.brand}</span>
        <span className="small muted">
          {it.title} · {it.variant}
        </span>
        {it.pid && <span className="tiny faint num">ID: {it.pid}</span>}
      </div>
      {onQty ? (
        <Stepper value={it.qty} set={(q) => onQty(it.key, q)} />
      ) : (
        <span className="num muted">×{it.qty}</span>
      )}
      <div className="col" style={{ alignItems: 'flex-end', gap: 6, minWidth: 80 }}>
        <Price usd={it.price * it.qty} cur={cur} className="num" />
        {onRemove && (
          <button
            className="icon-btn"
            style={{ width: 30, height: 30 }}
            onClick={() => onRemove(it.key)}
          >
            <Icon name="trash" size={15} />
          </button>
        )}
      </div>
    </div>
  )
}
