import { useToastStore, type ToastKind } from '@/stores/toast'

const DOT_COLOR: Record<ToastKind, string> = {
  success: 'var(--ok)',
  error: 'var(--danger)',
  info: 'var(--brand-1)',
}

/** Renders the global toast queue using the pre-styled .toast-wrap / .toast CSS.
 *  Mounted once in AdminLayout; click a toast to dismiss it early. */
export function Toaster() {
  const toasts = useToastStore((s) => s.toasts)
  const dismiss = useToastStore((s) => s.dismiss)
  return (
    <div className="toast-wrap">
      {toasts.map((t) => (
        <div key={t.id} className="toast" onClick={() => dismiss(t.id)} role="status">
          <span
            style={{ width: 8, height: 8, borderRadius: 99, background: DOT_COLOR[t.kind], flexShrink: 0 }}
          />
          <span>{t.message}</span>
        </div>
      ))}
    </div>
  )
}
