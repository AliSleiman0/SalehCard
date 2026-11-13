import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

export interface ChatText {
  en: string
  ar: string
}

/** One predefined question + answer; parentId set = a follow-up question. */
export interface ChatEntry {
  id: string
  parentId?: string
  question: ChatText
  answer: ChatText
  keywords?: string[]
  sortOrder: number
}

export interface ChatSupport {
  email?: string
  phone?: string
  whatsapp?: string
}

export interface ChatbotData {
  entries: ChatEntry[]
  support: ChatSupport
}

/** GET /api/v1/chatbot — enabled questions + the support contact (public). */
export function fetchChatbot(): Promise<ApiResponse<ChatbotData>> {
  return apiClient.get<ChatbotData>('/api/v1/chatbot')
}
