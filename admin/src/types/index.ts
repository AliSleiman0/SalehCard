// ---- API envelope (mirrors pkg/response on the Go side) ----
export interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
  }
  meta?: PaginationMeta
}

export interface PaginationMeta {
  page: number
  limit: number
  total: number
  pages: number
}

// ---- Shared domain types (match /web and the Go product module) ----
export interface I18nString {
  en: string
  ar: string
  tr: string
}

export type FulfillmentType = 'code' | 'account_credit' | 'transfer'

export interface Variant {
  id: string
  denomination: string
  price: number
  resellerPrice?: number
}

export interface RatingsSummary {
  average: number
  count: number
}

export interface Product {
  id: string
  title: I18nString
  category: string
  images: string[]
  variants: Variant[]
  fulfillmentType: FulfillmentType
  stock: number
  available: boolean
  ratings: RatingsSummary
  createdAt: string
  updatedAt: string
}

// ---- Inventory / codes (admin-only) ----
export type CodeStatus = 'available' | 'delivered' | 'expired'

export interface Code {
  id: string
  productId: string
  code: string
  pin?: string
  status: CodeStatus
  orderId?: string
  deliveredTo?: string
  deliveredAt?: string | null
  batch?: string
  createdAt: string
}

export interface InventoryStats {
  productId: string
  title: string
  category: string
  uploaded: number
  available: number
  delivered: number
  expired: number
  threshold: number
  level: StockLevel
}

export type StockLevel = 'hi' | 'mid' | 'lo'

export interface CodeUploadResult {
  inserted: number
  duplicates: number
  invalid: number
}

// ---- Admin identity ----
export type UserRole = 'customer' | 'reseller' | 'admin'

export interface AdminUser {
  id: string
  name: string
  email: string
  role: UserRole
}

// ---- Locale / theme ----
export type Locale = 'en' | 'ar' | 'tr'
export type Theme = 'light' | 'dark'
