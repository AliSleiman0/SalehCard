import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  Icon,
  PageHead,
  Avatar,
  StatusBadge,
  Chip,
  Modal,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { money, downloadCsv } from '@/lib/utils'
import { ApiError } from '@/lib/api-client'
import { useUsers } from '@/features/users/hooks/useUsers'
import { adaptUser } from '@/features/users/lib/adaptUser'
import {
  useResellers,
  useTiers,
  useCreateTier,
  useUpdateTier,
  useDeleteTier,
  usePromoteToReseller,
} from '../hooks/useResellers'
import { adaptReseller, tierColor } from '../lib/adaptReseller'
import { listResellers, type ResellerStatus, type TierDef, type AdminReseller } from '../api/resellers'

export default function ResellerListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const [tier, setTier] = useState('')
  const [status, setStatus] = useState<'' | ResellerStatus>('')
  const [search, setSearch] = useState(searchParams.get('q') ?? '')
  const [debouncedSearch, setDebouncedSearch] = useState(search)
  const [page, setPage] = useState(1)
  const [editTier, setEditTier] = useState<TierDef | null>(null)
  const [creating, setCreating] = useState(false)
  const [adding, setAdding] = useState(false)
  const [exporting, setExporting] = useState(false)

  // Adopt the URL's ?q= (e.g. the top-bar global search navigates to /resellers?q=…).
  useEffect(() => {
    setSearch(searchParams.get('q') ?? '')
  }, [searchParams])

  // Debounce the search term so we issue one request per pause, not per keystroke.
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  const tiersQuery = useTiers()
  const tiers = tiersQuery.data?.data ?? []

  const { data, isLoading, isError, refetch } = useResellers({
    page,
    tier: tier || undefined,
    status: status || undefined,
    q: debouncedSearch.trim(),
  })

  const rows = useMemo(() => (data?.data ?? []).map(adaptReseller), [data])
  const meta = data?.meta

  const onTier = (v: string) => {
    setTier(v)
    setPage(1)
  }

  // Filters shared by the list query and the CSV export.
  const filters = { tier: tier || undefined, status: status || undefined, q: debouncedSearch.trim() }

  const handleExport = async () => {
    if (exporting) return
    setExporting(true)
    try {
      const all: AdminReseller[] = []
      let p = 1
      let pages = 1
      do {
        const res = await listResellers({ ...filters, page: p, limit: 100 })
        all.push(...(res.data ?? []))
        pages = res.meta?.pages ?? 1
        p++
      } while (p <= pages)

      const header = ['ID', 'Email', 'Phone', 'Tier', 'Margin %', 'Sub-balance', 'Orders', 'Volume', 'Status']
      const csvRows = all.map((raw) => {
        const r = adaptReseller(raw)
        return [r.id, r.email, r.phone, r.tier, r.margin, r.balance.toFixed(2), r.orders, r.vol.toFixed(2), r.status]
      })
      downloadCsv(`resellers-${new Date().toISOString().slice(0, 10)}.csv`, [header, ...csvRows])
    } finally {
      setExporting(false)
    }
  }

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_operations'), t('nav_resellers')]}
        title="Reseller & agent management"
        sub={meta ? `${meta.total.toLocaleString()} resellers across ${tiers.length} tiers` : ''}
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> Refresh
        </button>
        <button className="abtn" onClick={handleExport} disabled={exporting}>
          <Icon name="download" size={15} /> {exporting ? '…' : t('export')}
        </button>
        <button className="abtn primary" onClick={() => setAdding(true)}>
          <Icon name="plus" size={15} /> Add reseller
        </button>
      </PageHead>

      {/* Tier definition cards */}
      <div className="g3 mb16">
        {tiersQuery.isLoading ? (
          <LoadingSpinner />
        ) : (
          <>
            {tiers.map((tn) => {
              const tc = tierColor(tn.name)
              return (
                <div className="acard pad" key={tn.id} style={{ borderColor: tc + '55' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 9, marginBottom: 12 }}>
                    <span
                      style={{
                        width: 30,
                        height: 30,
                        borderRadius: 9,
                        background: tc + '22',
                        color: tc,
                        display: 'grid',
                        placeItems: 'center',
                      }}
                    >
                      <Icon name="shield" size={16} />
                    </span>
                    <b style={{ fontSize: 15 }}>{tn.name}</b>
                    <span className="faint" style={{ marginInlineStart: 'auto', fontSize: 12.5 }}>
                      {tn.count} agents
                    </span>
                  </div>
                  <div>
                    <div className="faint" style={{ fontSize: 11.5, fontWeight: 700 }}>
                      Discount
                    </div>
                    <div className="num" style={{ fontSize: 19, fontWeight: 800, color: tc }}>
                      {tn.marginPercent}%
                    </div>
                  </div>
                  <button className="abtn xs" style={{ width: '100%', marginTop: 12 }} onClick={() => setEditTier(tn)}>
                    <Icon name="edit" size={13} /> Edit tier
                  </button>
                </div>
              )
            })}
            {/* Always rendered — the only way to create the first tier. */}
            <button
              className="acard pad"
              onClick={() => setCreating(true)}
              style={{
                border: '1px dashed var(--border)',
                background: 'transparent',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                gap: 8,
                minHeight: 132,
                color: 'var(--muted)',
                cursor: 'pointer',
              }}
            >
              <Icon name="plus" size={20} />
              <b style={{ fontSize: 13.5 }}>{tiers.length === 0 ? 'Create your first tier' : 'New tier'}</b>
            </button>
          </>
        )}
      </div>

      <div className="acard">
        <div className="toolbar">
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input
              placeholder="Search email or phone…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <div className="chiprow">
            <Chip on={tier === ''} onClick={() => onTier('')}>
              All tiers
            </Chip>
            {tiers.map((tn) => (
              <Chip key={tn.id} on={tier === tn.name} onClick={() => onTier(tn.name)}>
                {tn.name}
              </Chip>
            ))}
          </div>
          <select
            className="select"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value as '' | ResellerStatus)
              setPage(1)
            }}
          >
            <option value="">All statuses</option>
            <option value="active">Active</option>
            <option value="suspended">Suspended</option>
          </select>
        </div>

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message="Couldn't load resellers." onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title="No resellers found" />
        ) : (
          <>
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>Reseller</th>
                    <th>Tier</th>
                    <th>Sub-balance</th>
                    <th>Margin</th>
                    <th>Orders</th>
                    <th>Volume</th>
                    <th>{t('status')}</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((r) => {
                    const tc = tierColor(r.tier)
                    return (
                      <tr key={r.id} className="clickable" onClick={() => navigate(`/resellers/${r.id}`)}>
                        <td>
                          <div className="cellprod">
                            <Avatar name={r.name} color={r.tier ? tc : undefined} />
                            <div className="pn">
                              <b>{r.name}</b>
                              <span>{r.email || r.phone}</span>
                            </div>
                          </div>
                        </td>
                        <td>
                          {r.tier ? (
                            <span
                              className="pill-role"
                              style={{ background: tc + '22', color: tc, border: '1px solid ' + tc + '55' }}
                            >
                              {r.tier}
                            </span>
                          ) : (
                            <span className="faint">—</span>
                          )}
                        </td>
                        <td className="num strong">{money(r.balance, r.cur)}</td>
                        <td className="num">{r.margin}%</td>
                        <td className="num muted">{r.orders.toLocaleString()}</td>
                        <td className="num">{money(r.vol, r.cur)}</td>
                        <td>
                          <StatusBadge s={r.status} />
                        </td>
                        <td onClick={(e) => e.stopPropagation()}>
                          <div className="row-actions">
                            <span className="iact">
                              <Icon name="chevright" size={16} />
                            </span>
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
            <Pagination
              page={meta?.page ?? 1}
              pages={meta?.pages ?? 1}
              total={meta?.total ?? rows.length}
              shown={rows.length}
              limit={meta?.limit}
              label={t('pg_resellers')}
              onPage={setPage}
            />
          </>
        )}
      </div>

      {creating && <TierModal tier={null} onClose={() => setCreating(false)} />}
      {editTier && <TierModal tier={editTier} onClose={() => setEditTier(null)} />}
      {adding && <AddResellerModal tiers={tiers} onClose={() => setAdding(false)} />}
    </div>
  )
}

