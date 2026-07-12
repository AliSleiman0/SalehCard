// Lebanese phone helpers (web port of app/lib/.../phone_format.dart). The UI
// fixes the country to +961; these only need to produce a sane E.164 candidate —
// the backend re-validates.

export function phoneDigits(raw: string): string {
  return raw.replace(/\D/g, '')
}

// toE164Lebanon strips non-digits and a single leading zero, then prepends +961.
export function toE164Lebanon(raw: string): string {
  let digits = phoneDigits(raw)
  if (digits.startsWith('0')) digits = digits.slice(1)
  return `+961${digits}`
}

// isPlausibleLebanonPhone mirrors the app's ">= 7 digits" gate (after dropping a
// leading zero) so the UI can validate before hitting the API.
export function isPlausibleLebanonPhone(raw: string): boolean {
  const digits = phoneDigits(raw).replace(/^0/, '')
  return digits.length >= 7
}
