import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Avatar, Chip, EmptyState, ErrorState, Icon, LoadingSpinner, PageHead, Pagination } from '@/components'
import {
  CATEGORY_LABEL,
  COMPLAINT_CATEGORIES,
  STATUS_CLASS,
  STATUS_LABEL,
  contactLabel,
  type ComplaintCategory,
  type ComplaintStatus,
} from '../api/complaints'
import { useAssignees, useComplaintCounts, useComplaints } from '../hooks/useComplaints'

const TABS: [ComplaintStatus | '', string][] = [
  ['open', STATUS_LABEL.open],
  ['in_progress', STATUS_LABEL.in_progress],
  ['resolved', STATUS_LABEL.resolved],
  ['closed', STATUS_LABEL.closed],
  ['', 'All'],
]

export default function ComplaintListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [status, setStatus] = useState<ComplaintStatus | ''>('open')
  const [category, setCategory] = useState<ComplaintCategory | ''>('')
  const [assignee, setAssignee] = useState('')
  const [search, setSearch] = useState('')
  const [debounced, setDebounced] = useState('')
  const [page, setPage] = useState(1)

  useEffect(() => {
    const h = setTimeout(() => {
      setDebounced(search.trim())
      setPage(1)
    }, 300)
    return () => clearTimeout(h)
  }, [search])

  const { data, isLoading, isError, refetch } = useComplaints({
    page,
    status,
    category,
    assignee,
    search: debounced,
  })
  const { data: countsRes } = useComplaintCounts()
  const { data: adminsRes } = useAssignees()
  const admins = useMemo(() => adminsRes?.data ?? [], [adminsRes])
  const adminName = (id?: string) => (id ? contactLabel(admins.find((a) => a.id === id)) : '')
  const rows = data?.data ?? []
  const meta = data?.meta
  const counts = countsRes?.data

  const onFilter =
    <T,>(set: (v: T) => void) =>
    (v: T) => {
      set(v)
      setPage(1)
    }

  return (
    <div className="page page-wide">
      <PageHead
        back="/"
        crumbs={[t('grp_operations'), t('nav_complaints')]}
        title={t('nav_complaints')}
        sub={
          counts
            ? `${counts.byStatus.open + counts.byStatus.in_progress} open · ${counts.awaitingReply} awaiting a reply`
            : ''
        }
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> Refresh
        </button>
      </PageHead>

      <div className="acard">
        <div className="toolbar">
          <div className="chiprow">
            {TABS.map(([k, l]) => {
              const n = k && counts ? counts.byStatus[k] : undefined
              return (
                <Chip key={k || 'all'} on={status === k} onClick={() => onFilter(setStatus)(k)}>
                  {l}
                  {n !== undefined && n > 0 ? ` (${n})` : ''}
                </Chip>
              )
            })}
          </div>
          <div className="tb-spacer" />
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input
              placeholder="Number (C-1042), subject, phone, email or name…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <select
            className="select"
            value={category}
            onChange={(e) => onFilter(setCategory)(e.target.value as ComplaintCategory | '')}
          >
            <option value="">All categories</option>
            {COMPLAINT_CATEGORIES.map((c) => (
              <option key={c} value={c}>
                {CATEGORY_LABEL[c]}
              </option>
            ))}
          </select>
          <select className="select" value={assignee} onChange={(e) => onFilter(setAssignee)(e.target.value)}>
            <option value="">Any assignee</option>
            <option value="me">Assigned to me</option>
            <option value="none">Unassigned</option>
            {admins.map((a) => (
              <option key={a.id} value={a.id}>
                {contactLabel(a)}
              </option>
            ))}
          </select>
        </div>

        {isLoading ? (
          <LoadingSpinner />
        ) : isError && !data ? (
          <ErrorState message="Couldn't load complaints." onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title="No complaints" sub="Customer complaints submitted from the app or the website appear here." />
        ) : (
          <>
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>Number</th>
                    <th>Customer</th>
                    <th>Subject</th>
                    <th>Category</th>
                    <th>{t('status')}</th>
                    <th>Assignee</th>
                    <th>Last activity</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((c) => (
                    <tr
                      key={c.id}
                      className="clickable"
                      onClick={() => navigate(`/complaints/${c.id}`)}
                      style={c.awaitingReply ? { background: 'rgba(255,176,46,.06)' } : undefined}
                    >
                      <td>
                        <span className="mono strong">{c.ref}</span>
                      </td>
                      <td>
                        <div className="cellprod">
                          <Avatar name={contactLabel(c.customer)} />
                          <div className="pn">
                            <b>{contactLabel(c.customer)}</b>
                            <span>{c.customer.phone || c.customer.email || ''}</span>
                          </div>
                        </div>
                      </td>
                      <td style={{ maxWidth: 320 }}>
                        <div style={{ fontWeight: 700, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {c.subject}
                        </div>
                        {c.awaitingReply && (
                          <span style={{ color: 'var(--warn)', fontSize: 11.5, fontWeight: 700 }}>● Awaiting reply</span>
                        )}
                      </td>
                      <td>
                        <span className="bdg">{CATEGORY_LABEL[c.category]}</span>
                      </td>
                      <td>
                        <span className={STATUS_CLASS[c.status]}>
                          <i className="d" />
                          {STATUS_LABEL[c.status]}
                        </span>
                      </td>
                      <td className="muted" style={{ fontSize: 12.5 }}>
                        {adminName(c.assigneeId) || '—'}
                      </td>
                      <td className="muted" style={{ fontSize: 12 }}>
                        {new Date(c.lastActivityAt).toLocaleString()}
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
              label="complaints"
              onPage={setPage}
            />
          </>
        )}
      </div>
    </div>
  )
}
