import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { PageHead, Icon, Modal, Pagination, LoadingSpinner, EmptyState, ErrorState } from '@/components'
import { useCan } from '@/stores/auth'
import { toast } from '@/stores/toast'
import {
  useSuppliersList,
  useSupplierCatalog,
  useSupplierOrders,
  useSyncSupplier,
  useImportProducts,
  useUpdateSupplierSettings,
} from '../hooks/useSuppliers'
import type { SupplierView, SupplierHealth, CatalogProductView } from '../api/suppliers'

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

  const sync = useSyncSupplier()
  const runSync = (s: SupplierView) => {
    sync.mutate(s.id, {
      onSuccess: (res) => {
        const r = res.data
        if (r) {
          toast.success(
            `${s.name}: ${r.updated} updated, ${r.unavailable} unavailable, ${r.drift.length} price changes`,
          )
        }
      },
    })
  }

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_operations'), t('nav_suppliers')]}
        title={t('nav_suppliers')}
        sub="Upstream fulfillment suppliers — balance, health, mapped products, and catalog import."
      />

      {isLoading ? (
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
                    onClick={() => runSync(s)}
                  >
                    <Icon name="refresh" size={14} /> Sync catalog
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

// --- Browse & import ---

function BrowseModal({ supplier, onClose }: { supplier: SupplierView; onClose: () => void }) {
  const { data, isLoading, isError, refetch } = useSupplierCatalog(supplier.id)
  const products = useMemo<CatalogProductView[]>(() => data?.data?.products ?? [], [data])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [markup, setMarkup] = useState<string>(String(supplier.markupPercent || ''))
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
            <div style={{ marginLeft: 'auto', fontSize: 12, color: 'var(--text-faint)' }}>
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
                {products.map((p) => (
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
                      <a href={`/orders/${o.id}`}>{o.id.slice(-8)}</a>
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
  const save = useUpdateSupplierSettings()

  const submit = () => {
    save.mutate(
      {
        id: supplier.id,
        patch: {
          lowBalanceThreshold: threshold.trim() === '' ? 0 : Number(threshold),
          markupPercent: markup.trim() === '' ? 0 : Number(markup),
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
