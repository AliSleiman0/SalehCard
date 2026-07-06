import { apiClient } from '@/lib/api-client'
import type {
  ApiResponse,
  PaginationMeta,
  Product,
  FulfillmentType,
  FulfillmentMode,
  BridgeSpec,
  InputField,
  Verification,
} from '@/types'

export interface ProductListParams {
  page?: number
  limit?: number
  category?: string
  fulfillmentType?: FulfillmentType
  status?: 'active' | 'draft' | 'out'
  available?: boolean
  search?: string
}

export interface ProductInput {
  title: { en: string; ar: string; tr: string }
  description: { en: string; ar: string; tr: string }
  category: string
  images: string[]
  thumbnail?: string
  variants: { denomination: string; price: number; resellerPrice?: number; faceValue?: number }[]
  fulfillmentType: FulfillmentType
  // Execution mode. Sent as 'bridge_device' for mobile-recharge products (and
  // 'manual_operator' to turn a former bridge product back into plain credit);
  // omitted otherwise so the backend derives it.
  fulfillmentMode?: FulfillmentMode
  // Bridge recharge config. A non-empty `provider` sets it; `{provider:'',...}`
  // clears it. Omitted = leave unchanged.
  bridge?: BridgeSpec | { provider: ''; method: '' }
  stock: number
  available: boolean
  inputFields?: InputField[]
  // Purchase-time ID verification. A non-empty `app` enables it; `{provider:0,
  // app:''}` clears it (disable). Omitted = leave unchanged.
  verification?: Verification
}

const ADMIN = '/api/admin/products'

export async function listProducts(
  params: ProductListParams
): Promise<ApiResponse<Product[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.category) q.set('category', params.category)
  if (params.fulfillmentType) q.set('fulfillmentType', params.fulfillmentType)
  if (params.status) q.set('status', params.status)
  if (params.available !== undefined) q.set('available', String(params.available))
  if (params.search) q.set('search', params.search)
  const qs = q.toString()
  return apiClient.get<Product[]>(qs ? `${ADMIN}?${qs}` : ADMIN)
}

export function getProduct(id: string): Promise<ApiResponse<Product>> {
  return apiClient.get<Product>(`${ADMIN}/${id}`)
}

export function createProduct(input: ProductInput): Promise<ApiResponse<Product>> {
  return apiClient.post<Product>(ADMIN, input)
}

export function updateProduct(id: string, input: Partial<ProductInput>): Promise<ApiResponse<Product>> {
  return apiClient.put<Product>(`${ADMIN}/${id}`, input)
}

export function deleteProduct(id: string): Promise<ApiResponse<unknown>> {
  return apiClient.delete(`${ADMIN}/${id}`)
}

export interface UploadImageResult {
  imageUrl: string
  thumbnailUrl: string
}

// Product-agnostic: uploads first, then the returned URLs go into the normal
// create/update ProductInput (there's no product id yet on create).
export function uploadProductImage(file: File): Promise<ApiResponse<UploadImageResult>> {
  const form = new FormData()
  form.set('image', file)
  return apiClient.upload<UploadImageResult>(`${ADMIN}/images`, form)
}

export type BulkAction = 'activate' | 'deactivate' | 'delete'

export function bulkProductAction(ids: string[], action: BulkAction): Promise<ApiResponse<{ modified: number }>> {
  return apiClient.post<{ modified: number }>(`${ADMIN}/bulk`, { ids, action })
}
