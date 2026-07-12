export interface I18nString {
  en: string
  ar: string
  tr: string
}

export interface Variant {
  id: string
  denomination: string
  price: number
  offerPrice?: number
  resellerPrice?: number
}

export interface InputFieldConstraints {
  min?: number
  max?: number
  options?: string[]
}

// Per-product checkout field spec (Product.inputFields). Labels are i18n maps
// that may omit locales; `sensitive` values are masked client-side and never
// persisted server-side.
export interface InputField {
  key: string
  label: Partial<I18nString>
  type: 'text' | 'amount' | 'quantity' | 'select'
  constraints?: InputFieldConstraints
  sensitive: boolean
}

// Game-ID verification hook (Product.verification): `app` is the game slug
// for POST /products/{id}/verify-account.
export interface Verification {
  provider: number
  app: string
}

export interface OfferInfo {
  discountType: 'percent' | 'fixed'
  discountValue: number
  originalFromPrice: number
  offerFromPrice: number
  endsAt?: string
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
  // Top-level domain (games, app_topups, …) the product's category resolves to.
  // Optional: only migrated products carry it. Used for browse-by-domain.
  rootDomain?: string
  images: string[]
  thumbnail: string
  variants: Variant[]
  fulfillmentType: FulfillmentType
  stock: number
  available: boolean
  ratings: RatingsSummary
  inputFields?: InputField[]
  verification?: Verification
  offer?: OfferInfo
  createdAt: string
  updatedAt: string
}

// Category mirrors the API's GET /api/v1/categories item (the migrated taxonomy).
// productCount is only present for root domains (depth 0) when withCounts is set.
export interface Category {
  id: string
  legacyId: number
  parentLegacyId?: number
  slug: string
  name: I18nString
  image?: string
  sortOrder: number
  rootDomain: string
  depth: number
  visible: boolean
  productCount?: number
}

export type UserRole = 'customer' | 'reseller' | 'admin'

export interface SavedPlayerId {
  label: string
  value: string
}

export interface User {
  id: string
  name: string
  email: string
  phone?: string
  googleId?: string
  role: UserRole
  resellerTier?: string
  locale: string
  savedPlayerIds: SavedPlayerId[]
  walletBalance: number
  loyaltyPoints: number
  createdAt: string
  updatedAt: string
}

export type OrderStatus = 'pending' | 'processing' | 'completed' | 'failed' | 'refunded'

export type PaymentMethod = 'wallet' | 'card' | 'usdt'

export interface Recipient {
  name: string
  country: string
  detail: string
}

export interface OrderItem {
  productId: string
  variantId: string
  title: I18nString
  denomination: string
  category: string
  qty: number
  price: number
  fulfillmentType: FulfillmentType
  playerId?: string
  recipient?: Recipient
}

export interface TimelineEvent {
  status: string
  note: string
  at: string
}

export interface Fulfillment {
  deliveredCode?: string
  creditedToId?: string
  transferRef?: string
  statusTimeline: TimelineEvent[]
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
  fulfillment: Fulfillment
  createdAt: string
  updatedAt: string
}

// PlaceOrderInput is the request body for POST /api/v1/orders. The client sends
// only product/variant/qty (+ fulfillment target); the server re-prices.
export interface PlaceOrderItemInput {
  productId: string
  variantId: string
  qty: number
  playerId?: string
  recipient?: Recipient
  // Structured per-field capture for account_credit lines; labels are
  // resolved server-side from the product's inputFields spec.
  fields?: { key: string; value: string }[]
}

export interface PlaceOrderInput {
  items: PlaceOrderItemInput[]
  currency: string
  paymentMethod: PaymentMethod
  usdtNetwork?: string
  promoCode?: string
}

export type PaymentIntentStatus = 'pending' | 'confirming' | 'confirmed' | 'expired'

export interface PaymentIntent {
  id: string
  purpose: 'order' | 'topup'
  orderId?: string
  network: string
  address: string
  amountUsd: number
  receivedUsd: number
  status: PaymentIntentStatus
  txHash?: string
  createdAt: string
  expiresAt: string
}

export interface PaymentConfig {
  usdtEnabled: boolean
  network: string
  networks: string[]
  expiryMinutes: number
}

// POST /orders response: for usdt orders the intent rides FLAT next to the
// order fields (Go embeds the Order struct in placeOrderResponse).
export type PlacedOrder = Order & { paymentIntent?: PaymentIntent }

export type WalletTxType = 'topup' | 'purchase' | 'refund' | 'adjustment'

export interface WalletTransaction {
  id: string
  userId: string
  type: WalletTxType
  amount: number
  balanceAfter: number
  method: string
  ref: string
  createdAt: string
}

export interface WalletView {
  balance: number
  transactions: WalletTransaction[]
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

export interface AuthResponse {
  accessToken: string
  user: User
  // Admin-only: set when the account must complete an SMS second factor. The
  // storefront never implements the challenge — it surfaces a "use the admin
  // console" error instead (see assertCustomerSession).
  twoFactorRequired?: boolean
}
