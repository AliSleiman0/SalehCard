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
export function deleteEntry(id: string): Promise<ApiResponse<{ deleted: boolean; followUpsDeleted: number }>> {
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