/** Add reseller — search a user and promote them to the reseller role with a
 *  starting tier. Reuses the role + tier endpoints (no new backend). */
function AddResellerModal({ tiers, onClose }: { tiers: TierDef[]; onClose: () => void }) {
  const { t } = useTranslation()
  const [q, setQ] = useState('')
  const [debounced, setDebounced] = useState('')
  const [selectedId, setSelectedId] = useState('')
  const [tier, setTier] = useState(tiers[0]?.name ?? '')
  const [error, setError] = useState('')
  const promote = usePromoteToReseller()

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(q.trim()), 300)
    return () => clearTimeout(timer)
  }, [q])

  // Only non-resellers are candidates for promotion.
  const { data, isFetching } = useUsers({ q: debounced, limit: 8 })
  const candidates = (data?.data ?? []).map(adaptUser).filter((u) => u.role !== 'reseller')

  const submit = () => {
    if (!selectedId) {
      setError('Select a user to promote.')
      return
    }
    setError('')
    promote.mutate(
      { userId: selectedId, tier },
      { onSuccess: onClose, onError: (e) => setError(e instanceof ApiError ? e.message : 'Promotion failed.') },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={480}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 16 }}>Add reseller</h3>
        <label className="alabel">Find a user (email or phone)</label>
        <input className="afield" placeholder="Search users…" value={q} onChange={(e) => setQ(e.target.value)} />
        <div style={{ maxHeight: 220, overflowY: 'auto', marginTop: 10 }}>
          {isFetching && candidates.length === 0 ? (
            <div className="faint" style={{ fontSize: 12.5, padding: 8 }}>
              Searching…
            </div>
          ) : candidates.length === 0 ? (
            <div className="faint" style={{ fontSize: 12.5, padding: 8 }}>
              No eligible users found.
            </div>
          ) : (
            candidates.map((u) => (
              <button
                key={u.id}
                onClick={() => setSelectedId(u.id)}
                className={'abtn' + (selectedId === u.id ? ' primary' : '')}
                style={{ width: '100%', justifyContent: 'flex-start', marginBottom: 6 }}
              >
                <Avatar name={u.name} />
                <span style={{ marginInlineStart: 8 }}>
                  {u.email || u.phone} <span className="faint">· {u.role}</span>
                </span>
              </button>
            ))
          )}
        </div>
        <label className="alabel" style={{ marginTop: 14 }}>
          Starting tier
        </label>
        <select className="select" value={tier} onChange={(e) => setTier(e.target.value)} style={{ width: '100%' }}>
          <option value="">No tier</option>
          {tiers.map((tn) => (
            <option key={tn.id} value={tn.name}>
              {tn.name} ({tn.marginPercent}%)
            </option>
          ))}
        </select>
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 18, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={promote.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn primary" onClick={submit} disabled={promote.isPending || !selectedId}>
            <Icon name="check" size={15} /> Promote to reseller
          </button>
        </div>
      </div>
    </Modal>
  )
}

