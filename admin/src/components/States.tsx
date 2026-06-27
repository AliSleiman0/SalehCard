import { useTranslation } from 'react-i18next'
import { Icon, type IconName } from './Icon'

export function LoadingSpinner({ label }: { label?: string }) {
  const { t } = useTranslation()
  return (
    <div
      className="muted"
      style={{ display: 'grid', placeItems: 'center', padding: 48, gap: 10, fontWeight: 600 }}
    >
      <div className="skel" style={{ width: 28, height: 28, borderRadius: 99 }} />
      {label ?? t('loading')}
    </div>
  )
}

export function EmptyState({ icon = 'box', title, sub }: { icon?: IconName; title: string; sub?: string }) {
  return (
    <div style={{ display: 'grid', placeItems: 'center', padding: 48, textAlign: 'center', gap: 8 }}>
      <div
        style={{
          width: 48,
          height: 48,
          borderRadius: 14,
          background: 'var(--surface-2)',
          display: 'grid',
          placeItems: 'center',
          color: 'var(--text-faint)',
        }}
      >
        <Icon name={icon} size={22} />
      </div>
      <div style={{ fontWeight: 800, fontSize: 15 }}>{title}</div>
      {sub && <div className="faint" style={{ fontSize: 13 }}>{sub}</div>}
    </div>
  )
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div style={{ display: 'grid', placeItems: 'center', padding: 48, textAlign: 'center', gap: 10 }}>
      <div
        style={{
          width: 48,
          height: 48,
          borderRadius: 14,
          background: 'rgba(255,77,109,.14)',
          color: 'var(--danger)',
          display: 'grid',
          placeItems: 'center',
        }}
      >
        <Icon name="alert" size={22} />
      </div>
      <div style={{ fontWeight: 800, fontSize: 15 }}>{message}</div>
      {onRetry && (
        <button className="abtn sm" onClick={onRetry}>
          <Icon name="refresh" size={14} /> Retry
        </button>
      )}
    </div>
  )
}

/** Banner shown on screens whose backend is stubbed (501) or still mock-driven.
 *  An optional note overrides the default coming-soon/not-implemented label. */
export function ComingSoonNote({ mock, note }: { mock?: boolean; note?: string }) {
  const { t } = useTranslation()
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 10,
        padding: '12px 16px',
        marginBottom: 16,
        borderRadius: 'var(--ar-md)',
        background: 'var(--grad-soft)',
        border: '1px solid var(--ff-code-bd)',
        color: 'var(--text-dim)',
        fontSize: 12.5,
        fontWeight: 600,
      }}
    >
      <Icon name="bolt" size={15} />
      {note ?? (mock ? t('coming_soon') : t('not_implemented'))}
    </div>
  )
}
