import { useMemo, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
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
import { formatDateTime } from '@/lib/utils'

/** `user.bulk_sms` → `aud_act_user_bulk_sms` (dots and other separators become underscores). */
const keyPart = (s: string) => s.replace(/[^a-zA-Z0-9_]/g, '_')

/** Quick action filters — the money/permission actions an operator audits most.
 *  Labels are translation keys (looked up at render time). */
const FILTERS: [string, string][] = [
  ['', 'all'],
  ['wallet.adjust', 'aud_f_wallet'],
  ['reseller.balance_adjust', 'aud_f_reseller'],
  ['user.role_change', 'aud_f_role_change'],
  ['role.update', 'aud_f_role_edit'],
  ['order.refund', 'aud_f_refund'],
  ['kyc.decision', 'aud_f_kyc'],
]

/** Renders the summary object as compact `key: value` pills. */
function SummaryPills({ summary }: { summary?: Record<string, unknown> }) {
  const { t } = useTranslation()
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
          {t(`aud_key_${keyPart(k)}`, { defaultValue: k })}: {String(v)}
        </span>
      ))}
    </div>
  )
}

function Row({ e }: { e: AuditEntry }) {
  const { t } = useTranslation()
  const at = new Date(e.at)
  return (
    <div style={{ display: 'flex', gap: 14, padding: '12px 18px', alignItems: 'flex-start', borderBottom: '1px solid var(--border)' }}>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
          <b className="mono" style={{ fontSize: 13 }} title={e.action}>
            {t(`aud_act_${keyPart(e.action)}`, { defaultValue: e.action })}
          </b>
          <span className="faint" style={{ fontSize: 12 }}>
            {t(`aud_target_${keyPart(e.targetType)}`, { defaultValue: e.targetType })} <span className="mono">{e.targetId.slice(-8)}</span>
          </span>
          <span className="faint" style={{ fontSize: 12, marginInlineStart: 'auto' }} title={Number.isNaN(at.getTime()) ? undefined : at.toISOString()}>
            {formatDateTime(e.at)}
          </span>
        </div>
        <div style={{ fontSize: 12.5, color: 'var(--text-dim)', marginTop: 2 }}>
          <Trans i18nKey="aud_by" values={{ who: e.actorEmail || e.actorId || '—' }} components={{ b: <b /> }} />
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
        back="/"
        crumbs={[t('grp_system'), t('nav_audit')]}
        title={t('nav_audit')}
        sub={meta ? t('aud_sub', { n: meta.total.toLocaleString() }) : ''}
      >
        <button className="abtn" onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> {t('exp_refresh')}
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
                {t(l)}
              </Chip>
            ))}
          </div>
        </div>

        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message={t('aud_load_error')} onRetry={() => refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState title={t('aud_empty')} sub={t('aud_empty_sub')} />
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
