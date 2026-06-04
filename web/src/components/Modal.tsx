import { useEffect } from 'react'
import type { ReactNode } from 'react'
import { Icon } from './Icon'
import { Button } from './Button'

export function Modal({
  open,
  onClose,
  title,
  children,
}: {
  open: boolean
  onClose: () => void
  title?: string
  children: ReactNode
}) {
  useEffect(() => {
    if (!open) return
    const h = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="overlay" onClick={onClose}>
      <div className="modal card-pad" onClick={(e) => e.stopPropagation()}>
        {title && (
          <div className="row between" style={{ marginBottom: 16 }}>
            <h3 className="h3">{title}</h3>
            <Button
              variant="ghost"
              size="sm"
              style={{ padding: 9, borderRadius: 999 }}
              onClick={onClose}
            >
              <Icon name="close" size={16} />
            </Button>
          </div>
        )}
        {children}
      </div>
    </div>
  )
}