/** Create or edit a tier definition (name + margin). Pass tier=null to create.
 *  In edit mode it also offers delete (resellers on a removed tier fall back to
 *  retail — the backend treats an unknown tier as no margin). */
function TierModal({ tier, onClose }: { tier: TierDef | null; onClose: () => void }) {
  const { t } = useTranslation()
  const [name, setName] = useState(tier?.name ?? '')
  const [margin, setMargin] = useState(tier ? String(tier.marginPercent) : '')
  const [error, setError] = useState('')
  const create = useCreateTier()
  const update = useUpdateTier()
  const del = useDeleteTier()
  const busy = create.isPending || update.isPending || del.isPending

  const fail = (e: unknown) => setError(e instanceof ApiError ? e.message : 'Something went wrong.')

  const save = () => {
    const m = Number(margin)
    if (!name.trim()) {
      setError('Name is required.')
      return
    }
    if (!Number.isFinite(m) || m < 0 || m > 100) {
      setError('Margin must be between 0 and 100.')
      return
    }
    setError('')
    const input = { name: name.trim(), marginPercent: m }
    const opts = { onSuccess: onClose, onError: fail }
    if (tier) update.mutate({ id: tier.id, input }, opts)
    else create.mutate(input, opts)
  }

  const remove = () => {
    if (!tier) return
    const warn =
      tier.count > 0
        ? `Delete the ${tier.name} tier? ${tier.count} reseller(s) are on it and will fall back to retail pricing until reassigned.`
        : `Delete the ${tier.name} tier?`
    if (!window.confirm(warn)) return
    setError('')
    del.mutate(tier.id, { onSuccess: onClose, onError: fail })
  }

  return (
    <Modal onClose={onClose} maxWidth={420}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 16 }}>
          {tier ? `Edit ${tier.name} tier` : 'New tier'}
        </h3>
        <label className="alabel">Name</label>
        <input
          className="afield"
          placeholder="e.g. Gold"
          value={name}
          autoFocus={!tier}
          onChange={(e) => setName(e.target.value)}
        />
        <label className="alabel" style={{ marginTop: 14 }}>
          Discount / margin (%)
        </label>
        <input
          className="afield"
          type="number"
          min="0"
          max="100"
          step="0.5"
          placeholder="e.g. 10"
          value={margin}
          onChange={(e) => setMargin(e.target.value)}
        />
        <div className="ahint" style={{ marginTop: 8 }}>
          Resellers on this tier pay this % below retail on every product.
        </div>
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 18, justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            {tier && (
              <button className="abtn danger" onClick={remove} disabled={busy}>
                <Icon name="trash" size={15} /> {t('delete')}
              </button>
            )}
          </div>
          <div style={{ display: 'flex', gap: 10 }}>
            <button className="abtn" onClick={onClose} disabled={busy}>
              {t('cancel')}
            </button>
            <button className="abtn primary" onClick={save} disabled={busy}>
              <Icon name="check" size={15} /> {t('save')}
            </button>
          </div>
        </div>
      </div>
    </Modal>
  )
}
