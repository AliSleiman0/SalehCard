import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta } from '@/types'

/** Moderation state of a KYC submission. */
export type KycStatus = 'pending' | 'approved' | 'rejected'

/** Identity-document type. */
export type KycDocumentType = 'passport' | 'id_card' | 'license'

/** A KYC submission as returned by the admin API (the record + account contact). */
export interface AdminKyc {
  id: string
  userId: string
  userContact: string
  fullName: string
  dateOfBirth: string
  placeOfBirth: string
  placeOfResidence: string
  documentType: KycDocumentType
  documentNumber: string
  documentFrontUrl?: string
  documentBackUrl?: string
  status: KycStatus
  rejectionReason?: string
  reviewedBy?: string
  reviewedAt?: string
  createdAt: string
  updatedAt: string
}

export interface KycListParams {
  page?: number
  limit?: number
  status?: KycStatus | ''
  q?: string
}

const ADMIN = '/api/admin/kyc'

export function listKyc(
  params: KycListParams,
): Promise<ApiResponse<AdminKyc[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.status) q.set('status', params.status)
  if (params.q) q.set('q', params.q)
  const qs = q.toString()
  return apiClient.get<AdminKyc[]>(qs ? `${ADMIN}?${qs}` : ADMIN)
}

export function setKycStatus(
  id: string,
  status: KycStatus,
  rejectionReason = '',
): Promise<ApiResponse<AdminKyc>> {
  return apiClient.put<AdminKyc>(`${ADMIN}/${id}`, { status, rejectionReason })
}
