import { useMemo, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Icon, Panel, LoadingSpinner, ErrorState, EmptyState } from '@/components'
import { AcctSidebar } from '@/features/auth/components/AcctSidebar'
import { OrderRow } from '@/features/auth/components/OrderRow'
import { useLocaleStore } from '@/stores/locale'
import { useOrders } from '../hooks/useOrders'
import { adaptOrder } from '../lib/adaptOrder'
import { matchesOrder, normalizeOrderQuery } from '../lib/matchOrder'

export default function OrdersPage() {
  const { t } = useTranslation()
  const locale = useLocaleStore((s) => s.locale)
  const query = useOrders(true)
  // The search lives in ?q= so a refresh, a shared link or Back from an order
  // keeps it. The API returns the customer's whole history, so this filters on
  // the client (see matchOrder.ts for what is searched).
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''
  const searching = normalizeOrderQuery(q) !== ''
  const input = useRef<HTMLInputElement>(null)
  const all = query.data ?? []
  const orders = useMemo(
    () => (query.data ?? []).filter((o) => matchesOrder(o, q)).map((o) => adaptOrder(o, locale)),
    [query.data, locale, q],
  )
  const setQ = (v: string) => setParams(v ? { q: v } : {}, { replace: true })
  const clear = () => {
    setQ('')
    input.current?.focus()
  }

  return (
    <div className="wrap" style={{ padding: '26px 0 50px' }}>
      <h1 className="h1" style={{ marginBottom: 24 }}>
        {t('order_history')}
      </h1>
      <div className="cols-acct">
        <AcctSidebar active="orders" />
        <Panel>
          <div className="row" style={{ gap: 8, marginBottom: 6 }}>
            <span className="badge badge-secure">
              <Icon name="shield" size={12} />
              {t('secure_vault')}
            </span>
            <span className="small muted">{t('redeem_note')}</span>
          </div>
          {query.isLoading ? (
            <LoadingSpinner />
          ) : query.isError ? (
            <ErrorState
              title={t('no_results')}
              sub={t('no_results_sub')}
              onRetry={() => query.refetch()}
              retryLabel={t('retry')}
            />
          ) : all.length === 0 ? (
            <p className="muted" style={{ padding: '20px 4px' }}>
              {t('no_orders')}
            </p>
          ) : (
            <>
              <div className="searchbar order-search" role="search">
                <Icon name="search" size={18} />
                <input
                  ref={input}
                  type="search"
                  value={q}
                  onChange={(e) => setQ(e.target.value)}
                  onKeyDown={(e) => e.key === 'Escape' && q && clear()}
                  placeholder={t('order_search_ph')}
                  aria-label={t('order_search_label')}
                  enterKeyHint="search"
                  autoComplete="off"
                  spellCheck={false}
                />
                {q && (
                  <button type="button" className="searchbar-clear" onClick={clear} aria-label={t('order_search_clear')}>
                    <Icon name="close" size={14} />
                  </button>
                )}
              </div>
              {searching && orders.length === 0 ? (
                <EmptyState
                  icon="search"
                  title={t('order_search_none', { q: q.trim() })}
                  sub={t('order_search_none_sub')}
                  cta={t('order_search_clear')}
                  onCta={clear}
                />
              ) : (
                <>
                  {searching && (
                    <p className="small muted" aria-live="polite" style={{ margin: '0 4px 4px' }}>
                      {t('order_search_count', { count: orders.length, total: all.length })}
                    </p>
                  )}
                  {orders.map((o) => (
                    <OrderRow key={o.id} o={o} />
                  ))}
                </>
              )}
            </>
          )}
        </Panel>
      </div>
    </div>
  )
}
