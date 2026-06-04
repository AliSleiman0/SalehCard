import { Icon, type IconName } from './Icon'
import { Button } from './Button'

export function LoadingSpinner({ size = 28 }: { size?: number }) {
  return (
    <div className="col center" style={{ padding: 24 }}>
      <div className="spinner" style={{ width: size, height: size }} />
    </div>
  )
}

export interface EmptyStateProps {
  icon?: IconName
  title: string
  sub?: string
  cta?: string
  onCta?: () => void
  tone?: 'ghost' | 'error'
}

export function EmptyState({
  icon = 'search',
  title,
  sub,
  cta,
  onCta,
  tone = 'ghost',
}: EmptyStateProps) {
  return (
    <div
      className="col center"
      style={{
        textAlign: 'center',
        gap: 12,
        height: '100%',
        justifyContent: 'center',
        padding: 18,
      }}
    >
      <div
        style={{
          width: 72,
          height: 72,
          borderRadius: 22,
          display: 'grid',
          placeItems: 'center',
          background: tone === 'error' ? 'rgba(255,77,109,.12)' : 'var(--surface-2)',
          color: tone === 'error' ? 'var(--danger)' : 'var(--text-dim)',
          border: tone === 'error' ? '1px solid var(--danger)' : '1px solid var(--border)',
        }}
      >
        <Icon name={icon} size={30} />
      </div>
      <h3 className="h3" style={{ fontSize: 19 }}>
        {title}
      </h3>
      {sub && (
        <p className="small muted" style={{ maxWidth: 260 }}>
          {sub}
        </p>
      )}
      {cta && (
        <Button
          variant={tone === 'error' ? 'danger' : 'primary'}
          size="sm"
          style={{ marginTop: 4 }}
          onClick={onCta}
        >
          {cta}
        </Button>
      )}
    </div>
  )
}

export interface ErrorStateProps {
  title?: string
  sub?: string
  retryLabel?: string
  onRetry?: () => void
}

export function ErrorState({ title = '', sub = '', retryLabel, onRetry }: ErrorStateProps) {
  return (
    <EmptyState
      icon="close"
      tone="error"
      title={title}
      sub={sub}
      cta={onRetry ? retryLabel || 'Retry' : undefined}
      onCta={onRetry}
    />
  )
}
