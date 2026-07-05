import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Icon,
  PageHead,
  Chip,
  Pagination,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { useAuditLog } from '../hooks/useAuditLog'
import type { AuditEntry } from '../api/audit'

/** Quick action filters — the money/permission actions an operator audits most. */
const FILTERS: [string, string][] = [
  ['', 'All'],
  ['wallet.adjust', 'Wallet adjustments'],
  ['reseller.balance_adjust', 'Reseller balances'],
  ['user.role_change', 'Role changes'],
  ['order.refund', 'Refunds'],
  ['kyc.decision', 'KYC decisions'],
]

/** Renders the summary object as compact `key: value` pills. */
function SummaryPills({ summary }: { summary?: Record<string, unknown> }) {
  if (!summary || Object.keys(summary).length === 0) return null
  return (
    <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginTop: 4 }}>
      {Object.entries(summary).map(([k, v]) => (
        <span
          key={k}
          className="mono"
          style={{
            fontSize: 11.5,
            padding: '2px 8px',
            borderRadius: 6,
            background: 'var(--surface-2)',
            border: '1px solid var(--border)',
            color: 'var(--text-dim)',
          }}
        >
          {k}: {String(v)}
        </span>
      ))}
    </div>
  )
}

function Row({ e }: { e: AuditEntry }) {
  const at = new Date(e.at)
  return (
    <div style={{ display: 'flex', gap: 14, padding: '12px 18px', alignItems: 'flex-start', borderBottom: '1px solid var(--border)' }}>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
          <b className="mono" style={{ fontSize: 13 }}>{e.action}</b>
          <span className="faint" style={{ fontSize: 12 }}>
            {e.targetType} <span className="mono">{e.targetId.slice(-8)}</span>
          </span>
          <span className="faint" style={{ fontSize: 12, marginInlineStart: 'auto' }} title={at.toISOString()}>
            {at.toLocaleString()}
          </span>
        </div>
        <div style={{ fontSize: 12.5, color: 'var(--text-dim)', marginTop: 2 }}>
          by <b>{e.actorEmail || e.actorId || '—'}</b>
        </div>
        <SummaryPills summary={e.summary} />
      </div>
    </div>
  )
}

export default function AuditLogPage() {
  const { t } = useTranslation()
  const [action, setAction] = useState('')
  const [page, setPage] = useState(1)

  const { data, isLoading, isError, refetch } = useAuditLog({
    page,
    action: action || undefined,
  })

  const rows = useMemo(() => data?.data ?? [], [data])
  const meta = data?.meta

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_system'), 'Activity log']}
        title="Activity log"
        sub={meta ? `${meta.total.toLocaleString()} recorded actions` : ''}
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> Refresh
        </button>
      </PageHead>

      <div className="acard">
        <div className="toolbar">
          <div className="chiprow">
            {FILTERS.map(([k, l]) => (
              <Chip
                key={k || 'all'}
                on={action === k}
                onClick={() => {
                  setAction(k)
                  setPage(1)
                }}
              >
                {l}
              </Chip>
            ))}
          </div>
        </div>

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message="Couldn't load the activity log." onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title="No recorded actions yet" sub="Admin actions (role changes, wallet adjustments, refunds, moderation decisions) appear here." />
        ) : (
          <>
            <div>
              {rows.map((e) => (
                <Row key={e.id} e={e} />
              ))}
            </div>
            <Pagination
              page={meta?.page ?? 1}
              pages={meta?.pages ?? 1}
              total={meta?.total ?? rows.length}
              shown={rows.length}
              limit={meta?.limit}
              label={t('pg_actions')}
              onPage={setPage}
            />
          </>
        )}
      </div>
    </div>
  )
}
