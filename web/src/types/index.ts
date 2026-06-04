export interface I18nString {
  en: string
  ar: string
  tr: string
}

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

export type FulfillmentType = 'code' | 'account_credit' | 'transfer'

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

export type UserRole = 'customer' | 'reseller' | 'admin'

export interface User {
  id: string
  email: string
  googleId?: string
  role: UserRole
  locale: string
  savedPlayerIds: string[]
  walletBalance: number
  loyaltyPoints: number
  createdAt: string
  updatedAt: string
}

export type OrderStatus = 'pending' | 'processing' | 'completed' | 'failed' | 'refunded'

export type PaymentMethod = 'wallet' | 'card' | 'usdt'

export interface OrderItem {
  productId: string
  variantId: string
  qty: number
  price: number
  fulfillmentType: FulfillmentType
}

export interface Order {
  id: string
  userId: string
  items: OrderItem[]
  subtotal: number
  total: number
  currency: string
  paymentMethod: PaymentMethod
  status: OrderStatus
  createdAt: string
}

export interface PaginationMeta {
  page: number
  limit: number
  total: number
  pages: number
}

export interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
  }
  meta?: PaginationMeta
}
