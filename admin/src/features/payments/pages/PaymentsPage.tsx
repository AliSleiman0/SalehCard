import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Icon,
  PageHead,
  Chip,
  Modal,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { money } from '@/lib/utils'
import { ApiError } from '@/lib/api-client'
import { useCan } from '@/stores/auth'
import {
  usePayments,
  useDeposits,
  useAttributeDeposit,
  useIgnoreDeposit,
} from '../hooks/usePayments'
import {
  tronscanAddress,
  tronscanTx,
  type PaymentIntentStatus,
  type PaymentPurpose,
  type DepositStatus,
  type UnmatchedDeposit,
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

const DEPOSIT_FILTERS: [DepositStatus | '', string][] = [
  ['unmatched', 'Unmatched'],
  ['credited', 'Credited'],
  ['ignored', 'Ignored'],
  ['', 'All'],
]

const DEPOSIT_STATUS_CLASS: Record<DepositStatus, string> = {
  unmatched: 'st st-warn',
  credited: 'st st-ok',
  ignored: 'st',
}

/** A shortened hash/address for compact display (0x1234…abcd style). */
const short = (s: string) => (s.length <= 14 ? s : `${s.slice(0, 8)}…${s.slice(-6)}`)

/**
 * Full-precision USDT amount: shared-mode intents carry sub-cent "salt"
 * digits that identify the payer, so trimming to 2 decimals would hide the
 * part that matters. Up to 6 decimals, trailing zeros trimmed, min 2.
 */
const usdtAmount = (v: number) => {
  let s = v.toFixed(6)
  const dot = s.indexOf('.')
  while (s.length - dot - 1 > 2 && s.endsWith('0')) s = s.slice(0, -1)
  return `$${s}`
}

export default function PaymentsPage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState<'intents' | 'deposits'>('intents')

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_finance'), t('nav_payments')]}
        title={t('nav_payments')}
      />

      <div className="chiprow" style={{ marginBottom: 14 }}>
        <Chip on={tab === 'intents'} onClick={() => setTab('intents')}>
          {t('payments_tab_intents')}
        </Chip>
        <Chip on={tab === 'deposits'} onClick={() => setTab('deposits')}>
          {t('payments_tab_deposits')}
        </Chip>
      </div>

      {tab === 'intents' ? <IntentsView /> : <DepositsView />}
    </div>
  )
}

function IntentsView() {
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
        <button className="abtn" style={{ marginInlineStart: 'auto' }} onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> Refresh
        </button>
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
                  <b style={{ fontSize: 15 }}>{usdtAmount(r.amountUsd)}</b>
                  <span className="bdg">{r.network.toUpperCase()}</span>
                  <span className="bdg">{r.addressMode === 'shared' ? 'Shared' : 'Derived'}</span>
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
                    <span style={{ color: 'var(--danger)' }}>received {usdtAmount(r.receivedUsd)}</span>
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
  )
}

