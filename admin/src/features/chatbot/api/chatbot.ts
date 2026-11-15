import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

const BASE = '/api/admin/chatbot'

// --- AI knowledge: the text the assistant answers from ----------------------

/** GET/PUT /chatbot/knowledge. Plain text in any language; paragraphs are
 *  separated by a blank line. androidOnly never reaches the iOS app. */
export interface Knowledge {
  general: string
  androidOnly: string
  updatedBy?: string
  updatedAt?: string
  usedChars: number
  maxChars: number
}

export interface KnowledgeInput {
  general: string
  androidOnly: string
}

export function getKnowledge(): Promise<ApiResponse<Knowledge>> {
  return apiClient.get<Knowledge>(`${BASE}/knowledge`)
}
export function saveKnowledge(input: KnowledgeInput): Promise<ApiResponse<Knowledge>> {
  return apiClient.put<Knowledge>(`${BASE}/knowledge`, input)
}

/** The characters the texts add to the budget, counted like the server
 *  (code points after trimming and CRLF → LF, not UTF-16 units). */
export function knowledgeChars(k: KnowledgeInput): number {
  const n = (s: string) => Array.from(s.replace(/\r\n/g, '\n').trim()).length
  return n(k.general) + n(k.androidOnly)
}

/** The number of paragraphs (blank-line separated) in a text — each is one
 *  piece the assistant can cite. */
export function paragraphCount(s: string): number {
  return s
    .replace(/\r\n/g, '\n')
    .split(/\n\s*\n/)
    .filter((p) => p.trim()).length
}

// --- Conversation log -------------------------------------------------------

export type ChatOutcome = 'answered' | 'unanswered' | 'off_topic' | 'filtered' | 'error'

/** How the model classified the question (the server decides what is shown). */
export type ChatIntent = 'answer' | 'handoff' | 'clarify' | 'smalltalk' | 'out_of_scope'

/** One AI question and what the assistant did with it (kept 30 days). */
export interface ChatLog {
  id: string
  userId: string
  lang: string
  question: string
  answer?: string
  outcome: ChatOutcome
  intent?: ChatIntent
  /** The start of each knowledge paragraph the answer was based on. */
  sources?: string[]
  inputTokens: number
  cachedTokens: number
  outputTokens: number
  latencyMs: number
  flagged: boolean
  flaggedAt?: string
  createdAt: string
}

export type LogFilter = '' | 'unanswered' | 'flagged'

export function listLogs(
  filter: LogFilter,
  page: number
): Promise<ApiResponse<{ items: ChatLog[]; page: number; hasMore: boolean }>> {
  const qs = new URLSearchParams({ page: String(page) })
  if (filter) qs.set('filter', filter)
  return apiClient.get<{ items: ChatLog[]; page: number; hasMore: boolean }>(`${BASE}/logs?${qs}`)
}
