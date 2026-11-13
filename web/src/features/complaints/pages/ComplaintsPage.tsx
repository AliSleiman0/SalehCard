import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Button, ErrorState, Icon, LoadingSpinner, Panel } from '@/components'
import { AcctSidebar } from '@/features/auth/components/AcctSidebar'
import { statusBadgeClass } from '../api/complaints'
import { useComplaints } from '../hooks/useComplaints'

export default function ComplaintsPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const query = useComplaints()
  const rows = query.data ?? []

  return (
    <div className="wrap" style={{ padding: '26px 0 50px' }}>
      <div className="row" style={{ justifyContent: 'space-between', alignItems: 'center', marginBottom: 24, gap: 12, flexWrap: 'wrap' }}>
        <h1 className="h1" style={{ margin: 0 }}>
          {t('complaints')}
        </h1>
        <Button variant="primary" size="sm" onClick={() => navigate('/complaints/new')}>
          <Icon name="plus" size={16} />
          {t('complaint_submit')}
        </Button>
      </div>
      <div className="cols-acct">
        <AcctSidebar active="complaints" />
        <Panel>
          {query.isLoading ? (
            <LoadingSpinner />
          ) : query.isError && !query.data ? (
            <ErrorState
              title={t('no_results')}
              sub={t('no_results_sub')}
              onRetry={() => query.refetch()}
              retryLabel={t('retry')}
            />
          ) : rows.length === 0 ? (
            <div className="col" style={{ alignItems: 'center', gap: 12, padding: '36px 8px', textAlign: 'center' }}>
              <Icon name="chat" size={32} />
              <b>{t('complaints_empty')}</b>
              <span className="small muted">{t('complaints_empty_sub')}</span>
              <Button variant="primary" onClick={() => navigate('/complaints/new')}>
                {t('complaint_submit')}
              </Button>
            </div>
          ) : (
            <div className="col" style={{ gap: 0 }}>
              {rows.map((c) => (
                <a
                  key={c.id}
                  className="clickable"
                  onClick={() => navigate(`/complaints/${c.id}`)}
                  style={{
                    display: 'flex',
                    gap: 12,
                    alignItems: 'center',
                    padding: '14px 4px',
                    borderBottom: '1px solid var(--border)',
                    color: 'inherit',
                  }}
                >
                  <div className="col" style={{ gap: 3, minWidth: 0, flex: 1 }}>
                    <div className="row" style={{ gap: 8, alignItems: 'center' }}>
                      <span className="small faint" style={{ fontWeight: 700 }}>
                        {c.ref}
                      </span>
                      <span className={statusBadgeClass(c.status)} style={{ fontSize: 10.5 }}>
                        {t('complaint_status_' + c.status)}
                      </span>
                    </div>
                    <b style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{c.subject}</b>
                    <span className="tiny faint">
                      {t('complaint_last_activity')}: {new Date(c.updatedAt).toLocaleString()}
                    </span>
                  </div>
                  <Icon name="chevron" size={16} />
                </a>
              ))}
            </div>
          )}
        </Panel>
      </div>
    </div>
  )
}
