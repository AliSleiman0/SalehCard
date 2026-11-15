import { useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { filterCatalog } from '../lib/filterCatalog'
import { syncSummary } from '../lib/syncSummary'
import { useTranslation } from 'react-i18next'
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
const HEALTH_LABEL: Record<SupplierHealth, string> = {
  ok: 'OK',
  low_balance: 'Low balance',
  auth_error: 'Auth error',
  ip_blocked: 'IP blocked',
  unreachable: 'Unreachable',
  maintenance: 'Maintenance',
  not_probed: 'Not probed',
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
  const setTab = (t: 'suppliers' | 'alerts') => {
    const next = new URLSearchParams(params)
    if (t === 'alerts') next.set('tab', 'alerts')
    else next.delete('tab')
    setParams(next, { replace: true })
  }
  const openAlerts = useSupplierAlerts('open')
  const openCount = openAlerts.data?.data?.open ?? 0

  const sync = useSyncSupplier()
  const runSync = (s: SupplierView) => {
    sync.mutate(s.id, {
      onSuccess: (res) => {
        if (res.data) toast.success(`${s.name}: ${syncSummary(res.data)}`)
      },
    })
  }

  return (
    <div className="page page-wide">
      <PageHead
        back="/"
        crumbs={[t('grp_operations'), t('nav_suppliers')]}
        title={t('nav_suppliers')}
        sub="Upstream fulfillment suppliers — balance, health, mapped products, and catalog import."
      />

      <SyncStatusStrip />

      <Tabs
        items={[
          { k: 'suppliers' as const, label: 'Suppliers' },
          { k: 'alerts' as const, label: openCount > 0 ? `Alerts (${openCount})` : 'Alerts' },
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
        <ErrorState message="Could not load suppliers." onRetry={() => refetch()} />
      ) : suppliers.length === 0 ? (
        <EmptyState
          title="No suppliers configured"
          sub="Set a supplier credential (SUPPLIER_*_TOKEN / SUPPLIER_UMANAGE_KEY) to enable a supplier."
        />
      ) : (
        <div className="acard-grid">
          {suppliers.map((s) => (
            <div className="acard pad" key={s.id} style={{ marginBottom: 16 }}>
              <div className="panelhead">
                <Icon name={s.kind === 'telecom' ? 'globe' : 'box'} size={16} />
                <h3>{s.name}</h3>
                <span className="bdg">{s.kind}</span>
                <span className={`st ${HEALTH_CLASS[s.health] ?? 'st-mute'}`} style={{ marginLeft: 'auto' }}>
                  <i className="d" /> {HEALTH_LABEL[s.health] ?? s.health}
                </span>
              </div>

              <div style={{ display: 'flex', gap: 24, flexWrap: 'wrap', margin: '10px 0 14px' }}>
                <div>
                  <div style={{ fontSize: 24, fontWeight: 800 }}>{s.balanceText}</div>
                  <div style={{ fontSize: 11, color: 'var(--text-faint)', fontWeight: 700 }}>
                    balance ({s.currency})
                  </div>
                </div>
                <div>
                  <div style={{ fontSize: 24, fontWeight: 800 }}>{s.mappedProducts}</div>
                  <div style={{ fontSize: 11, color: 'var(--text-faint)', fontWeight: 700 }}>mapped products</div>
                </div>
                <div>
                  <div style={{ fontSize: 24, fontWeight: 800 }}>{s.markupPercent}%</div>
                  <div style={{ fontSize: 11, color: 'var(--text-faint)', fontWeight: 700 }}>default markup</div>
                </div>
                <div>
                  <div style={{ fontSize: 24, fontWeight: 800 }}>
                    {s.lowBalanceThreshold > 0 ? s.lowBalanceThreshold.toLocaleString() : '—'}
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--text-faint)', fontWeight: 700 }}>low-balance alert</div>
                </div>
              </div>

              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                {canManage && (
                  <button className="abtn sm primary" onClick={() => setBrowse(s)}>
                    <Icon name="download" size={14} /> Browse &amp; import
                  </button>
                )}
                {canManage && (
                  <button
                    className="abtn sm"
                    disabled={sync.isPending}
                    title="Check every product linked to this supplier now: record its costs, pause links that sell at a loss, hide / re-show products with its stock. Never changes a selling price."
                    onClick={() => runSync(s)}
                  >
                    <Icon name="refresh" size={14} /> Check now
                  </button>
                )}
                <button className="abtn sm" onClick={() => setOrders(s)}>
                  <Icon name="bag" size={14} /> Recent orders
                </button>
                {canManage && (
                  <button className="abtn sm ghost" onClick={() => setSettings(s)}>
                    <Icon name="settings" size={14} /> Settings
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
            <b>Automatic supplier check is on</b> (every {mins} min) and{' '}
            <span className="st st-warn">only warns you</span> under Alerts — it doesn&apos;t change any product yet.
            Pausing a supplier that sells at a loss, switching to the next supplier, and taking out-of-stock products
            off sale are not turned on yet.
          </span>
        ) : (
          <span>
            <b>Automatic supplier check is on</b> (every {mins} min). It pauses a supplier that would sell at a loss,
            sends orders to the next supplier, and takes a product off sale when no supplier has it (and back on when
            one does). It never changes your prices.
          </span>
        )
      ) : (
        <span>
          <b>Automatic supplier check is off.</b> &ldquo;Check now&rdquo; on a supplier checks its products once
          {st.dryRun ? ' (warnings only)' : ''}.
        </span>
      )}
      {last && (
        <span className="faint">
          Last check {relativeTimeInline(last.finishedAt)}: {syncSummary(last)}
        </span>
      )}
      {st.running && <span className="st st-mute">running…</span>}
    </div>
  )
}

// --- Alerts ---

const ALERT_LABEL: Record<SupplierAlertKind, { label: string; cls: string }> = {
  cost_changed: { label: 'Cost changed', cls: 'st-warn' },
  loss_paused: { label: 'Selling at a loss', cls: 'st-danger' },
  loss_recovered: { label: 'Profitable again', cls: 'st-ok' },
  product_hidden: { label: 'Taken off sale', cls: 'st-danger' },
  product_shown: { label: 'Back on sale', cls: 'st-ok' },
  fx_missing: { label: 'Rate missing', cls: 'st-warn' },
  fallback_exhausted: { label: 'No supplier left', cls: 'st-danger' },
}

// Warning-only mode: nothing was done, so name what WOULD happen.
const ALERT_LABEL_WOULD: Partial<Record<SupplierAlertKind, string>> = {
  product_hidden: 'Would go off sale',
  product_shown: 'Would go back on sale',
  loss_recovered: 'Would be profitable again',
}

function alertLabel(a: SupplierAlert): { label: string; cls: string } {
  const k = ALERT_LABEL[a.kind] ?? { label: a.kind, cls: 'st-mute' }
  const would = a.dryRun ? ALERT_LABEL_WOULD[a.kind] : undefined
  return would ? { ...k, label: would } : k
}

function AlertsPanel({ canManage, suppliers }: { canManage: boolean; suppliers: SupplierView[] }) {
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
        <h3>Supplier alerts</h3>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: 8 }}>
          <button className={`abtn sm ${show === 'open' ? 'primary' : 'ghost'}`} onClick={() => setShow('open')}>
            Open
          </button>
          <button className={`abtn sm ${show === 'all' ? 'primary' : 'ghost'}`} onClick={() => setShow('all')}>
            All
          </button>
          {canManage && show === 'open' && alerts.length > 1 && (
            <button className="abtn sm" disabled={ackAll.isPending} onClick={() => ackAll.mutate()}>
              Acknowledge all
            </button>
          )}
        </div>
      </div>
      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message="Could not load alerts." onRetry={() => refetch()} />
      ) : alerts.length === 0 ? (
        <EmptyState
          title={show === 'open' ? 'Nothing needs your attention' : 'No alerts yet'}
          sub="You don't add alerts here — they appear by themselves when a supplier changes its price, a product would sell at a loss, a product runs out at every supplier, or an order can't be sent to any supplier."
        />
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table className="tbl" aria-label="Supplier alerts">
            <thead>
              <tr>
                <th>Kind</th>
                <th>What happened</th>
                <th>Last seen</th>
                <th style={{ textAlign: 'right' }} />
              </tr>
            </thead>
            <tbody>
              {alerts.map((a) => {
                const k = alertLabel(a)
                return (
                  <tr key={a.id} style={{ opacity: a.open ? 1 : 0.6 }}>
                    <td style={{ whiteSpace: 'nowrap' }}>
                      <span className={`st ${k.cls}`}>
                        <i className="d" /> {k.label}
                      </span>
                      {a.dryRun && (
                        <div>
                          <span
                            className="bdg"
                            style={{ marginTop: 4 }}
                            title="Nothing was changed: automatic changes are not turned on yet."
                          >
                            warning only
                          </span>
                        </div>
                      )}
                    </td>
                    <td style={{ fontSize: 13 }}>
                      {a.message}
                      <div className="faint" style={{ fontSize: 12, marginTop: 2 }}>
                        {a.productId && <Link to={`/products/${a.productId}/edit`}>Open product</Link>}
                        {a.orderId && <Link to={`/orders/${a.orderId}`}>Open order {orderNumber(a.orderId)}</Link>}
                        {a.provider && !a.productId && !a.orderId ? provName(a.provider) : ''}
                        {a.count > 1 ? ` · seen ${a.count}×` : ''}
                        {!a.open && a.ackBy ? ` · closed by ${a.ackBy}` : ''}
                      </div>
                    </td>
                    <td className="muted" style={{ whiteSpace: 'nowrap' }}>
                      {relativeTime(a.lastAt)}
                    </td>
                    <td style={{ textAlign: 'right' }}>
                      {canManage && a.open && (
                        <button className="abtn xs" disabled={ack.isPending} onClick={() => ack.mutate(a.id)}>
                          Acknowledge
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
            toast.success(`Imported ${r.created} product(s)${r.skipped.length ? `, ${r.skipped.length} skipped` : ''}`)
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
        <h3>Browse &amp; import — {supplier.name}</h3>
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message="The supplier catalog is unavailable (check balance/health)." onRetry={() => refetch()} />
      ) : products.length === 0 ? (
        <EmptyState title="No upstream products" />
      ) : (
        <>
          <div style={{ display: 'flex', gap: 10, alignItems: 'end', margin: '10px 0' }}>
            <div>
              <label className="alabel">Markup %</label>
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
              <label className="alabel">Search</label>
              <input
                className="afield"
                type="search"
                aria-label="Search supplier products"
                placeholder="Product, category or ID…"
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
              Select all shown
            </button>
            <div style={{ fontSize: 12, color: 'var(--text-faint)', whiteSpace: 'nowrap' }}>
              {search.trim() ? `${shown.length} of ${products.length} · ` : ''}
              {selected.size} selected
            </div>
            <button
              className="abtn primary"
              disabled={selected.size === 0 || importer.isPending}
              onClick={doImport}
            >
              <Icon name="plus" size={14} /> Import selected (hidden)
            </button>
          </div>

          <div style={{ maxHeight: 420, overflow: 'auto' }}>
            <table className="tbl">
              <thead>
                <tr>
                  <th style={{ width: 34 }} />
                  <th>Product</th>
                  <th>Category</th>
                  <th className="num">Base price</th>
                  <th>Type</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {shown.length === 0 && (
                  <tr>
                    <td colSpan={6} className="muted" style={{ padding: 16 }}>
                      No products match “{search.trim()}”.
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
                          <i className="d" /> mapped
                        </span>
                      ) : p.available ? (
                        <span className="st st-mute">
                          <i className="d" /> available
                        </span>
                      ) : (
                        <span className="st st-warn">
                          <i className="d" /> unavailable
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
        <h3>Recent orders — {supplier.name}</h3>
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message="Could not load orders." onRetry={() => refetch()} />
      ) : orders.length === 0 ? (
        <EmptyState title="No orders yet" sub="Orders fulfilled through this supplier will appear here." />
      ) : (
        <>
          <div style={{ maxHeight: 440, overflow: 'auto' }}>
            <table className="tbl">
              <thead>
                <tr>
                  <th>Order</th>
                  <th>Status</th>
                  <th className="num">Total</th>
                  <th>Upstream ref</th>
                  <th>Created</th>
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
                        <i className="d" /> {o.stuckAt ? 'stuck' : o.status}
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
        <h3>Settings — {supplier.name}</h3>
      </div>
      <div style={{ display: 'grid', gap: 12, margin: '12px 0' }}>
        <div>
          <label className="alabel">Low-balance alert threshold ({supplier.currency})</label>
          <input
            className="afield"
            type="number"
            value={threshold}
            onChange={(e) => setThreshold(e.target.value)}
            placeholder="0 = no alert"
          />
        </div>
        <div>
          <label className="alabel">Default import markup %</label>
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
            <label className="alabel">USD per 1 {supplier.currency}</label>
            <input
              className="afield"
              type="number"
              step="any"
              value={fx}
              onChange={(e) => setFx(e.target.value)}
              placeholder="e.g. 0.0000112 (1 / 89,500)"
            />
            <div className="ahint" style={{ margin: '4px 0 0' }}>
              The automatic supplier check uses this rate to compare this supplier&apos;s prices with yours (to spot
              selling at a loss).
              Empty = no loss check for {supplier.name}.
            </div>
          </div>
        )}
      </div>
      <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
        <button className="abtn" onClick={onClose}>
          Cancel
        </button>
        <button className="abtn primary" disabled={save.isPending} onClick={submit}>
          Save
        </button>
      </div>
    </Modal>
  )
}
