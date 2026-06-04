import { createContext, useContext, useState } from 'react'
import type { ReactNode } from 'react'
import { Icon, type IconName } from './Icon'

export type ToastFn = (msg: string, icon?: IconName) => void

interface ToastItem {
  id: number
  msg: string
  icon: IconName
}

const ToastCtx = createContext<ToastFn | null>(null)

let _id = 0

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([])

  const push: ToastFn = (msg, icon = 'check') => {
    const id = ++_id
    setToasts((t) => [...t, { id, msg, icon }])
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), 2200)
  }

  return (
    <ToastCtx.Provider value={push}>
      {children}
      <div className="toast-wrap">
        {toasts.map((t) => (
          <div className="toast" key={t.id}>
            <span style={{ color: 'var(--ok)' }}>
              <Icon name={t.icon} size={18} />
            </span>
            {t.msg}
          </div>
        ))}
      </div>
    </ToastCtx.Provider>
  )
}

export function useToast(): ToastFn {
  const ctx = useContext(ToastCtx)
  if (!ctx) return () => {}
  return ctx
}
