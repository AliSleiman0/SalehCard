import i18n from '@/i18n/config'

/** Names in the admin's quote marks, comma-separated: `"a", "b"` in English,
 *  `«a»، «b»` in Arabic. */
export function quoted(items: string[]): string {
  return items.map((v) => i18n.t('prod_q', { v })).join(i18n.t('prod_list_sep'))
}
