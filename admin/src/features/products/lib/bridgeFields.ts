import type { InputField } from '@/types'

// reconcileBridgePhoneField guarantees a bridge (mobile-recharge) product carries
// exactly one input field keyed `phone`, positioned first.
//
// The key `phone` is load-bearing: the Flutter checkout validates a Lebanese
// mobile only on the field whose key === 'phone', and the server's bridgePhone
// fallback matches only that key. First-position matters because the app derives
// the recharge number from the FIRST non-empty field (so a `qty` field must not
// precede it). Legacy imports carry the phone under another key (e.g. `field_1`),
// so we reuse the first text field — renaming its key to `phone` — rather than
// appending a duplicate.
export function reconcileBridgePhoneField(fields: InputField[]): InputField[] {
  let idx = fields.findIndex((f) => f.key === 'phone')
  if (idx === -1) idx = fields.findIndex((f) => f.type === 'text')
  const phone: InputField =
    idx === -1
      ? { key: 'phone', label: { en: 'Mobile number', ar: 'رقم الهاتف' }, type: 'text', sensitive: false }
      : { ...fields[idx], key: 'phone', type: 'text' }
  const rest = fields.filter((f, i) => i !== idx && f.key !== 'phone')
  return [phone, ...rest]
}
