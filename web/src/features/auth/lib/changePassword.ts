// Client-side rules for the self-service password change, mirroring the API
// (api/internal/modules/user/password.go): at least 8 characters, at most 72
// BYTES (bcrypt's limit — an Arabic letter is 2 bytes in UTF-8), and the
// confirmation must match. The server re-checks everything.

export const MIN_PASSWORD_CHARS = 8
export const MAX_PASSWORD_BYTES = 72

export function utf8Bytes(s: string): number {
  return new TextEncoder().encode(s).length
}

export interface ChangePasswordFields {
  current: string
  next: string
  confirm: string
}

// validateChangePassword returns the i18n key of the first problem, or null.
export function validateChangePassword(f: ChangePasswordFields): string | null {
  if (!f.current) return 'pw_err_current_required'
  if ([...f.next].length < MIN_PASSWORD_CHARS) return 'pw_err_short'
  if (utf8Bytes(f.next) > MAX_PASSWORD_BYTES) return 'pw_err_long'
  if (f.next !== f.confirm) return 'pw_err_mismatch'
  return null
}

// changePasswordErrorKey maps a server error code to its i18n key; unknown
// codes return null so the caller can show the server's own message.
export function changePasswordErrorKey(code: string | undefined): string | null {
  switch (code) {
    case 'WRONG_PASSWORD':
      return 'pw_err_wrong'
    case 'WEAK_PASSWORD':
      return 'pw_err_short'
    case 'PASSWORD_TOO_LONG':
      return 'pw_err_long'
    case 'NO_PASSWORD_SET':
      return 'pw_err_no_password'
    case 'RATE_LIMITED':
      return 'pw_err_rate_limited'
    default:
      return null
  }
}
