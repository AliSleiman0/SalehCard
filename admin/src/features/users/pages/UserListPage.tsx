import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  Icon,
  PageHead,
  Avatar,
  RoleBadge,
  StatusBadge,
  Checkbox,
  Chip,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { useBulk } from '@/hooks/useBulk'
import { money } from '@/lib/utils'
import { useUsers } from '../hooks/useUsers'
import { adaptUser } from '../lib/adaptUser'
import type { UserStatus } from '../api/users'
import type { UserRole } from '@/types'

const ROLES: [string, string][] = [
  ['all', 'All'],
  ['customer', 'Customers'],
  ['reseller', 'Resellers'],
  ['admin', 'Admins'],
]

export default function UserListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const [role, setRole] = useState<'all' | UserRole>('all')
  const [status, setStatus] = useState<'all' | UserStatus>('all')
  const [search, setSearch] = useState(searchParams.get('q') ?? '')
  const [debouncedSearch, setDebouncedSearch] = useState(search)
  const [page, setPage] = useState(1)

  // Adopt the URL's ?q= (e.g. the top-bar global search navigates to /users?q=…).
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

  const { data, isLoading, isError, refetch } = useUsers({
    page,
    role: role === 'all' ? '' : role,
    status: status === 'all' ? '' : status,
    q: debouncedSearch.trim(),
  })

  const rows = useMemo(() => (data?.data ?? []).map(adaptUser), [data])
  const meta = data?.meta
  const bulk = useBulk(rows.map((r) => r.id))

  // Reset to page 1 whenever a filter changes (search resets via its debounce).
  const onFilter = <T,>(setter: (v: T) => void) => (v: T) => {
    setter(v)
    setPage(1)
  }

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_operations'), t('nav_users')]}
        title={t('nav_users')}
        sub={meta ? `${meta.total.toLocaleString()} registered users` : ''}
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> Refresh
        </button>
        <button className="abtn" disabled title="Coming soon">
          <Icon name="download" size={15} /> {t('export')}
        </button>
      </PageHead>

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
            {ROLES.map(([k, l]) => (
              <Chip key={k} on={role === k} onClick={() => onFilter(setRole)(k as 'all' | UserRole)}>
                {l}
              </Chip>
            ))}
          </div>
          <select
            className="select"
            value={status}
            onChange={(e) => onFilter(setStatus)(e.target.value as 'all' | UserStatus)}
          >
            <option value="all">All statuses</option>
            <option value="active">Active</option>
            <option value="suspended">Suspended</option>
          </select>
        </div>

        {bulk.some && (
          <div className="bulkbar">
            <Checkbox on onClick={bulk.clear} />
            <span>
              {bulk.sel.length} {t('selected')}
            </span>
            <div className="ba-act">
              <button className="abtn xs" disabled title="Coming soon">
                <Icon name="send" size={13} /> Email
              </button>
              <button className="abtn xs danger" disabled title="Coming soon">
                <Icon name="x" size={13} /> {t('suspend')}
              </button>
            </div>
          </div>
        )}

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message="Couldn't load users." onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title="No users found" />
        ) : (
          <>
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th style={{ width: 36 }}>
                      <Checkbox on={bulk.all} onClick={bulk.toggleAll} />
                    </th>
                    <th>User</th>
                    <th>Role</th>
                    <th>Wallet</th>
                    <th>Orders</th>
                    <th>Total spent</th>
                    <th>Joined</th>
                    <th>{t('status')}</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((u) => (
                    <tr key={u.id} className="clickable" onClick={() => navigate(`/users/${u.id}`)}>
                      <td onClick={(e) => e.stopPropagation()}>
                        <Checkbox on={bulk.sel.includes(u.id)} onClick={() => bulk.toggle(u.id)} />
                      </td>
                      <td>
                        <div className="cellprod">
                          <Avatar name={u.name} />
                          <div className="pn">
                            <b>{u.name}</b>
                            <span>{u.email || u.phone}</span>
                          </div>
                        </div>
                      </td>
                      <td>
                        <RoleBadge role={u.role} />
                      </td>
                      <td className="num strong">{money(u.balance, u.cur)}</td>
                      <td className="num muted">{u.orders}</td>
                      <td className="num">{money(u.spent, u.cur)}</td>
                      <td className="muted" style={{ fontSize: 12.5 }}>
                        {u.joined}
                      </td>
                      <td>
                        <StatusBadge s={u.status} />
                      </td>
                      <td onClick={(e) => e.stopPropagation()}>
                        <div className="row-actions">
                          <span className="iact">
                            <Icon name="chevright" size={16} />
                          </span>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Pagination
              page={meta?.page ?? 1}
              pages={meta?.pages ?? 1}
              total={meta?.total ?? rows.length}
              shown={rows.length}
              limit={meta?.limit}
              label="users"
              onPage={setPage}
            />
          </>
        )}
      </div>
    </div>
  )
}