function DepositsView() {
  const { t } = useTranslation()
  const can = useCan()
  const canManage = can('payments.manage')
  const [status, setStatus] = useState<DepositStatus | ''>('unmatched')
  const [page, setPage] = useState(1)
  const [toAttribute, setToAttribute] = useState<UnmatchedDeposit | null>(null)
  const [toIgnore, setToIgnore] = useState<UnmatchedDeposit | null>(null)

  const { data, isLoading, isError, refetch } = useDeposits({
    page,
    status: status || undefined,
  })
  const rows = useMemo(() => data?.data ?? [], [data])
  const meta = data?.meta

  return (
    <div className="acard">
      <div className="toolbar" style={{ gap: 16, flexWrap: 'wrap' }}>
        <div className="chiprow">
          {DEPOSIT_FILTERS.map(([k, l]) => (
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
        <button className="abtn" style={{ marginInlineStart: 'auto' }} onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> Refresh
        </button>
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message="Couldn't load deposits." onRetry={() => refetch()} />
      ) : rows.length === 0 ? (
        <EmptyState
          title={t('payments_deposits_empty')}
          sub={t('payments_deposits_empty_sub')}
        />
      ) : (
        <>
          <div>
            {rows.map((d) => (
              <div key={d.id} style={{ borderBottom: '1px solid var(--border)', padding: '14px 18px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                  <b style={{ fontSize: 15 }}>{usdtAmount(d.amountUsd)}</b>
                  <span className="bdg">{d.network.toUpperCase()}</span>
                  <span className="faint" style={{ fontSize: 12 }}>
                    {new Date(d.blockTime).toLocaleString()}
                  </span>
                  <span style={{ marginInlineStart: 'auto' }} className={DEPOSIT_STATUS_CLASS[d.status]}>
                    <i className="d" />
                    {d.status}
                  </span>
                </div>

                <div
                  style={{ display: 'flex', gap: 14, flexWrap: 'wrap', marginTop: 8, fontSize: 12.5, alignItems: 'center' }}
                  className="faint"
                >
                  <a href={tronscanTx(d.txHash)} target="_blank" rel="noreferrer" title={d.txHash}>
                    tx {short(d.txHash)} ↗
                  </a>
                  <span title={d.fromAddress}>from {short(d.fromAddress)}</span>
                  {d.attributedBy && <span>· by {d.attributedBy}</span>}
                  {d.note && <span>· {d.note}</span>}
                  {canManage && d.status === 'unmatched' && (
                    <span style={{ marginInlineStart: 'auto', display: 'flex', gap: 8 }}>
                      <button className="abtn primary sm" onClick={() => setToAttribute(d)}>
                        {t('payments_attribute')}
                      </button>
                      <button className="abtn sm" onClick={() => setToIgnore(d)}>
                        {t('payments_ignore')}
                      </button>
                    </span>
                  )}
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
            label={t('payments_tab_deposits')}
            onPage={setPage}
          />
        </>
      )}

      {toAttribute && <AttributeModal dep={toAttribute} onClose={() => setToAttribute(null)} />}
      {toIgnore && <IgnoreModal dep={toIgnore} onClose={() => setToIgnore(null)} />}
    </div>
  )
}

/** Credit an unmatched deposit to a customer found by email or user id. */
function AttributeModal({ dep, onClose }: { dep: UnmatchedDeposit; onClose: () => void }) {
  const { t } = useTranslation()
  const [user, setUser] = useState('')
  const [note, setNote] = useState('')
  const [error, setError] = useState('')
  const attributeM = useAttributeDeposit()

  const apply = () => {
    if (!user.trim()) {
      setError('A customer email or user id is required.')
      return
    }
    setError('')
    attributeM.mutate(
      { id: dep.id, user: user.trim(), note: note.trim() || undefined },
      {
        onSuccess: onClose,
        onError: (e) => setError(e instanceof ApiError ? e.message : 'Attribution failed.'),
      },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={460}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>{t('payments_attribute_title')}</h3>
        <p style={{ fontSize: 13.5, color: 'var(--text-dim)', marginBottom: 16 }}>
          Credit the <b>{money(dep.amountUsd)}</b> deposit (tx {short(dep.txHash)}) to a customer's
          wallet. This cannot be undone.
        </p>
        <label className="alabel">Customer email or user id</label>
        <input
          className="afield"
          placeholder="customer@example.com"
          value={user}
          onChange={(e) => setUser(e.target.value)}
        />
        <label className="alabel" style={{ marginTop: 12 }}>
          Note (optional)
        </label>
        <input
          className="afield"
          placeholder="e.g. customer sent the base amount without the cents…"
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 18, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={attributeM.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn primary" onClick={apply} disabled={attributeM.isPending}>
            <Icon name="check" size={15} /> {t('payments_attribute')}
          </button>
        </div>
      </div>
    </Modal>
  )
}

/** Ignore an unmatched deposit (dust/spam/unknown sender). */
function IgnoreModal({ dep, onClose }: { dep: UnmatchedDeposit; onClose: () => void }) {
  const { t } = useTranslation()
  const [note, setNote] = useState('')
  const [error, setError] = useState('')
  const ignoreM = useIgnoreDeposit()

  const apply = () => {
    setError('')
    ignoreM.mutate(
      { id: dep.id, note: note.trim() || undefined },
      {
        onSuccess: onClose,
        onError: (e) => setError(e instanceof ApiError ? e.message : 'Could not ignore deposit.'),
      },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={440}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>{t('payments_ignore_title')}</h3>
        <p style={{ fontSize: 13.5, color: 'var(--text-dim)', marginBottom: 16 }}>
          Mark the <b>{money(dep.amountUsd)}</b> deposit (tx {short(dep.txHash)}) as ignored — no
          customer will be credited.
        </p>
        <label className="alabel">Note (optional)</label>
        <input
          className="afield"
          placeholder="e.g. dust / spam transfer…"
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 18, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={ignoreM.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn danger" onClick={apply} disabled={ignoreM.isPending}>
            <Icon name="x" size={15} /> {t('payments_ignore')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
