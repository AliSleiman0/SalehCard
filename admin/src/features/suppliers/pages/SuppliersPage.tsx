import { useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { filterCatalog } from '../lib/filterCatalog'
import { syncSummary } from '../lib/syncSummary'
import { alertText } from '../lib/alertText'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { PageHead, Icon, Modal, Pagination, LoadingSpinner, EmptyState, ErrorState, Tabs } from '@/components'
import { useCan } from '@/stores/auth'
import { toast } from '@/stores/toast'
import {
  useSuppliersList,
  useSupplierCatalog,
  useSupplierOrders,
  useSyncSupplier,
  useImportProducts,
  useUpdateSupplierSettings,
  useSyncStatus,
  useSupplierAlerts,
  useAckSupplierAlert,
  useAckAllSupplierAlerts,
} from '../hooks/useSuppliers'
import type {
  SupplierView,
  SupplierHealth,
  CatalogProductView,
  SupplierAlert,
  SupplierAlertKind,
} from '../api/suppliers'
import { orderNumber } from '@/lib/orderNumber'
import { relativeTime, relativeTimeInline } from '@/lib/utils'

const HEALTH_CLASS: Record<SupplierHealth, string> = {
  ok: 'st-ok',
  low_balance: 'st-warn',
  auth_error: 'st-danger',
  ip_blocked: 'st-danger',
  unreachable: 'st-danger',
  maintenance: 'st-danger',
  not_probed: 'st-mute',
}

export default function SuppliersPage() {
  const { t } = useTranslation()
  const can = useCan()
  const canManage = can('suppliers.manage')

  const { data, isLoading, isError, refetch } = useSuppliersList()
  const suppliers = useMemo(() => data?.data?.suppliers ?? [], [data])

  const [browse, setBrowse] = useState<SupplierView | null>(null)
  const [orders, setOrders] = useState<SupplierView | null>(null)
  const [settings, setSettings] = useState<SupplierView | null>(null)
  const [params, setParams] = useSearchParams()
  const tab = params.get('tab') === 'alerts' ? 'alerts' : 'suppliers'
  const setTab = (k: 'suppliers' | 'alerts') => {
    const next = new URLSearchParams(params)
    if (k === 'alerts') next.set('tab', 'alerts')
    else next.delete('tab')
    setParams(next, { replace: true })
  }
  const openAlerts = useSupplierAlerts('open')
  const openCount = openAlerts.data?.data?.open ?? 0

  const sync = useSyncSupplier()
  const runSync = (s: SupplierView) => {
    sync.mutate(s.id, {
      onSuccess: (res) => {
        if (res.data) toast.success(`${s.name}: ${syncSummary(res.data, t)}`)
      },
    })
  }

  return (
    <div className="page page-wide">
      <PageHead
        back="/"
        crumbs={[t('grp_operations'), t('nav_suppliers')]}
        title={t('nav_suppliers')}
        sub={t('sup_sub')}
      />

      <SyncStatusStrip />

      <Tabs
        items={[
          { k: 'suppliers' as const, label: t('sup_tab_suppliers') },
          { k: 'alerts' as const, label: openCount > 0 ? t('sup_tab_alerts_n', { n: openCount }) : t('sup_tab_alerts') },
        ]}
        value={tab}
        onChange={setTab}
      />
      <div style={{ height: 14 }} />

      {tab === 'alerts' ? (
        <AlertsPanel canManage={canManage} suppliers={suppliers} />
      ) : isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message={t('sup_load_error')} onRetry={() => refetch()} />
      ) : suppliers.length === 0 ? (
        <EmptyState title={t('sup_none_title')} sub={t('sup_none_sub')} />
      ) : (
        <div className="acard-grid">
          {suppliers.map((s) => (
            <div className="acard pad" key={s.id} style={{ marginBottom: 16 }}>
              <div className="panelhead">
                <Icon name={s.kind === 'telecom' ? 'globe' : 'box'} size={16} />
                <h3>{s.name}</h3>
                <span className="bdg">{t(`sup_kind_${s.kind}`, { defaultValue: s.kind })}</span>
                <span className={`st ${HEALTH_CLASS[s.health] ?? 'st-mute'}`} style={{ marginLeft: 'auto' }}>
                  <i className="d" /> {t(`sup_health_${s.health}`, { defaultValue: s.health })}
                </span>
              </div>

              <div style={{ display: 'flex', gap: 24, flexWrap: 'wrap', margin: '10px 0 14px' }}>
                <div>
                  <div style={{ fontSize: 24, fontWeight: 800 }}>{s.balanceText}</div>
                  <div style={{ fontSize: 11, color: 'var(--text-faint)', fontWeight: 700 }}>
                    {t('sup_balance', { currency: s.currency })}
                  </div>
                </div>
                <div>
                  <div style={{ fontSize: 24, fontWeight: 800 }}>{s.mappedProducts}</div>
                  <div style={{ fontSize: 11, color: 'var(--text-faint)', fontWeight: 700 }}>{t('sup_mapped')}</div>
                </div>
                <div>
                  <div style={{ fontSize: 24, fontWeight: 800 }}>{s.markupPercent}%</div>
                  <div style={{ fontSize: 11, color: 'var(--text-faint)', fontWeight: 700 }}>{t('sup_markup')}</div>
                </div>
                <div>
                  <div style={{ fontSize: 24, fontWeight: 800 }}>
                    {s.lowBalanceThreshold > 0 ? s.lowBalanceThreshold.toLocaleString() : '—'}
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--text-faint)', fontWeight: 700 }}>
                    {t('sup_low_balance_alert')}
                  </div>
                </div>
              </div>

              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                {canManage && (
                  <button className="abtn sm primary" onClick={() => setBrowse(s)}>
                    <Icon name="download" size={14} /> {t('sup_browse')}
                  </button>
                )}
                {canManage && (
                  <button
                    className="abtn sm"
                    disabled={sync.isPending}
                    title={t('sup_check_now_tip')}
                    onClick={() => runSync(s)}
                  >
                    <Icon name="refresh" size={14} /> {t('sup_check_now')}
                  </button>
                )}
                <button className="abtn sm" onClick={() => setOrders(s)}>
                  <Icon name="bag" size={14} /> {t('sup_recent_orders')}
                </button>
                {canManage && (
                  <button className="abtn sm ghost" onClick={() => setSettings(s)}>
                    <Icon name="settings" size={14} /> {t('sup_settings')}
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {browse && <BrowseModal supplier={browse} onClose={() => setBrowse(null)} />}
      {orders && <OrdersModal supplier={orders} onClose={() => setOrders(null)} />}
      {settings && <SettingsModal supplier={settings} onClose={() => setSettings(null)} />}
    </div>
  )
}

// --- Scheduled sync status ---

function SyncStatusStrip() {
  const { t } = useTranslation()
  const { data } = useSyncStatus()
  const st = data?.data
  if (!st) return null
  const mins = Math.round(st.intervalSeconds / 60)
  const last = st.last
  return (
    <div className="acard pad" style={{ marginBottom: 14, display: 'flex', gap: 10, alignItems: 'center', flexWrap: 'wrap', fontSize: 13 }}>
      <Icon name="refresh" size={15} />
      {st.enabled ? (
        st.dryRun ? (
          <span>
            <b>{t('sup_auto_on')}</b> {t('sup_every', { n: mins })} —{' '}
            <span className="st st-warn">{t('sup_warn_badge')}</span> {t('sup_warn_body')}
          </span>
        ) : (
          <span>
            <b>{t('sup_auto_on')}</b> {t('sup_every', { n: mins })}. {t('sup_live_body')}
          </span>
        )
      ) : (
        <span>
          <b>{t('sup_auto_off')}</b> {t('sup_off_body')}
          {st.dryRun ? ` ${t('sup_warnings_only_paren')}` : ''}
        </span>
      )}
      {last && (
        <span className="faint">
          {t('sup_last_check', { when: relativeTimeInline(last.finishedAt), summary: syncSummary(last, t) })}
        </span>
      )}
      {st.running && <span className="st st-mute">{t('sup_running')}</span>}
    </div>
  )
}

// --- Alerts ---

const ALERT_CLASS: Record<SupplierAlertKind, string> = {
  cost_changed: 'st-warn',
  loss_paused: 'st-danger',
  loss_recovered: 'st-ok',
  product_hidden: 'st-danger',
  product_shown: 'st-ok',
  fx_missing: 'st-warn',
  fallback_exhausted: 'st-danger',
}

// Warning-only mode: nothing was done, so these name what WOULD happen.
const ALERT_WOULD: Partial<Record<SupplierAlertKind, true>> = {
  product_hidden: true,
  product_shown: true,
  loss_recovered: true,
}

function alertLabel(a: SupplierAlert, t: TFunction): { label: string; cls: string } {
  const key = a.dryRun && ALERT_WOULD[a.kind] ? `sup_alert_would_${a.kind}` : `sup_alert_${a.kind}`
  return { label: t(key, { defaultValue: a.kind }), cls: ALERT_CLASS[a.kind] ?? 'st-mute' }
}

function AlertsPanel({ canManage, suppliers }: { canManage: boolean; suppliers: SupplierView[] }) {
  const { t, i18n } = useTranslation()
  const [show, setShow] = useState<'open' | 'all'>('open')
  const { data, isLoading, isError, refetch } = useSupplierAlerts(show)
  const alerts = data?.data?.alerts ?? []
  const ack = useAckSupplierAlert()
  const ackAll = useAckAllSupplierAlerts()
  const provName = (id?: number) => (id ? suppliers.find((s) => s.id === id)?.name ?? `#${id}` : '')

  return (
    <div className="acard pad">
      <div className="panelhead">
        <Icon name="bell" size={16} />
        <h3>{t('sup_alerts_title')}</h3>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: 8 }}>
          <button className={`abtn sm ${show === 'open' ? 'primary' : 'ghost'}`} onClick={() => setShow('open')}>
            {t('sup_show_open')}
          </button>
          <button className={`abtn sm ${show === 'all' ? 'primary' : 'ghost'}`} onClick={() => setShow('all')}>
            {t('sup_show_all')}
          </button>
          {canManage && show === 'open' && alerts.length > 1 && (
            <button className="abtn sm" disabled={ackAll.isPending} onClick={() => ackAll.mutate()}>
              {t('sup_ack_all')}
            </button>
          )}
        </div>
      </div>
      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message={t('sup_alerts_error')} onRetry={() => refetch()} />
      ) : alerts.length === 0 ? (
        <EmptyState
          title={show === 'open' ? t('sup_alerts_empty_open') : t('sup_alerts_empty_all')}
          sub={t('sup_alerts_empty_sub')}
        />
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table className="tbl" aria-label={t('sup_alerts_title')}>
            <thead>
              <tr>
                <th>{t('sup_col_kind')}</th>
                <th>{t('sup_col_what')}</th>
                <th>{t('sup_col_last_seen')}</th>
                <th style={{ textAlign: 'right' }} />
              </tr>
            </thead>
            <tbody>
              {alerts.map((a) => {
                const k = alertLabel(a, t)
                return (
                  <tr key={a.id} style={{ opacity: a.open ? 1 : 0.6 }}>
                    <td style={{ whiteSpace: 'nowrap' }}>
                      <span className={`st ${k.cls}`}>
                        <i className="d" /> {k.label}
                      </span>
                      {a.dryRun && (
                        <div>
                          <span className="bdg" style={{ marginTop: 4 }} title={t('sup_warning_only_tip')}>
                            {t('sup_warning_only')}
                          </span>
                        </div>
                      )}
                    </td>
                    <td style={{ fontSize: 13 }}>
                      {alertText(a, t, i18n.language)}
                      <div className="faint" style={{ fontSize: 12, marginTop: 2 }}>
                        {a.productId && <Link to={`/products/${a.productId}/edit`}>{t('sup_open_product')}</Link>}
                        {a.orderId && (
                          <Link to={`/orders/${a.orderId}`}>{t('sup_open_order', { n: orderNumber(a.orderId) })}</Link>
                        )}
                        {a.provider && !a.productId && !a.orderId ? provName(a.provider) : ''}
                        {a.count > 1 ? ` · ${t('sup_seen_n', { n: a.count })}` : ''}
                        {!a.open && a.ackBy ? ` · ${t('sup_closed_by', { who: a.ackBy })}` : ''}
                      </div>
                    </td>
                    <td className="muted" style={{ whiteSpace: 'nowrap' }}>
                      {relativeTime(a.lastAt)}
                    </td>
                    <td style={{ textAlign: 'right' }}>
                      {canManage && a.open && (
                        <button className="abtn xs" disabled={ack.isPending} onClick={() => ack.mutate(a.id)}>
                          {t('sup_ack')}
                        </button>
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

// --- Browse & import ---

function BrowseModal({ supplier, onClose }: { supplier: SupplierView; onClose: () => void }) {
  const { t } = useTranslation()
  const { data, isLoading, isError, refetch } = useSupplierCatalog(supplier.id)
  const products = useMemo<CatalogProductView[]>(() => data?.data?.products ?? [], [data])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [markup, setMarkup] = useState<string>(String(supplier.markupPercent || ''))
  const [search, setSearch] = useState('')
  const shown = useMemo(() => filterCatalog(products, search), [products, search])
  const importer = useImportProducts()

  const toggle = (upstreamId: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(upstreamId)) next.delete(upstreamId)
      else next.add(upstreamId)
      return next
    })
  }

  const doImport = () => {
    const markupPercent = markup.trim() === '' ? null : Number(markup)
    const items = Array.from(selected).map((upstreamId) => ({ upstreamId, markupPercent }))
    importer.mutate(
      { id: supplier.id, items },
      {
        onSuccess: (res) => {
          const r = res.data
          if (r) {
            toast.success(
              t('sup_imported', { created: r.created }) +
                (r.skipped.length ? t('sup_imported_skipped', { n: r.skipped.length }) : ''),
            )
            setSelected(new Set())
          }
        },
      },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={880}>
      <div className="panelhead">
        <Icon name="download" size={16} />
        <h3>{t('sup_browse_title', { name: supplier.name })}</h3>
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message={t('sup_catalog_error')} onRetry={() => refetch()} />
      ) : products.length === 0 ? (
        <EmptyState title={t('sup_no_upstream')} />
      ) : (
        <>
          <div style={{ display: 'flex', gap: 10, alignItems: 'end', margin: '10px 0' }}>
            <div>
              <label className="alabel">{t('sup_markup_pct')}</label>
              <input
                className="afield"
                style={{ width: 120 }}
                type="number"
                value={markup}
                onChange={(e) => setMarkup(e.target.value)}
                placeholder={String(supplier.markupPercent)}
              />
            </div>
            <div style={{ flex: 1, minWidth: 180 }}>
              <label className="alabel">{t('sup_search')}</label>
              <input
                className="afield"
                type="search"
                aria-label={t('sup_search_aria')}
                placeholder={t('sup_search_ph')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            <button
              className="abtn"
              disabled={shown.every((p) => p.mapped || selected.has(p.upstreamId))}
              onClick={() =>
                setSelected((prev) => {
                  const next = new Set(prev)
                  for (const p of shown) if (!p.mapped) next.add(p.upstreamId)
                  return next
                })
              }
            >
              {t('sup_select_all_shown')}
            </button>
            <div style={{ fontSize: 12, color: 'var(--text-faint)', whiteSpace: 'nowrap' }}>
              {search.trim() ? t('sup_shown_of', { shown: shown.length, total: products.length }) : ''}
              {t('sup_selected_n', { n: selected.size })}
            </div>
            <button
              className="abtn primary"
              disabled={selected.size === 0 || importer.isPending}
              onClick={doImport}
            >
              <Icon name="plus" size={14} /> {t('sup_import_selected')}
            </button>
          </div>

          <div style={{ maxHeight: 420, overflow: 'auto' }}>
            <table className="tbl">
              <thead>
                <tr>
                  <th style={{ width: 34 }} />
                  <th>{t('sup_col_product')}</th>
                  <th>{t('sup_col_category')}</th>
                  <th className="num">{t('sup_col_base_price')}</th>
                  <th>{t('sup_col_type')}</th>
                  <th>{t('sup_col_status')}</th>
                </tr>
              </thead>
              <tbody>
                {shown.length === 0 && (
                  <tr>
                    <td colSpan={6} className="muted" style={{ padding: 16 }}>
                      {t('sup_no_match', { q: search.trim() })}
                    </td>
                  </tr>
                )}
                {shown.map((p) => (
                  <tr key={p.upstreamId}>
                    <td>
                      <input
                        type="checkbox"
                        disabled={p.mapped}
                        checked={selected.has(p.upstreamId)}
                        onChange={() => toggle(p.upstreamId)}
                      />
                    </td>
                    <td className="strong">{p.name}</td>
                    <td className="muted">{p.category}</td>
                    <td className="num mono">
                      {p.basePrice} {p.currency}
                    </td>
                    <td className="muted">{p.productType}</td>
                    <td>
                      {p.mapped ? (
                        <span className="st st-ok">
                          <i className="d" /> {t('sup_st_mapped')}
                        </span>
                      ) : p.available ? (
                        <span className="st st-mute">
                          <i className="d" /> {t('sup_st_available')}
                        </span>
                      ) : (
                        <span className="st st-warn">
                          <i className="d" /> {t('sup_st_unavailable')}
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </Modal>
  )
}

// --- Recent orders ---

const ORDERS_PAGE_SIZE = 20

function OrdersModal({ supplier, onClose }: { supplier: SupplierView; onClose: () => void }) {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const { data, isLoading, isError, refetch } = useSupplierOrders(supplier.id, {
    limit: ORDERS_PAGE_SIZE,
    offset: (page - 1) * ORDERS_PAGE_SIZE,
  })
  const orders = data?.data?.orders ?? []
  const meta = data?.meta

  return (
    <Modal onClose={onClose} maxWidth={860}>
      <div className="panelhead">
        <Icon name="bag" size={16} />
        <h3>{t('sup_orders_title', { name: supplier.name })}</h3>
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message={t('sup_orders_error')} onRetry={() => refetch()} />
      ) : orders.length === 0 ? (
        <EmptyState title={t('sup_orders_empty')} sub={t('sup_orders_empty_sub')} />
      ) : (
        <>
          <div style={{ maxHeight: 440, overflow: 'auto' }}>
            <table className="tbl">
              <thead>
                <tr>
                  <th>{t('sup_col_order')}</th>
                  <th>{t('sup_col_status')}</th>
                  <th className="num">{t('sup_col_total')}</th>
                  <th>{t('sup_col_upstream_ref')}</th>
                  <th>{t('sup_col_created')}</th>
                </tr>
              </thead>
              <tbody>
                {orders.map((o) => (
                  <tr key={o.id}>
                    <td className="mono">
                      <a href={`/orders/${o.id}`}>{orderNumber(o.id)}</a>
                    </td>
                    <td>
                      <span className={`st ${o.stuckAt ? 'st-danger' : o.status === 'completed' ? 'st-ok' : o.status === 'refunded' || o.status === 'failed' ? 'st-warn' : 'st-mute'}`}>
                        <i className="d" />{' '}
                        {o.stuckAt ? t('sup_order_st_stuck') : t(`sup_order_st_${o.status}`, { defaultValue: o.status })}
                      </span>
                    </td>
                    <td className="num mono">
                      {o.total} {o.currency}
                    </td>
                    <td className="mono muted">{o.upstreamRef || '—'}</td>
                    <td className="muted">{new Date(o.createdAt).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Pagination
            page={page}
            pages={meta ? meta.pages : 1}
            total={meta ? meta.total : orders.length}
            shown={orders.length}
            limit={ORDERS_PAGE_SIZE}
            onPage={setPage}
          />
        </>
      )}
    </Modal>
  )
}

// --- Settings ---

function SettingsModal({ supplier, onClose }: { supplier: SupplierView; onClose: () => void }) {
  const { t } = useTranslation()
  const [threshold, setThreshold] = useState(String(supplier.lowBalanceThreshold || ''))
  const [markup, setMarkup] = useState(String(supplier.markupPercent || ''))
  const [fx, setFx] = useState(supplier.fxToUsd ? String(supplier.fxToUsd) : '')
  const save = useUpdateSupplierSettings()
  const usd = supplier.currency === 'USD'

  const submit = () => {
    save.mutate(
      {
        id: supplier.id,
        patch: {
          lowBalanceThreshold: threshold.trim() === '' ? 0 : Number(threshold),
          markupPercent: markup.trim() === '' ? 0 : Number(markup),
          ...(usd ? {} : { fxToUsd: fx.trim() === '' ? 0 : Number(fx) }),
        },
      },
      { onSuccess: () => onClose() },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={420}>
      <div className="panelhead">
        <Icon name="settings" size={16} />
        <h3>{t('sup_settings_title', { name: supplier.name })}</h3>
      </div>
      <div style={{ display: 'grid', gap: 12, margin: '12px 0' }}>
        <div>
          <label className="alabel">{t('sup_threshold_label', { currency: supplier.currency })}</label>
          <input
            className="afield"
            type="number"
            value={threshold}
            onChange={(e) => setThreshold(e.target.value)}
            placeholder={t('sup_threshold_ph')}
          />
        </div>
        <div>
          <label className="alabel">{t('sup_default_markup_label')}</label>
          <input
            className="afield"
            type="number"
            value={markup}
            onChange={(e) => setMarkup(e.target.value)}
            placeholder="0"
          />
        </div>
        {!usd && (
          <div>
            <label className="alabel">{t('sup_fx_label', { currency: supplier.currency })}</label>
            <input
              className="afield"
              type="number"
              step="any"
              value={fx}
              onChange={(e) => setFx(e.target.value)}
              placeholder={t('sup_fx_ph')}
            />
            <div className="ahint" style={{ margin: '4px 0 0' }}>
              {t('sup_fx_hint', { name: supplier.name })}
            </div>
          </div>
        )}
      </div>
      <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
        <button className="abtn" onClick={onClose}>
          {t('cancel')}
        </button>
        <button className="abtn primary" disabled={save.isPending} onClick={submit}>
          {t('sup_save')}
        </button>
      </div>
    </Modal>
  )
}
