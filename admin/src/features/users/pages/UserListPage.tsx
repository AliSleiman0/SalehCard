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
  Modal,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { useBulk } from '@/hooks/useBulk'
import { money, downloadCsv } from '@/lib/utils'
import { toast } from '@/stores/toast'
import { ApiError } from '@/lib/api-client'
import { useUsers, useBulkUserAction, useBulkEmail } from '../hooks/useUsers'
import { adaptUser } from '../lib/adaptUser'
import { listUsers, type UserStatus, type BulkUserAction, type AdminUser } from '../api/users'
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
  const [exporting, setExporting] = useState(false)
  const [emailIds, setEmailIds] = useState<string[] | null>(null)

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

  // Filters shared by the list query and the CSV export (page added per call).
  const filters = {
    role: role === 'all' ? ('' as const) : role,
    status: status === 'all' ? ('' as const) : status,
    q: debouncedSearch.trim(),
  }

  const { data, isLoading, isError, refetch } = useUsers({ page, ...filters })
  const bulkAction = useBulkUserAction()

  const rows = useMemo(() => (data?.data ?? []).map(adaptUser), [data])
  const meta = data?.meta
  const bulk = useBulk(rows.map((r) => r.id))

  // Reset to page 1 whenever a filter changes (search resets via its debounce).
  const onFilter = <T,>(setter: (v: T) => void) => (v: T) => {
    setter(v)
    setPage(1)
  }

  const runBulk = (action: BulkUserAction) => {
    const verb = action === 'suspend' ? 'Suspend' : 'Activate'
    if (!window.confirm(`${verb} ${bulk.sel.length} user(s)?`)) return
    bulkAction.mutate({ ids: bulk.sel, action }, { onSuccess: () => bulk.clear() })
  }

  // Export users matching the current filters (all pages — the backend caps
  // limit at 100, so page to total). CSV mirrors the table columns.
  const handleExport = async () => {
    if (exporting) return
    setExporting(true)
    try {
      const all: AdminUser[] = []
      let p = 1
      let pages = 1
      do {
        const res = await listUsers({ ...filters, page: p, limit: 100 })
        all.push(...(res.data ?? []))
        pages = res.meta?.pages ?? 1
        p++
      } while (p <= pages)

      const header = ['ID', 'Email', 'Phone', 'Role', 'Status', 'Wallet', 'Orders', 'Total spent', 'Joined']
      const csvRows = all.map((u) => {
        const v = adaptUser(u)
        return [v.id, v.email, v.phone, v.role, v.status, v.balance.toFixed(2), v.orders, v.spent.toFixed(2), v.joined]
      })
      downloadCsv(`users-${new Date().toISOString().slice(0, 10)}.csv`, [header, ...csvRows])
    } finally {
      setExporting(false)
    }
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
        <button className="abtn" onClick={handleExport} disabled={exporting}>
          <Icon name="download" size={15} /> {exporting ? '…' : t('export')}
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
              <button className="abtn xs ok" onClick={() => runBulk('activate')} disabled={bulkAction.isPending}>
                <Icon name="check" size={13} /> {t('activate')}
              </button>
              <button className="abtn xs danger" onClick={() => runBulk('suspend')} disabled={bulkAction.isPending}>
                <Icon name="x" size={13} /> {t('suspend')}
              </button>
              <button className="abtn xs" onClick={() => setEmailIds(bulk.sel)}>
                <Icon name="send" size={13} /> Email
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

      {emailIds && (
        <BulkEmailModal
          ids={emailIds}
          onClose={() => setEmailIds(null)}
          onSent={() => {
            setEmailIds(null)
            bulk.clear()
          }}
        />
      )}
    </div>
  )
}

/** Compose + send a plain-text email to the selected users. Recipients without
 *  an email are skipped server-side; the toast reports how many were queued. */
function BulkEmailModal({ ids, onClose, onSent }: { ids: string[]; onClose: () => void; onSent: () => void }) {
  const { t } = useTranslation()
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const [error, setError] = useState('')
  const send = useBulkEmail()

  const submit = () => {
    if (!subject.trim() || !body.trim()) {
      setError('Subject and message are required.')
      return
    }
    setError('')
    send.mutate(
      { ids, subject: subject.trim(), body: body.trim() },
      {
        onSuccess: (res) => {
          toast.success(`Email queued to ${res.data?.queued ?? 0} recipient(s).`)
          onSent()
        },
        onError: (e) => setError(e instanceof ApiError ? e.message : 'Send failed.'),
      },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={520}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>Email {ids.length} user(s)</h3>
        <p style={{ fontSize: 12.5, color: 'var(--text-dim)', marginBottom: 16 }}>
          Users without an email address are skipped automatically.
        </p>
        <label className="alabel">Subject</label>
        <input className="afield" value={subject} onChange={(e) => setSubject(e.target.value)} />
        <label className="alabel" style={{ marginTop: 14 }}>
          Message
        </label>
        <textarea
          className="afield"
          style={{ minHeight: 120 }}
          value={body}
          onChange={(e) => setBody(e.target.value)}
        />
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 18, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={send.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn primary" onClick={submit} disabled={send.isPending}>
            <Icon name="send" size={15} /> {send.isPending ? 'Sending…' : 'Send'}
          </button>
        </div>
      </div>
    </Modal>
  )
}
