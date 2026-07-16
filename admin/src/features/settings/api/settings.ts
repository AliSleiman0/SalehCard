import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

/** Read-only integration status (provider name + configured flag, no secrets). */
export interface Integration {
  provider: string
  configured: boolean
}

/** One admin-defined currency rate line shown to customers (both free text). */
export interface ExchangeRate {
  label: string
  value: string
}

/** Platform settings as returned by GET /api/admin/settings (matches the Go model). */
export interface AdminSettings {
  storeName: string
  supportEmail: string
  supportPhone: string
  defaultLanguage: string
  defaultCurrency: string
  lowStockThreshold: number
  maintenanceMode: boolean
  /** Loyalty program: whether completed orders earn points. */
  loyaltyEnabled: boolean
  /** Spend (order-total dollars) that mints one point: points = floor(total / rate). */
  loyaltyEarnUsdPerPoint: number
  /** Require an SMS second factor for admin logins (off by default). */
  adminSmsTwoFactorEnabled: boolean
  /** Currency rates surfaced on the customer top-up screen. */
  exchangeRates: ExchangeRate[]
  updatedAt: string
  updatedBy?: string
  integrations?: Record<string, Integration>
}

/** PUT payload — every field optional; omitted fields are left unchanged. */
export interface SettingsInput {
  storeName?: string
  supportEmail?: string
  supportPhone?: string
  defaultLanguage?: string
  defaultCurrency?: string
  lowStockThreshold?: number
  maintenanceMode?: boolean
  loyaltyEnabled?: boolean
  loyaltyEarnUsdPerPoint?: number
  adminSmsTwoFactorEnabled?: boolean
  exchangeRates?: ExchangeRate[]
}

const ADMIN = '/api/admin/settings'

export function getSettings(): Promise<ApiResponse<AdminSettings>> {
  return apiClient.get<AdminSettings>(ADMIN)
}

export function updateSettings(input: SettingsInput): Promise<ApiResponse<AdminSettings>> {
  return apiClient.put<AdminSettings>(ADMIN, input)
}
