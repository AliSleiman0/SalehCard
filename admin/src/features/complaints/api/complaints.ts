import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta } from '@/types'

export type ComplaintStatus = 'open' | 'in_progress' | 'resolved' | 'closed'
export type ComplaintCategory = 'order' | 'topup' | 'account' | 'other'

export const COMPLAINT_STATUSES: ComplaintStatus[] = ['open', 'in_progress', 'resolved', 'closed']
export const COMPLAINT_CATEGORIES: ComplaintCategory[] = ['order', 'topup', 'account', 'other']

export const STATUS_LABEL: Record<ComplaintStatus, string> = {
  open: 'Open',
  in_progress: 'In progress',
  resolved: 'Resolved',
  closed: 'Closed',
}

export const STATUS_CLASS: Record<ComplaintStatus, string> = {
  open: 'st st-warn',
  in_progress: 'st st-info',
  resolved: 'st st-ok',
  closed: 'st st-mute',
}

export const CATEGORY_LABEL: Record<ComplaintCategory, string> = {
  order: 'Order',
  topup: 'Top-up',
  account: 'Account',
  other: 'Other',
}

/** A customer / admin identity as the complaint endpoints return it. */
export interface ComplaintContact {
  id: string
  name?: string
  email?: string
  phone?: string
  status?: string
}

/** One queue row. */
export interface ComplaintRow {
  id: string
  number: number
  ref: string
  category: ComplaintCategory
  subject: string
  status: ComplaintStatus
  assigneeId?: string
  orderId?: string
  customer: ComplaintContact
  awaitingReply: boolean
  lastActivityAt: string
  createdAt: string
}

export interface ComplaintMessage {
  id: string
  author: 'customer' | 'admin'
  adminId?: string
  adminLabel?: string
  body: string
  photoUrl?: string
  internal: boolean
  createdAt: string
}

export interface ComplaintOrderSummary {
  id: string
  status: string
  total: number
  currency: string
  createdAt: string
  items: { title: { en: string; ar: string }; denomination: string; qty: number }[]
}

/** The full console view (thread incl. internal notes, contact, linked order). */
export interface ComplaintDetail {
  id: string
  number: number
  ref: string
  userId: string
  category: ComplaintCategory
  orderId?: string
  subject: string
  status: ComplaintStatus
  assigneeId?: string
  messages: ComplaintMessage[]
  awaitingReply: boolean
  customer: ComplaintContact
  order?: ComplaintOrderSummary
  createdAt: string
  updatedAt: string
}

export interface ComplaintCounts {
  byStatus: Record<ComplaintStatus, number>
  awaitingReply: number
}

export interface ComplaintListParams {
  page?: number
  limit?: number
  status?: ComplaintStatus | ''
  category?: ComplaintCategory | ''
  assignee?: string
  search?: string
}

const ADMIN = '/api/admin/complaints'

export function listComplaints(
  params: ComplaintListParams,
): Promise<ApiResponse<ComplaintRow[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.status) q.set('status', params.status)
  if (params.category) q.set('category', params.category)
  if (params.assignee) q.set('assignee', params.assignee)
  if (params.search) q.set('search', params.search)
  const qs = q.toString()
  return apiClient.get<ComplaintRow[]>(qs ? `${ADMIN}?${qs}` : ADMIN)
}

export function getComplaintCounts(): Promise<ApiResponse<ComplaintCounts>> {
  return apiClient.get<ComplaintCounts>(`${ADMIN}/counts`)
}

export function listAssignees(): Promise<ApiResponse<ComplaintContact[]>> {
  return apiClient.get<ComplaintContact[]>(`${ADMIN}/assignees`)
}

export function getComplaint(id: string): Promise<ApiResponse<ComplaintDetail>> {
  return apiClient.get<ComplaintDetail>(`${ADMIN}/${id}`)
}

export function replyComplaint(id: string, body: string, internal: boolean): Promise<ApiResponse<ComplaintDetail>> {
  return apiClient.post<ComplaintDetail>(`${ADMIN}/${id}/messages`, { body, internal })
}

export function setComplaintStatus(id: string, status: ComplaintStatus): Promise<ApiResponse<ComplaintDetail>> {
  return apiClient.put<ComplaintDetail>(`${ADMIN}/${id}/status`, { status })
}

export function setComplaintAssignee(id: string, adminId: string | null): Promise<ApiResponse<ComplaintDetail>> {
  return apiClient.put<ComplaintDetail>(`${ADMIN}/${id}/assignee`, { adminId })
}

/** Display label for a contact: name, else email, else phone, else a short id. */
export function contactLabel(c: ComplaintContact | undefined): string {
  if (!c) return '—'
  return c.name || c.email || c.phone || c.id.slice(-8)
}

/** wa.me link for a phone number (digits only), or '' when there is none. */
export function whatsappLink(phone: string | undefined): string {
  const digits = (phone ?? '').replace(/\D/g, '')
  return digits ? `https://wa.me/${digits}` : ''
}
