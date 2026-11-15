import type { AskTurn } from '../api/chatbot'

/** whatsappLink builds a wa.me link from a free-typed number ("+961 70…"). */
export function whatsappLink(raw: string): string | null {
  const digits = raw.replace(/[^0-9]/g, '').replace(/^00/, '')
  return digits.length >= 6 ? `https://wa.me/${digits}` : null
}

/** askLang maps the site locale to the assistant's languages (Turkish → English). */
export function askLang(locale: string): 'en' | 'ar' {
  return locale === 'ar' ? 'ar' : 'en'
}

/** askHistory keeps the last `max` turns (oldest first) to send as context. */
export function askHistory(msgs: { from: 'bot' | 'user'; text: string }[], max = 6): AskTurn[] {
  return msgs.slice(-max).map((m) => ({
    role: m.from === 'user' ? 'user' : 'assistant',
    content: m.text,
  }))
}

/** Why an AI question got no answer: the customer's limit (429
 *  CHATBOT_AI_LIMIT), or anything else (off, outage, network) = unavailable. */
export function askFailure(code: string | undefined): 'limited' | 'unavailable' {
  return code === 'CHATBOT_AI_LIMIT' ? 'limited' : 'unavailable'
}
