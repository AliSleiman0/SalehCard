import { describe, it, expect } from 'vitest'
import { baselineFrom, newPopups } from './popups'
import type { AppNotification } from '../types'

const n = (id: string, kind: string, at: string, readAt?: string): AppNotification => ({
  id,
  kind,
  title: id,
  body: '',
  createdAt: at,
  readAt,
})

describe('newPopups', () => {
  const since = Date.parse('2026-11-12T10:00:00Z')
  it('pops only unread top-up notes newer than the baseline, oldest first', () => {
    const list = [
      n('b', 'topup_rejected', '2026-11-12T10:05:00Z'),
      n('a', 'topup_approved', '2026-11-12T10:01:00Z'),
      n('old', 'topup_approved', '2026-11-12T09:59:00Z'),
      n('read', 'topup_approved', '2026-11-12T10:02:00Z', '2026-11-12T10:03:00Z'),
      n('order', 'order_completed', '2026-11-12T10:04:00Z'),
    ]
    expect(newPopups(list, since).map((x) => x.id)).toEqual(['a', 'b'])
  })
})

describe('baselineFrom', () => {
  it('is the newest server timestamp, or 0 for an empty inbox', () => {
    expect(baselineFrom([])).toBe(0)
    const list = [n('a', 'order_completed', '2026-11-12T10:01:00Z'), n('b', 'topup_approved', '2026-11-12T10:05:00Z')]
    expect(baselineFrom(list)).toBe(Date.parse('2026-11-12T10:05:00Z'))
  })
})
