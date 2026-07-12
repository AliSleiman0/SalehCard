export type NotificationKind =
  | 'order_completed'
  | 'order_refunded'
  | 'topup_approved'
  | 'topup_rejected'
  | 'kyc_approved'
  | 'kyc_rejected'
  | (string & {})

export interface AppNotification {
  id: string
  kind: NotificationKind
  title: string
  body: string
  data?: Record<string, string>
  readAt?: string
  createdAt: string
}
