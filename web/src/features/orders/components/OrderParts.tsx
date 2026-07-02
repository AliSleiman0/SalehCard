import { useTranslation } from 'react-i18next'
import { Icon, ImageArt, Price, Badge, Button, useToast } from '@/components'
import { useCurrencyStore } from '@/stores/currency'
import type { OrderView } from '../types'

// Single source of truth for order-status badges (head, list rows, credit card).
const STATUS_VARIANT: Record<string, 'instant' | 'soft' | 'disc'> = {
  completed: 'instant',
  refunded: 'disc',
  failed: 'disc',
}

export function OrderStatusBadge({ status }: { status?: string }) {
  const { t } = useTranslation()
  if (!status) return null
  const variant = STATUS_VARIANT[status] ?? 'soft'
  return (
    <Badge variant={variant}>
      {variant === 'instant' ? <Icon name="check" size={11} /> : null}
      {t(status)}
    </Badge>
  )
}

// StatusNotice replaces the fulfillment details for terminal non-success orders
// (refunded / failed), where showing a code vault or "completed" credit card lies.
export function StatusNotice({ status }: { status?: string }) {
  const { t } = useTranslation()
  const note = status === 'failed' ? t('order_failed_note') : t('order_refunded_note')
  return (
    <div
      className="panel card-pad"
      style={{
        border: '1px solid var(--danger)',
        display: 'flex',
        gap: 12,
        alignItems: 'flex-start',
      }}
    >
      <span style={{ color: 'var(--danger)', flex: 'none', marginTop: 2 }}>
        <Icon name="wallet" size={18} />
      </span>
      <div className="col" style={{ gap: 4 }}>
        <span style={{ fontWeight: 800 }}>{t(status ?? 'refunded')}</span>
        <p className="small muted" style={{ margin: 0 }}>
          {note}
        </p>
      </div>
    </div>
  )
}

export function OrderHead({ o }: { o: OrderView }) {
  const cur = useCurrencyStore((s) => s.currency)
  return (
    <div className="row between">
      <div className="row" style={{ gap: 14 }}>
        <ImageArt
          art={o.art}
          word={o.product.split(' — ')[0].split(' ')[0]}
          h={54}
          wordSize={13}
          radius={12}
          style={{ width: 72 }}
        />
        <div className="col" style={{ gap: 2, alignItems: 'flex-start' }}>
          <span style={{ fontWeight: 800 }}>{o.product}</span>
          <span className="tiny faint num">
            {o.id} · {o.method}
          </span>
          <div style={{ marginTop: 4 }}>
            <OrderStatusBadge status={o.status} />
          </div>
        </div>
      </div>
      <Price usd={o.total} cur={cur} className="h3 num" />
    </div>
  )
}

export function CreditConfirm({ o }: { o: OrderView }) {
  const { t } = useTranslation()
  return (
    <div
      className="panel card-pad"
      style={{
        background: 'var(--grad-soft)',
        border: '1px solid var(--border-strong)',
        display: 'flex',
        flexDirection: 'column',
        gap: 12,
      }}
    >
      <div className="row between">
        <span className="row" style={{ gap: 8, fontWeight: 800 }}>
          <Icon name="user" size={16} />
          {t('credited_to')}
        </span>
        <OrderStatusBadge status={o.status} />
      </div>
      <div className="row between">
        <span className="muted small">{t('credited_to')}</span>
        <span className="num" style={{ fontWeight: 800, fontSize: 17 }}>
          {o.account}
        </span>
      </div>
      <div className="row between">
        <span className="muted small">{t('amount_credited')}</span>
        <span style={{ fontWeight: 800 }}>{o.amount}</span>
      </div>
      <div className="row between">
        <span className="muted small">{t('credited_at')}</span>
        <span className="num small">{o.ts}</span>
      </div>
      <p className="tiny faint">{t('no_code_note')}</p>
    </div>
  )
}

export function TransferTimeline({ o }: { o: OrderView }) {
  const { t } = useTranslation()
  const rank =
    ({ submitted: 1, processing: 2, completed: 3 } as Record<string, number>)[o.status ?? ''] || 1
  const steps = o.steps ?? [{ k: 'submitted', ts: '' }, { k: 'processing', ts: '' }, { k: 'completed', ts: '' }]
  return (
    <div className="col" style={{ gap: 0, paddingInlineStart: 4 }}>
      {steps.map((s, i) => {
        const reached = i + 1 <= rank
        const current = i + 1 === rank && rank < 3
        return (
          <div key={s.k} className="row" style={{ gap: 14, alignItems: 'flex-start' }}>
            <div className="col center" style={{ width: 26 }}>
              <span
                style={{
                  width: 22,
                  height: 22,
                  borderRadius: 99,
                  display: 'grid',
                  placeItems: 'center',
                  flex: 'none',
                  background: reached ? 'var(--ok)' : 'var(--surface-2)',
                  color: '#04121a',
                  boxShadow: current ? '0 0 0 5px rgba(47,212,122,.18)' : 'none',
                  border: reached ? 0 : '1.5px solid var(--border-strong)',
                }}
              >
                {reached && <Icon name="check" size={13} stroke={3} />}
              </span>
              {i < steps.length - 1 && (
                <span
                  style={{
                    width: 2,
                    height: 30,
                    background: i + 1 < rank ? 'var(--ok)' : 'var(--border)',
                  }}
                />
              )}
            </div>
            <div className="col" style={{ gap: 1, paddingBottom: 16 }}>
              <span style={{ fontWeight: 700, color: reached ? 'var(--text)' : 'var(--text-faint)' }}>
                {t('st_' + s.k)}
              </span>
              {s.ts && <span className="tiny faint num">{s.ts}</span>}
            </div>
          </div>
        )
      })}
    </div>
  )
}

export function TransferDetail({ o }: { o: OrderView }) {
  const { t } = useTranslation()
  const toast = useToast()
  const r = o.recipient ?? { name: '', country: '', detail: '' }
  return (
    <div className="col" style={{ gap: 16 }}>
      <div className="vault" style={{ fontSize: 16 }}>
        <div className="col" style={{ gap: 2 }}>
          <span className="tiny muted" style={{ letterSpacing: 0 }}>
            {t('reference')}
          </span>
          <span className="code" style={{ filter: 'none', fontWeight: 700 }}>
            {o.ref}
          </span>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            navigator.clipboard?.writeText(o.ref ?? '').catch(() => {})
            toast(t('copied'), 'copy')
          }}
        >
          <Icon name="copy" size={15} />
          {t('copy')}
        </Button>
      </div>
      <div
        className="panel card-pad"
        style={{ display: 'flex', flexDirection: 'column', gap: 8 }}
      >
        <span className="label" style={{ margin: 0 }}>
          {t('recipient')}
        </span>
        <div className="row between">
          <span className="muted small">{t('recipient_name')}</span>
          <span style={{ fontWeight: 700 }}>{r.name}</span>
        </div>
        <div className="row between">
          <span className="muted small">{t('country')}</span>
          <span style={{ fontWeight: 700 }}>{r.country}</span>
        </div>
        {r.detail && (
          <div className="row between">
            <span className="muted small">{t('method')}</span>
            <span className="num" style={{ fontWeight: 700 }}>
              {r.detail}
            </span>
          </div>
        )}
      </div>
      <TransferTimeline o={o} />
      <p className="tiny faint">{t('transfer_note')}</p>
    </div>
  )
}
