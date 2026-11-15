import type { TFunction } from 'i18next'
import type { SupplierAlert } from '../api/suppliers'

/** The alert's message in the admin's language. The server writes the English
 *  sentence (`message`) plus its values (`args`); for any other language the
 *  sentence is rebuilt from the translated template. Alerts without args (raised
 *  before args existed) and English stay on the server text. */
export function alertText(a: SupplierAlert, t: TFunction, lang: string): string {
  const g = a.args
  if (!g || lang.startsWith('en')) return a.message
  const w = !!a.dryRun // warning-only: says what WOULD happen
  const per = !g.per || g.per === '1' ? t('sup_per_unit') : t('sup_per_n', { n: g.per })
  const v = { ...g, per }
  switch (a.kind) {
    case 'cost_changed':
      return t('sup_msg_cost_changed', v)
    case 'loss_paused': {
      const outcome = g.outcome === 'switched' || g.outcome === 'offsale' ? g.outcome : 'backup'
      const parts = [
        t(w ? 'sup_msg_loss_would' : 'sup_msg_loss', v),
        t(`sup_msg_loss_${outcome}${w ? '_would' : ''}`, v),
      ]
      if (w) parts.push(t('sup_msg_nothing_yet'))
      return parts.join(' ')
    }
    case 'loss_recovered':
      return t(w ? 'sup_msg_recovered_would' : 'sup_msg_recovered', v)
    case 'product_hidden': {
      const reason = t(g.reason === 'loss' ? 'sup_reason_loss' : 'sup_reason_unavailable')
      return t(w ? 'sup_msg_hidden_would' : 'sup_msg_hidden', { ...v, reason })
    }
    case 'product_shown':
      return t(w ? 'sup_msg_shown_would' : 'sup_msg_shown', v)
    case 'fx_missing':
      return t('sup_msg_fx_missing', { ...v, n: g.count })
    case 'fallback_exhausted':
      return t('sup_msg_fallback_exhausted', v)
    default:
      return a.message
  }
}
