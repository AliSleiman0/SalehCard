import i18n from '@/i18n/config'
import { ApiError } from '@/lib/api-client'

/** The message to show for a failed request, in the admin's language.
 *  English keeps the server's own message (it is often more specific). Other
 *  languages show the translated `err_<CODE>` text when one exists, else the
 *  server's message. Anything that is not an API error shows `fallbackKey`. */
export function apiErrorText(e: unknown, fallbackKey: string): string {
  if (!(e instanceof ApiError)) return i18n.t(fallbackKey)
  const lang = i18n.language || 'en'
  const key = `err_${e.code}`
  if (!lang.startsWith('en') && e.code && i18n.exists(key)) return i18n.t(key)
  return e.message || i18n.t(fallbackKey)
}
