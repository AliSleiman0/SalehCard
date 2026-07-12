import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon, LoadingSpinner, ErrorState, EmptyState } from '@/components'
import { useNotifications } from '../hooks/useNotifications'
import { useMarkAllRead } from '../hooks/useMarkAllRead'
import { iconForKind, fmtNotifDate } from '../lib/adaptNotification'

export default function NotificationsPage() {
  const { t } = useTranslation()
  const query = useNotifications()
  const markAllRead = useMarkAllRead()
  const marked = useRef(false)

  // Mark everything read once when the inbox opens (best-effort).
  useEffect(() => {
    if (marked.current) return
    marked.current = true
    markAllRead.mutate()
  }, [markAllRead])

  const items = query.data ?? []

  return (
    <div className="wrap" style={{ padding: '26px 0 50px', maxWidth: 640 }}>
      <h1 className="h1" style={{ marginBottom: 22 }}>
        {t('notifications')}
      </h1>
      {query.isLoading ? (
        <LoadingSpinner />
      ) : query.isError ? (
        <ErrorState title={t('retry')} onRetry={() => query.refetch()} retryLabel={t('retry')} />
      ) : items.length === 0 ? (
        <EmptyState icon="bell" title={t('notif_empty')} sub={t('notif_empty_sub')} />
      ) : (
        <div className="panel card-pad" style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {items.map((n) => {
            const { icon, grad } = iconForKind(n.kind)
            const unread = !n.readAt
            return (
              <div
                key={n.id}
                className="lrow"
                style={{
                  alignItems: 'flex-start',
                  gap: 12,
                  padding: '12px 8px',
                  background: unread ? 'var(--grad-soft)' : undefined,
                  borderRadius: 10,
                }}
              >
                <span className="mi" style={{ background: grad, flex: 'none' }}>
                  <Icon name={icon} size={16} />
                </span>
                <div className="col" style={{ gap: 2, flex: 1, minWidth: 0 }}>
                  <span style={{ fontWeight: 800 }}>{n.title}</span>
                  {n.body && <span className="small muted">{n.body}</span>}
                  <span className="tiny faint">{fmtNotifDate(n.createdAt)}</span>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
