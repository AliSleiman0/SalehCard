import { apiClient } from '@/lib/api-client'
import type { ApiResponse, I18nString } from '@/types'

export interface ApiOfferProduct {
  id: string
  title: I18nString
  images: string[]
  category: string
  fromPrice: number
  inStock: boolean
}

export interface ApiOffer {
  id: string
  productId: string
  discountType: 'percent' | 'fixed'
  discountValue: number
  endsAt?: string
  product: ApiOfferProduct
  originalFromPrice: number
  offerFromPrice: number
}

export async function fetchOffers(): Promise<ApiResponse<ApiOffer[]>> {
  return apiClient.get<ApiOffer[]>('/api/v1/offers')
}
