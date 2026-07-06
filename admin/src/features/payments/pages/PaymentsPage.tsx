import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Icon,
  PageHead,
  Chip,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { money } from '@/lib/utils'
import { usePayments } from '../hooks/usePayments'
import {
  tronscanAddress,
  tronscanTx,
  type PaymentIntentStatus,
  type PaymentPurpose,
} from '../api/payments'

const STATUS_FILTERS: [PaymentIntentStatus | '', string][] = [
  ['', 'All'],
  ['pending', 'Pending'],
  ['confirming', 'Confirming'],
  ['confirmed', 'Confirmed'],
  ['expired', 'Expired'],
]

const PURPOSE_FILTERS: [PaymentPurpose | '', string][] = [
  ['', 'All types'],
  ['topup', 'Top-ups'],
  ['order', 'Orders'],
]

const STATUS_CLASS: Record<PaymentIntentStatus, string> = {
  pending: 'st st-warn',
  confirming: 'st st-warn',
  confirmed: 'st st-ok',
  expired: 'st st-danger',
}

/** A shortened hash/address for compact display (0x1234…abcd style). */
const short = (s: string) => (s.length <= 14 ? s : `${s.slice(0, 8)}…${s.slice(-6)}`)

export default function PaymentsPage() {
  const { t } = useTranslation()
  const [status, setStatus] = useState<PaymentIntentStatus | ''>('')
  const [purpose, setPurpose] = useState<PaymentPurpose | ''>('')
  const [page, setPage] = useState(1)

  const { data, isLoading, isError, refetch } = usePayments({
    page,
    status: status || undefined,
    purpose: purpose || undefined,
  })
  const rows = useMemo(() => data?.data ?? [], [data])
  const meta = data?.meta

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_finance'), t('nav_payments')]}
        title={t('nav_payments')}
        sub={meta ? `${meta.total.toLocaleString()} payments` : ''}
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> Refresh
        </button>
      </PageHead>

      <div className="acard">
        <div className="toolbar" style={{ gap: 16, flexWrap: 'wrap' }}>
          <div className="chiprow">
            {STATUS_FILTERS.map(([k, l]) => (
              <Chip
                key={k || 'all'}
                on={status === k}
                onClick={() => {
                  setStatus(k)
                  setPage(1)
                }}
              >
                {l}
              </Chip>
            ))}
          </div>
          <div className="chiprow">
            {PURPOSE_FILTERS.map(([k, l]) => (
              <Chip
                key={k || 'alltypes'}
                on={purpose === k}
                onClick={() => {
                  setPurpose(k)
                  setPage(1)
                }}
              >
                {l}
              </Chip>
            ))}
          </div>
        </div>

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message="Couldn't load payments." onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState
            title="No USDT payments"
            sub="On-chain USDT deposits (wallet top-ups and order payments) appear here as customers pay."
          />
        ) : (
          <>
            <div>
              {rows.map((r) => (
                <div key={r.id} style={{ borderBottom: '1px solid var(--border)', padding: '14px 18px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                    <b style={{ fontSize: 15 }}>{money(r.amountUsd)}</b>
                    <span className="bdg">{r.network.toUpperCase()}</span>
                    <span className="bdg">{r.purpose === 'order' ? 'Order' : 'Top-up'}</span>
                    <span className="faint" style={{ fontSize: 12.5 }}>
                      {r.user.email || r.user.phone || r.userId.slice(-8)}
                    </span>
                    <span className="faint" style={{ fontSize: 12 }}>
                      · {new Date(r.createdAt).toLocaleString()}
                    </span>
                    <span style={{ marginInlineStart: 'auto' }} className={STATUS_CLASS[r.status]}>
                      <i className="d" />
                      {r.status}
                    </span>
                  </div>

                  <div
                    style={{ display: 'flex', gap: 14, flexWrap: 'wrap', marginTop: 8, fontSize: 12.5 }}
                    className="faint"
                  >
                    <a href={tronscanAddress(r.address)} target="_blank" rel="noreferrer" title={r.address}>
                      {short(r.address)} ↗
                    </a>
                    {r.txHash && (
                      <a href={tronscanTx(r.txHash)} target="_blank" rel="noreferrer" title={r.txHash}>
                        tx {short(r.txHash)} ↗
                      </a>
                    )}
                    {r.orderId && <span>order …{r.orderId.slice(-6)}</span>}
                    {r.receivedUsd > 0 && r.receivedUsd !== r.amountUsd && (
                      <span style={{ color: 'var(--danger)' }}>received {money(r.receivedUsd)}</span>
                    )}
                    {r.settlement && <span>· {r.settlement}</span>}
                  </div>
                </div>
              ))}
            </div>
            <Pagination
              page={meta?.page ?? 1}
              pages={meta?.pages ?? 1}
              total={meta?.total ?? rows.length}
              shown={rows.length}
              limit={meta?.limit}
              label={t('nav_payments')}
              onPage={setPage}
            />
          </>
        )}
      </div>
    </div>
  )
}
