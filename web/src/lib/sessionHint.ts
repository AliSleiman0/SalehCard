// A non-sensitive hint that a session likely exists. The httpOnly refresh
// cookie remains the source of truth — JS can't read it — but the app uses this
// flag to decide whether the on-load /api/v1/auth/refresh probe is worth making.
// Without it, every anonymous/first-time visitor triggers a guaranteed-401
// refresh on boot (console noise + a wasted request). Set on a successful
// login/register/refresh; cleared on logout, failed refresh, or a stale hint.
const KEY = 'salehcard.session'

export function markSessionHint(): void {
  try {
    localStorage.setItem(KEY, '1')
  } catch {
    /* storage unavailable (private mode / disabled) — degrade to always-probe */
  }
}

export function clearSessionHint(): void {
  try {
    localStorage.removeItem(KEY)
  } catch {
    /* ignore */
  }
}

export function hasSessionHint(): boolean {
  try {
    return localStorage.getItem(KEY) === '1'
  } catch {
    return false
  }
}
