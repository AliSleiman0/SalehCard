import type { IconName } from '@/components'
import type { NotificationKind } from '../types'

// Maps a notification kind to a gradient icon tile. Unknown kinds fall back to a
// neutral bell — title/body come from the server, so new kinds still render.
export function iconForKind(kind: NotificationKind): { icon: IconName; grad: string } {
  switch (kind) {
    case 'order_completed':
      return { icon: 'check', grad: 'var(--grad)' }
    case 'order_refunded':
      return { icon: 'repeat', grad: 'linear-gradient(135deg,#ffd76b,#ffae34)' }
    case 'topup_approved':
      return { icon: 'wallet', grad: 'var(--grad-cyan)' }
    case 'topup_rejected':
      return { icon: 'close', grad: 'linear-gradient(135deg,#ff6b6b,#c0392b)' }
    case 'kyc_approved':
      return { icon: 'shield', grad: 'var(--grad)' }
    case 'kyc_rejected':
      return { icon: 'close', grad: 'linear-gradient(135deg,#ff6b6b,#c0392b)' }
    default:
      return { icon: 'bell', grad: 'linear-gradient(135deg,#7a2bff,#d633ff)' }
  }
}

export function fmtNotifDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' })
}
