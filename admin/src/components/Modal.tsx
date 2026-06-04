import { useEffect } from 'react'

interface ModalProps {
  onClose: () => void
  children: React.ReactNode
  maxWidth?: number
}

/** Centered overlay modal (`.overlay` + `.modal`). Closes on backdrop click + Esc. */
export function Modal({ onClose, children, maxWidth = 460 }: ModalProps) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <div className="overlay" onClick={onClose}>
      <div className="modal" style={{ maxWidth }} onClick={(e) => e.stopPropagation()}>
        {children}
      </div>
    </div>
  )
}
