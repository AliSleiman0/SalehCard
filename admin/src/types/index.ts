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
  /** Amount the bridge transfers on a transfer_credit recharge (bridge products). */
  faceValue?: number
}

/** How an order is executed (orthogonal to fulfillmentType). bridge_device and
 *  api are surfaced in the admin editor; the rest are backend-derived. */
export type FulfillmentMode = 'api' | 'manual_operator' | 'inventory' | 'bridge_device'

/** Lebanese mobile-recharge fulfillment config (bridge_device products). */
export interface BridgeSpec {
  provider: 'touch' | 'alfa'
  method: 'transfer_credit' | 'recharge_line'
}

export interface RatingsSummary {
  average: number
  count: number
}

export type InputFieldType = 'text' | 'amount' | 'quantity' | 'select'

// Customer-input field spec on a product. The label is 2-locale (en/ar) — this
// mirrors the Go `InputField`/`I18nLabel`. Admins edit key, type, labels, and
// constraints (min/max for quantity/amount, options for select) in the product
// editor; only `legacyName` is preserved verbatim.
export interface InputField {
  key: string
  label: { en: string; ar: string }
  legacyName?: string
  type: InputFieldType
  sensitive?: boolean
  constraints?: { min?: number; max?: number; options?: string[] }
}

// Purchase-time account-ID verification (check_name). `app` is the provider's
// game slug (e.g. "pubgm-global"); `provider` is the numeric verification-provider
// id. Mirrors the Go `Verification`. Absent = the product needs no verification.
export interface Verification {
  provider: number
  app: string
}

export interface Product {
  id: string
  title: I18nString
  description: I18nString
  category: string
  /** Managed taxonomy node this product is assigned to (absent = unassigned). */
  categoryId?: string
  /** Display-image URLs. Legacy-import products can return null (no images). */
  images: string[] | null
  thumbnail?: string
  variants: Variant[]
  fulfillmentType: FulfillmentType
  fulfillmentMode?: FulfillmentMode
  /** Upstream supplier registry id (api-mode products). */
  fulfillmentProvider?: number
  /** The supplier's own product id (api-mode products). */
  upstreamProductId?: string
  bridge?: BridgeSpec
  stock: number
  available: boolean
  ratings: RatingsSummary
  inputFields?: InputField[]
  verification?: Verification
  /** Economics. Only `cost` (admin-set unit cost) is editable in the console;
   *  retail/currency come from the legacy catalog importer. Absent = no cost set. */
  pricing?: { cost?: number; retail?: number; currency?: string }
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
  /** Unit cost (Product.pricing.cost) and Available×cost. Absent = no cost set → show "—". */
  unitPrice?: number
  totalValue?: number
}

export type StockLevel = 'hi' | 'mid' | 'lo'

export interface CodeUploadResult {
  inserted: number
  duplicates: number
  invalid: number
}

// ---- Saved player IDs (match the Go user module's SavedPlayerID) ----
// Each saved game/account ID is a labelled pair. Legacy docs that stored a bare
// string are normalised to `{ label, value }` (label == value) by the API's
// tolerant BSON decode, so the client always receives objects.
export interface SavedPlayerId {
  label: string
  value: string
}

// ---- Admin identity ----
export type UserRole = 'customer' | 'reseller' | 'admin'

export interface AdminUser {
  id: string
  name: string
  email: string
  role: UserRole
  // RBAC permission set resolved by the backend at login/refresh:
  // "<domain>.view" / "<domain>.manage" entries, or the single wildcard "*"
  // for a super admin (an admin with no custom role assigned).
  permissions: string[]
}

// ---- Locale / theme ----
export type Locale = 'en' | 'ar' | 'tr'
export type Theme = 'light' | 'dark'
