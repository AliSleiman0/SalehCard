import { describe, it, expect } from 'vitest'
import { changePasswordErrorKey, utf8Bytes, validateChangePassword } from './changePassword'

const ok = { current: 'password123', next: 'newpass123', confirm: 'newpass123' }

describe('validateChangePassword', () => {
  it('accepts a valid change', () => {
    expect(validateChangePassword(ok)).toBeNull()
  })

  it('requires the current password', () => {
    expect(validateChangePassword({ ...ok, current: '' })).toBe('pw_err_current_required')
  })

  it('blocks passwords under 8 characters', () => {
    expect(validateChangePassword({ ...ok, next: 'short1', confirm: 'short1' })).toBe('pw_err_short')
    expect(validateChangePassword({ ...ok, next: '12345678', confirm: '12345678' })).toBeNull()
  })

  it('counts the 72 limit in UTF-8 bytes, not characters', () => {
    const ascii72 = 'a'.repeat(72)
    expect(validateChangePassword({ ...ok, next: ascii72, confirm: ascii72 })).toBeNull()
    const ascii73 = 'a'.repeat(73)
    expect(validateChangePassword({ ...ok, next: ascii73, confirm: ascii73 })).toBe('pw_err_long')
    // 37 Arabic letters = 74 bytes: over the limit despite being 37 characters.
    const arabic = 'ب'.repeat(37)
    expect(utf8Bytes(arabic)).toBe(74)
    expect(validateChangePassword({ ...ok, next: arabic, confirm: arabic })).toBe('pw_err_long')
    const arabic36 = 'ب'.repeat(36)
    expect(validateChangePassword({ ...ok, next: arabic36, confirm: arabic36 })).toBeNull()
  })

  it('requires the confirmation to match', () => {
    expect(validateChangePassword({ ...ok, confirm: 'newpass124' })).toBe('pw_err_mismatch')
  })
})

describe('changePasswordErrorKey', () => {
  it('maps every server code to its own message', () => {
    expect(changePasswordErrorKey('WRONG_PASSWORD')).toBe('pw_err_wrong')
    expect(changePasswordErrorKey('WEAK_PASSWORD')).toBe('pw_err_short')
    expect(changePasswordErrorKey('PASSWORD_TOO_LONG')).toBe('pw_err_long')
    expect(changePasswordErrorKey('NO_PASSWORD_SET')).toBe('pw_err_no_password')
    expect(changePasswordErrorKey('RATE_LIMITED')).toBe('pw_err_rate_limited')
  })

  it('returns null for unknown codes so the server message is shown', () => {
    expect(changePasswordErrorKey('NETWORK_ERROR')).toBeNull()
    expect(changePasswordErrorKey(undefined)).toBeNull()
  })
})
