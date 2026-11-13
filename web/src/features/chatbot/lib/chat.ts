import type { ChatEntry, ChatText } from '../api/chatbot'

type Locale = 'en' | 'ar' | 'tr'

/** text picks the locale's string; Turkish (and blanks) fall back en → ar. */
export function text(t: ChatText, locale: Locale): string {
  if (locale === 'ar') return t.ar || t.en
  return t.en || t.ar
}

const bySort = (a: ChatEntry, b: ChatEntry) => a.sortOrder - b.sortOrder

/** topLevel returns the top-level questions in display order. */
export function topLevel(entries: ChatEntry[]): ChatEntry[] {
  return entries.filter((e) => !e.parentId).sort(bySort)
}

/** followUps returns the follow-up questions shown after `parentId`'s answer. */
export function followUps(entries: ChatEntry[], parentId: string): ChatEntry[] {
  return entries.filter((e) => e.parentId === parentId).sort(bySort)
}

/** filterEntries matches a typed query against question text (both languages)
 *  and admin keywords, case-insensitively; an empty query matches nothing. */
export function filterEntries(entries: ChatEntry[], query: string): ChatEntry[] {
  const q = query.trim().toLowerCase()
  if (!q) return []
  return entries
    .filter(
      (e) =>
        e.question.en.toLowerCase().includes(q) ||
        e.question.ar.toLowerCase().includes(q) ||
        (e.keywords ?? []).some((k) => k.includes(q) || q.includes(k)),
    )
    .sort(bySort)
}

/** whatsappLink builds a wa.me link from a free-typed number ("+961 70…"). */
export function whatsappLink(raw: string): string | null {
  const digits = raw.replace(/[^0-9]/g, '').replace(/^00/, '')
  return digits.length >= 6 ? `https://wa.me/${digits}` : null
}
