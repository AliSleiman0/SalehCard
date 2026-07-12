import type { User } from '@/types'

// displayName prefers the real account name, then a humanized email local-part,
// then the phone — so phone-only accounts don't render blank.
export function displayName(user: User | null): string {
  if (!user) return ''
  if (user.name?.trim()) return user.name.trim()
  if (user.email) {
    return user.email
      .split('@')[0]
      .split(/[._-]+/)
      .filter(Boolean)
      .map((p) => p.charAt(0).toUpperCase() + p.slice(1))
      .join(' ')
  }
  return user.phone ?? ''
}

export function initials(user: User | null): string {
  const name = displayName(user)
  if (!name) return '?'
  // Phone-only fallback (starts with +/digit) has no letters to initial.
  if (/^[+\d]/.test(name)) return '#'
  const parts = name.split(' ')
  const chars = parts.length > 1 ? parts[0][0] + parts[1][0] : name.slice(0, 2)
  return chars.toUpperCase()
}
