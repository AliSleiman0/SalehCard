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
import { money } from '@/lib/utils'
import { ApiError } from '@/lib/api-client'
import { useResellers, useTiers, useUpdateTier } from '../hooks/useResellers'
import { adaptReseller, tierColor } from '../lib/adaptReseller'
import type { ResellerStatus, TierDef } from '../api/resellers'

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
        <button className="abtn" disabled title="Coming soon">
          <Icon name="download" size={15} /> {t('export')}
        </button>
        <button className="abtn primary" disabled title="Promote a user to reseller from the Users page">
          <Icon name="plus" size={15} /> Add reseller
        </button>
      </PageHead>

      {/* Tier definition cards */}
      <div className="g3 mb16">
        {tiersQuery.isLoading ? (
          <LoadingSpinner />
        ) : (
          tiers.map((tn) => {
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
          })
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
              label="resellers"
              onPage={setPage}
            />
          </>
        )}
      </div>

      {editTier && <TierEditModal tier={editTier} onClose={() => setEditTier(null)} />}
    </div>
  )
}

/** Edit a tier definition's margin. */
function TierEditModal({ tier, onClose }: { tier: TierDef; onClose: () => void }) {
  const { t } = useTranslation()
  const [name, setName] = useState(tier.name)
  const [margin, setMargin] = useState(String(tier.marginPercent))
  const [error, setError] = useState('')
  const update = useUpdateTier()

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
    update.mutate(
      { id: tier.id, input: { name: name.trim(), marginPercent: m } },
      {
        onSuccess: onClose,
        onError: (e) => setError(e instanceof ApiError ? e.message : 'Update failed.'),
      },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={420}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 16 }}>Edit {tier.name} tier</h3>
        <label className="alabel">Name</label>
        <input className="afield" value={name} onChange={(e) => setName(e.target.value)} />
        <label className="alabel" style={{ marginTop: 14 }}>
          Discount / margin (%)
        </label>
        <input
          className="afield"
          type="number"
          min="0"
          max="100"
          step="0.5"
          value={margin}
          onChange={(e) => setMargin(e.target.value)}
        />
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 18, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={update.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn primary" onClick={save} disabled={update.isPending}>
            <Icon name="check" size={15} /> {t('save')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
