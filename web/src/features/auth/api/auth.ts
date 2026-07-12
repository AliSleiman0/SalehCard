import { apiClient } from '@/lib/api-client'
import type { ApiResponse, AuthResponse, SavedPlayerId, User } from '@/types'

export interface LoginInput {
  email: string
  password: string
}

export interface RegisterInput {
  name?: string
  email: string
  password: string
  locale?: string
}

export interface RequestOtpInput {
  phone: string
}

export interface VerifyOtpInput {
  phone: string
  code: string
  // password + name are sent only on first sign-in (account creation).
  password?: string
  name?: string
}

export interface LoginPhoneInput {
  phone: string
  password: string
}

export async function login(input: LoginInput): Promise<ApiResponse<AuthResponse>> {
  return apiClient.post<AuthResponse>('/api/v1/auth/login', input)
}

export async function register(input: RegisterInput): Promise<ApiResponse<AuthResponse>> {
  return apiClient.post<AuthResponse>('/api/v1/auth/register', input)
}

// requestOtp fires an SMS code to the phone (rate-limited 5/min/IP server-side).
export async function requestOtp(input: RequestOtpInput): Promise<ApiResponse<{ sent: boolean }>> {
  return apiClient.post<{ sent: boolean }>('/api/v1/auth/otp/request', input)
}

// verifyOtp completes a code login, or creates the account when password + name
// are supplied on a first sign-in.
export async function verifyOtp(input: VerifyOtpInput): Promise<ApiResponse<AuthResponse>> {
  return apiClient.post<AuthResponse>('/api/v1/auth/otp/verify', input)
}

export async function loginPhone(input: LoginPhoneInput): Promise<ApiResponse<AuthResponse>> {
  return apiClient.post<AuthResponse>('/api/v1/auth/login-phone', input)
}

// refresh exchanges the httpOnly refresh cookie for a fresh access token + user.
// Used for on-load session restoration; the api-client also calls the raw
// endpoint directly during its 401 retry flow.
export async function refresh(): Promise<ApiResponse<AuthResponse>> {
  return apiClient.post<AuthResponse>('/api/v1/auth/refresh')
}

export async function logout(): Promise<ApiResponse<{ success: boolean }>> {
  return apiClient.post<{ success: boolean }>('/api/v1/auth/logout')
}

export async function fetchMe(): Promise<ApiResponse<User>> {
  return apiClient.get<User>('/api/v1/users/me')
}

export interface UpdateProfileInput {
  locale?: string
  savedPlayerIds?: SavedPlayerId[]
}

// updateProfile PATCHes the current user's profile; the API returns the full
// updated user, which callers push into the auth store.
export async function updateProfile(input: UpdateProfileInput): Promise<ApiResponse<User>> {
  return apiClient.patch<User>('/api/v1/users/me', input)
}
