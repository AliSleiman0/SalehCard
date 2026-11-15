import i18n from '@/i18n/config'
import { ApiError } from '@/lib/api-client'
import { apiErrorText } from '@/lib/apiErrorText'

/** apiErrorText, plus translations for fixed server sentences that arrive under
 *  a generic code (BAD_REQUEST, CONFLICT) and so have no `err_<CODE>` key of
 *  their own. `messages` maps the server's English sentence to a key. English
 *  keeps the server's message. */
export function serverErrorText(e: unknown, fallbackKey: string, messages: Record<string, string>): string {
  const key = e instanceof ApiError ? messages[e.message] : undefined
  if (key && !(i18n.language || 'en').startsWith('en')) return i18n.t(key)
  return apiErrorText(e, fallbackKey)
}
