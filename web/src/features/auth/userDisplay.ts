import type { User } from '@/types'

// The backend User has no display name yet, so we derive a friendly name and
// initials from the email local-part for the account UI.
export function displayName(user: User | null): string {
  if (!user) return ''
  const local = user.email.split('@')[0]
  return local
    .split(/[._-]+/)
    .filter(Boolean)
    .map((p) => p.charAt(0).toUpperCase() + p.slice(1))
    .join(' ')
}

export function initials(user: User | null): string {
  const name = displayName(user)
  if (!name) return '?'
  const parts = name.split(' ')
  const chars = parts.length > 1 ? parts[0][0] + parts[1][0] : name.slice(0, 2)
  return chars.toUpperCase()
}
