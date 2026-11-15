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
import { formatDateTime } from '@/lib/utils'

const TABS: [ComplaintStatus | '', string][] = [
  ['open', STATUS_LABEL.open],
  ['in_progress', STATUS_LABEL.in_progress],
  ['resolved', STATUS_LABEL.resolved],
  ['closed', STATUS_LABEL.closed],
  ['', 'all'],
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
            ? t('cmp_list_sub', { open: counts.byStatus.open + counts.byStatus.in_progress, awaiting: counts.awaitingReply })
            : ''
        }
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> {t('exp_refresh')}
        </button>
      </PageHead>

      <div className="acard">
        <div className="toolbar">
          <div className="chiprow">
            {TABS.map(([k, l]) => {
              const n = k && counts ? counts.byStatus[k] : undefined
              return (
                <Chip key={k || 'all'} on={status === k} onClick={() => onFilter(setStatus)(k)}>
                  {t(l)}
                  {n !== undefined && n > 0 ? ` (${n})` : ''}
                </Chip>
              )
            })}
          </div>
          <div className="tb-spacer" />
          <div className="fsearch">
            <Icon name="search" size={15} />
            <input
              placeholder={t('cmp_search_ph')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <select
            className="select"
            value={category}
            onChange={(e) => onFilter(setCategory)(e.target.value as ComplaintCategory | '')}
          >
            <option value="">{t('exp_all_categories')}</option>
            {COMPLAINT_CATEGORIES.map((c) => (
              <option key={c} value={c}>
                {t(CATEGORY_LABEL[c])}
              </option>
            ))}
          </select>
          <select className="select" value={assignee} onChange={(e) => onFilter(setAssignee)(e.target.value)}>
            <option value="">{t('cmp_any_assignee')}</option>
            <option value="me">{t('cmp_assigned_to_me')}</option>
            <option value="none">{t('cmp_unassigned')}</option>
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
          <ErrorState message={t('cmp_load_error')} onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title={t('cmp_empty')} sub={t('cmp_empty_sub')} />
        ) : (
          <>
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>{t('cmp_col_number')}</th>
                    <th>{t('cmp_col_customer')}</th>
                    <th>{t('cmp_col_subject')}</th>
                    <th>{t('exp_category')}</th>
                    <th>{t('status')}</th>
                    <th>{t('cmp_assignee')}</th>
                    <th>{t('cmp_col_last_activity')}</th>
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
                          <span style={{ color: 'var(--warn)', fontSize: 11.5, fontWeight: 700 }}>● {t('cmp_awaiting_reply')}</span>
                        )}
                      </td>
                      <td>
                        <span className="bdg">{t(CATEGORY_LABEL[c.category])}</span>
                      </td>
                      <td>
                        <span className={STATUS_CLASS[c.status]}>
                          <i className="d" />
                          {t(STATUS_LABEL[c.status])}
                        </span>
                      </td>
                      <td className="muted" style={{ fontSize: 12.5 }}>
                        {adminName(c.assigneeId) || '—'}
                      </td>
                      <td className="muted" style={{ fontSize: 12 }}>
                        {formatDateTime(c.lastActivityAt)}
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
              label={t('cmp_pg_complaints')}
              onPage={setPage}
            />
          </>
        )}
      </div>
    </div>
  )
}
