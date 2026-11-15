import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

/** Bilingual text (the storefront speaks English and Arabic). */
export interface ChatText {
  en: string
  ar: string
}

/** One predefined question + answer. parentId set = a follow-up question. */
export interface ChatEntry {
  id: string
  parentId?: string
  question: ChatText
  answer: ChatText
  keywords?: string[]
  sortOrder: number
  enabled: boolean
  createdBy?: string
  createdAt: string
  updatedAt: string
}

export interface ChatEntryInput {
  parentId: string
  question: ChatText
  answer: ChatText
  keywords: string[]
  sortOrder: number
  enabled: boolean
}

const BASE = '/api/admin/chatbot'

export function listEntries(): Promise<ApiResponse<ChatEntry[]>> {
  return apiClient.get<ChatEntry[]>(BASE)
}
export function createEntry(input: ChatEntryInput): Promise<ApiResponse<ChatEntry>> {
  return apiClient.post<ChatEntry>(BASE, input)
}
export function updateEntry(id: string, input: ChatEntryInput): Promise<ApiResponse<ChatEntry>> {
  return apiClient.put<ChatEntry>(`${BASE}/${id}`, input)
}
export function deleteEntry(
  id: string
): Promise<ApiResponse<{ deleted: boolean; followUpsDeleted: number }>> {
  return apiClient.delete<{ deleted: boolean; followUpsDeleted: number }>(`${BASE}/${id}`)
}

/** groupEntries orders top-level questions by sortOrder, each followed by its
 *  follow-ups (also by sortOrder). Orphans (parent missing) are listed last. */
export function groupEntries(all: ChatEntry[]): { top: ChatEntry; children: ChatEntry[] }[] {
  const bySort = (a: ChatEntry, b: ChatEntry) =>
    a.sortOrder - b.sortOrder || a.createdAt.localeCompare(b.createdAt)
  const tops = all.filter((e) => !e.parentId).sort(bySort)
  const ids = new Set(tops.map((e) => e.id))
  const groups = tops.map((top) => ({
    top,
    children: all.filter((e) => e.parentId === top.id).sort(bySort),
  }))
  for (const orphan of all.filter((e) => e.parentId && !ids.has(e.parentId))) {
    groups.push({ top: orphan, children: [] })
  }
  return groups
}

// --- AI assistant: knowledge base + conversation log ------------------------

/** One knowledge-base article the AI assistant answers from. */
export interface KBDoc {
  id: string
  slug?: string
  title: ChatText
  body: ChatText
  tags?: string[]
  sortOrder: number
  enabled: boolean
  createdBy?: string
  updatedBy?: string
  createdAt: string
  updatedAt: string
}

export interface KBDocInput {
  slug: string
  title: ChatText
  body: ChatText
  tags: string[]
  sortOrder: number
  enabled: boolean
}

/** GET /chatbot/kb: every article plus the size budget (enabled articles only count). */
export interface KBList {
  docs: KBDoc[]
  usedChars: number
  maxChars: number
  maxDocs: number
}

/** Tag that keeps an article away from the iOS app (App Review). */
export const ANDROID_ONLY_TAG = 'android-only'

export function listKB(): Promise<ApiResponse<KBList>> {
  return apiClient.get<KBList>(`${BASE}/kb`)
}
export function createKB(input: KBDocInput): Promise<ApiResponse<KBDoc>> {
  return apiClient.post<KBDoc>(`${BASE}/kb`, input)
}
export function updateKB(id: string, input: KBDocInput): Promise<ApiResponse<KBDoc>> {
  return apiClient.put<KBDoc>(`${BASE}/kb/${id}`, input)
}
export function deleteKB(id: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiClient.delete<{ deleted: boolean }>(`${BASE}/kb/${id}`)
}

/** The characters an article adds to the knowledge-base budget. */
export function kbDocChars(d: { title: ChatText; body: ChatText }): number {
  // Count code points like the server (utf8.RuneCountInString), not UTF-16 units.
  const n = (s: string) => Array.from(s).length
  return n(d.title.en) + n(d.title.ar) + n(d.body.en) + n(d.body.ar)
}

export type ChatOutcome = 'answered' | 'unanswered' | 'filtered' | 'error'

/** One AI question and what the assistant did with it (kept 30 days). */
export interface ChatLog {
  id: string
  userId: string
  lang: string
  question: string
  answer?: string
  outcome: ChatOutcome
  sourceIds?: string[]
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
