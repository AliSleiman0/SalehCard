import type { ApiResponse } from '@/types'

// ApiError carries the backend error code alongside the human message so hooks
// and screens can branch on machine codes (KYC_REQUIRED, OTP_THROTTLED, …).
export class ApiError extends Error {
  code?: string
  constructor(message: string, code?: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
  }
}

// unwrap returns the payload of a successful envelope, or throws an ApiError
// carrying the server's code + message (falling back to `fallback`).
export function unwrap<T>(res: ApiResponse<T>, fallback: string): T {
  if (!res.success || res.data === undefined) {
    throw new ApiError(res.error?.message ?? fallback, res.error?.code)
  }
  return res.data
}
