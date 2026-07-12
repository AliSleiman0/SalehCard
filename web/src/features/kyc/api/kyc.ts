import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'
import type { KycProfile, KycStatus, KycSubmissionInput } from '../types'

// Normalizes the server status (tolerates 'approved' as 'verified', unknown → 'unverified').
function normalizeStatus(raw: string | undefined): KycStatus {
  if (raw === 'approved' || raw === 'verified') return 'verified'
  if (raw === 'pending' || raw === 'rejected' || raw === 'unverified') return raw
  return 'unverified'
}

export async function fetchKycProfile(): Promise<ApiResponse<KycProfile>> {
  const res = await apiClient.get<KycProfile>('/api/v1/kyc/me')
  if (res.success && res.data) {
    res.data.status = normalizeStatus(res.data.status)
  }
  return res
}

export async function submitKyc(input: KycSubmissionInput): Promise<ApiResponse<KycProfile>> {
  const res = await apiClient.post<KycProfile>('/api/v1/kyc', input)
  if (res.success && res.data) {
    res.data.status = normalizeStatus(res.data.status)
  }
  return res
}

// uploadKycDocument posts one photo (multipart field `image`) and returns its URL.
export async function uploadKycDocument(blob: Blob): Promise<ApiResponse<{ imageUrl: string }>> {
  const form = new FormData()
  form.append('image', blob, 'document.jpg')
  return apiClient.postForm<{ imageUrl: string }>('/api/v1/kyc/documents', form)
}
