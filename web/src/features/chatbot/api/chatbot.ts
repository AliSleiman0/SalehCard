import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

export interface ChatSupport {
  email?: string
  phone?: string
  whatsapp?: string
}

/** GET /api/v1/chatbot. `ai` = typed questions go to the AI assistant (admin
 *  Settings toggle + a configured provider). The server still sends the
 *  retired quick questions as `entries` for old app builds; the site ignores
 *  them (2026-11-15). */
export interface ChatbotData {
  support: ChatSupport
  ai?: boolean
}

/** The assistant's reply. answered=false → `answer` is the server's polite
 *  "can't help, contact support" text. */
export interface AskResult {
  logId?: string
  answer: string
  answered: boolean
  showSupport: boolean
}

export interface AskTurn {
  role: 'user' | 'assistant'
  content: string
}

export function fetchChatbot(): Promise<ApiResponse<ChatbotData>> {
  return apiClient.get<ChatbotData>('/api/v1/chatbot')
}

/** POST /api/v1/chatbot/ask (signed-in). The website sells everything the
 *  Android app does, so it is answered from the full knowledge (`web`). */
export function askChatbot(
  question: string,
  lang: 'en' | 'ar',
  history: AskTurn[],
): Promise<ApiResponse<AskResult>> {
  return apiClient.post<AskResult>('/api/v1/chatbot/ask', {
    question,
    lang,
    platform: 'web',
    history,
  })
}

/** POST /api/v1/chatbot/ask/{logId}/report — flags an AI answer for review. */
export function reportAnswer(logId: string): Promise<ApiResponse<{ reported: boolean }>> {
  return apiClient.post<{ reported: boolean }>(`/api/v1/chatbot/ask/${logId}/report`)
}
