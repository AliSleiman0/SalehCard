import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Chip, EmptyState, ErrorState, LoadingSpinner } from '@/components'
import { listLogs, type ChatIntent, type ChatOutcome, type LogFilter } from '../api/chatbot'
import { formatDateTime } from '@/lib/utils'

/** Labels are translation keys (looked up at render time). */
const OUTCOME: Record<ChatOutcome, { label: string; cls: string }> = {
  answered: { label: 'bot_outcome_answered', cls: 'st-ok' },
  unanswered: { label: 'bot_outcome_unanswered', cls: 'st-warn' },
  off_topic: { label: 'bot_outcome_off_topic', cls: 'st-mute' },
  filtered: { label: 'bot_outcome_filtered', cls: 'st-danger' },
  error: { label: 'bot_outcome_error', cls: 'st-danger' },
}

/** The finer kind of reply, shown under the result where it adds something. */
const INTENT_NOTE: Partial<Record<ChatIntent, string>> = {
  handoff: 'bot_intent_handoff',
  clarify: 'bot_intent_clarify',
  smalltalk: 'bot_intent_smalltalk',
}

/** The AI conversations (kept 30 days): what customers asked and what the
 *  assistant did. "Not covered" questions are the gaps to add to the knowledge. */
export function ConversationsTab() {
  const { t } = useTranslation()
  const [filter, setFilter] = useState<LogFilter>('unanswered')
  const [page, setPage] = useState(1)

  const q = useQuery({
    queryKey: ['admin', 'chatbot', 'logs', filter, page],
    queryFn: async () => {
      const res = await listLogs(filter, page)
      if (!res.success || !res.data) throw new Error(res.error?.message ?? 'Failed to load')
      return res.data
    },
  })
  const pick = (f: LogFilter) => {
    setFilter(f)
    setPage(1)
  }

  return (
    <>
      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 12 }}>
        <Chip on={filter === 'unanswered'} onClick={() => pick('unanswered')}>
          {t('bot_filter_unanswered')}
        </Chip>
        <Chip on={filter === 'flagged'} onClick={() => pick('flagged')}>
          {t('bot_filter_flagged')}
        </Chip>
        <Chip on={filter === ''} onClick={() => pick('')}>
          {t('all')}
        </Chip>
      </div>
      <div className="acard" style={{ padding: 0 }}>
        {q.isLoading ? (
          <LoadingSpinner />
        ) : q.isError ? (
          <ErrorState message={t('bot_logs_error')} onRetry={() => q.refetch()} />
        ) : !q.data || q.data.items.length === 0 ? (
          <EmptyState
            icon="send"
            title={t('bot_logs_empty')}
            sub={t('bot_logs_empty_sub')}
          />
        ) : (
          <table className="tbl">
            <thead>
              <tr>
                <th>{t('bot_col_when')}</th>
                <th>{t('bot_col_question')}</th>
                <th>{t('bot_col_answer')}</th>
                <th>{t('bot_col_result')}</th>
                <th>{t('cmp_col_customer')}</th>
              </tr>
            </thead>
            <tbody>
              {q.data.items.map((l) => {
                const o = OUTCOME[l.outcome] ?? OUTCOME.error
                return (
                  <tr key={l.id}>
                    <td className="faint" style={{ whiteSpace: 'nowrap', fontSize: 12.5 }}>
                      {formatDateTime(l.createdAt)}
                    </td>
                    <td style={{ maxWidth: 280 }} dir="auto">
                      {l.question}
                    </td>
                    <td
                      className="muted"
                      style={{ maxWidth: 380, fontSize: 12.5, whiteSpace: 'pre-wrap' }}
                      dir="auto"
                    >
                      {l.answer || '—'}
                      {l.sources && l.sources.length > 0 && (
                        <div className="faint" dir="auto" style={{ fontSize: 11.5, marginTop: 6 }}>
                          {t('bot_from', { sources: l.sources.map((s) => t('cmn_quote', { s })).join(' · ') })}
                        </div>
                      )}
                    </td>
                    <td>
                      <span className={`bdg ${o.cls}`}>{t(o.label)}</span>
                      {l.intent && INTENT_NOTE[l.intent] && (
                        <div className="faint" style={{ fontSize: 11.5, marginTop: 4 }}>
                          {t(INTENT_NOTE[l.intent]!)}
                        </div>
                      )}
                      {l.flagged && (
                        <span className="bdg st-danger" style={{ marginInlineStart: 6 }}>
                          {t('bot_reported')}
                        </span>
                      )}
                    </td>
                    <td>
                      <Link to={`/users/${l.userId}`} style={{ fontSize: 12.5 }}>
                        {t('view')}
                      </Link>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </div>
      {q.data && (page > 1 || q.data.hasMore) && (
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 12 }}>
          <button className="abtn" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            {t('bot_newer')}
          </button>
          <button className="abtn" disabled={!q.data.hasMore} onClick={() => setPage((p) => p + 1)}>
            {t('bot_older')}
          </button>
        </div>
      )}
    </>
  )
}
