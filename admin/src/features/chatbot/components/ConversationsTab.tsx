import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Chip, EmptyState, ErrorState, LoadingSpinner } from '@/components'
import { listLogs, type ChatOutcome, type LogFilter } from '../api/chatbot'

const OUTCOME: Record<ChatOutcome, { label: string; cls: string }> = {
  answered: { label: 'Answered', cls: 'st-ok' },
  unanswered: { label: 'Not covered', cls: 'st-warn' },
  filtered: { label: 'Blocked', cls: 'st-danger' },
  error: { label: 'Error', cls: 'st-danger' },
}

/** The AI conversations (kept 30 days): what customers asked and what the
 *  assistant did. "Not covered" questions are the gaps to write articles for. */
export function ConversationsTab() {
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
          Not answered
        </Chip>
        <Chip on={filter === 'flagged'} onClick={() => pick('flagged')}>
          Reported by customers
        </Chip>
        <Chip on={filter === ''} onClick={() => pick('')}>
          All
        </Chip>
      </div>
      <div className="acard" style={{ padding: 0 }}>
        {q.isLoading ? (
          <LoadingSpinner />
        ) : q.isError ? (
          <ErrorState message="Could not load the conversations." onRetry={() => q.refetch()} />
        ) : !q.data || q.data.items.length === 0 ? (
          <EmptyState
            icon="send"
            title="Nothing here"
            sub="AI questions appear here once customers use the assistant (kept for 30 days)."
          />
        ) : (
          <table className="tbl">
            <thead>
              <tr>
                <th>When</th>
                <th>Question</th>
                <th>Answer</th>
                <th>Result</th>
                <th>Customer</th>
              </tr>
            </thead>
            <tbody>
              {q.data.items.map((l) => {
                const o = OUTCOME[l.outcome] ?? OUTCOME.error
                return (
                  <tr key={l.id}>
                    <td className="faint" style={{ whiteSpace: 'nowrap', fontSize: 12.5 }}>
                      {new Date(l.createdAt).toLocaleString()}
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
                    </td>
                    <td>
                      <span className={`bdg ${o.cls}`}>{o.label}</span>
                      {l.flagged && (
                        <span className="bdg st-danger" style={{ marginInlineStart: 6 }}>
                          Reported
                        </span>
                      )}
                    </td>
                    <td>
                      <Link to={`/users/${l.userId}`} style={{ fontSize: 12.5 }}>
                        View
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
            Newer
          </button>
          <button className="abtn" disabled={!q.data.hasMore} onClick={() => setPage((p) => p + 1)}>
            Older
          </button>
        </div>
      )}
    </>
  )
}
