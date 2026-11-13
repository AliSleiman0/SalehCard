import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

export type ComplaintStatus = 'open' | 'in_progress' | 'resolved' | 'closed'
export type ComplaintCategory = 'order' | 'topup' | 'account' | 'other'

export const COMPLAINT_CATEGORIES: ComplaintCategory[] = ['order', 'topup', 'account', 'other']

/** One thread entry as the customer sees it (internal notes never arrive). */
export interface ComplaintMessage {
  id: string
  author: 'customer' | 'admin'
  body: string
  photoUrl?: string
  createdAt: string
}

export interface Complaint {
  id: string
  number: number
  ref: string
  category: ComplaintCategory
  orderId?: string
  subject: string
  status: ComplaintStatus
  /** Absent on list rows. */
  messages?: ComplaintMessage[]
  createdAt: string
  updatedAt: string
}

export interface CreateComplaintInput {
  category: ComplaintCategory
  subject: string
  message: string
  orderId?: string
  photoUrl?: string
}

const BASE = '/api/v1/complaints'

export function fetchComplaints(): Promise<ApiResponse<Complaint[]>> {
  return apiClient.get<Complaint[]>(`${BASE}?limit=50`)
}

export function fetchComplaint(id: string): Promise<ApiResponse<Complaint>> {
  return apiClient.get<Complaint>(`${BASE}/${id}`)
}

export function createComplaint(input: CreateComplaintInput): Promise<ApiResponse<Complaint>> {
  return apiClient.post<Complaint>(BASE, input)
}

export function replyComplaint(id: string, body: string, photoUrl?: string): Promise<ApiResponse<Complaint>> {
  return apiClient.post<Complaint>(`${BASE}/${id}/messages`, { body, photoUrl })
}

/** Uploads one complaint photo (multipart `image`) → its stored URL. */
export function uploadComplaintPhoto(file: Blob): Promise<ApiResponse<{ url: string }>> {
  const form = new FormData()
  form.append('image', file, 'photo.jpg')
  return apiClient.postForm<{ url: string }>(`${BASE}/photos`, form)
}

/** Badge class per status (storefront `.badge` variants). */
export function statusBadgeClass(s: ComplaintStatus): string {
  switch (s) {
    case 'open':
      return 'badge badge-agent'
    case 'in_progress':
      return 'badge badge-secure'
    case 'resolved':
      return 'badge badge-instant'
    default:
      return 'badge badge-soft'
  }
}

/** Customer-side form validation; returns an i18n key or '' when valid. */
export function validateComplaint(input: { subject: string; message: string }): string {
  const subject = input.subject.trim()
  const message = input.message.trim()
  if (!subject) return 'complaint_err_subject'
  if ([...subject].length > 120) return 'complaint_err_subject_long'
  if (!message) return 'complaint_err_message'
  if ([...message].length > 2000) return 'complaint_err_message_long'
  return ''
}
