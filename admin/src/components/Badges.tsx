import { useTranslation } from 'react-i18next'
import type { FulfillmentType } from '@/types'

/** Short fulfillment key used across the admin UI for color-coding. */
export type FfKey = 'code' | 'credit' | 'transfer'

export function ffKey(ft: FulfillmentType): FfKey {
  if (ft === 'account_credit') return 'credit'
  if (ft === 'transfer') return 'transfer'
  return 'code'
}

const FF_CLASS: Record<FfKey, string> = {
  code: 'ff-code',
  credit: 'ff-credit',
  transfer: 'ff-transfer',
}

const FF_I18N: Record<FfKey, string> = {
  code: 'ff_code',
  credit: 'ff_credit',
  transfer: 'ff_transfer',
}

/** Consistent blue(code) / green(credit) / orange(transfer) badge. */
export function FfBadge({ ff }: { ff: FfKey }) {
  const { t } = useTranslation()
  return (
    <span className={'bdg ' + FF_CLASS[ff]}>
      <i className="d" />
      {t(FF_I18N[ff])}
    </span>
  )
}

const STATUS_MAP: Record<string, [string, string]> = {
  active: ['st-ok', 'Active'],
  delivered: ['st-ok', 'Delivered'],
  completed: ['st-ok', 'Completed'],
  approved: ['st-ok', 'Approved'],
  verified: ['st-ok', 'Verified'],
  processing: ['st-warn', 'Processing'],
  pending: ['st-warn', 'Pending'],
  scheduled: ['st-warn', 'Scheduled'],
  invited: ['st-warn', 'Invited'],
  draft: ['st-mute', 'Draft'],
  paused: ['st-mute', 'Paused'],
  expired: ['st-mute', 'Expired'],
  depleted: ['st-mute', 'Depleted'],
  rejected: ['st-mute', 'Rejected'],
  out: ['st-danger', 'Out of stock'],
  failed: ['st-danger', 'Failed'],
  refunded: ['st-danger', 'Refunded'],
  suspended: ['st-danger', 'Suspended'],
  deleted: ['st-mute', 'Deleted'],
}

export function StatusBadge({ s }: { s: string }) {
  const [cls, label] = STATUS_MAP[s] || ['st-mute', s]
  return (
    <span className={'st ' + cls}>
      <i className="d" />
      {label}
    </span>
  )
}

export function RoleBadge({ role }: { role: string }) {
  return <span className={'pill-role role-' + role}>{role}</span>
}
