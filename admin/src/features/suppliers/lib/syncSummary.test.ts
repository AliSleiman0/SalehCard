import { describe, expect, it } from 'vitest'
import { syncSummary } from './syncSummary'
import type { SyncReport } from '../api/suppliers'

const base: SyncReport = {
  dryRun: false,
  startedAt: '2026-11-15T18:00:00Z',
  finishedAt: '2026-11-15T18:00:02Z',
  suppliers: [{ id: 10, name: 'jentel', status: 'ok' }],
  checked: 12,
  snapshots: 3,
  costChanges: 0,
  paused: 0,
  recovered: 0,
  promoted: 0,
  hidden: 0,
  shown: 0,
  conflicts: 0,
  alerts: 0,
  errors: 0,
}

describe('syncSummary', () => {
  it('says "no changes" for a quiet pass', () => {
    expect(syncSummary(base)).toBe('12 products checked · no changes')
  })

  it('lists what changed, singular and plural', () => {
    expect(syncSummary({ ...base, costChanges: 2, paused: 1, promoted: 1, hidden: 1 })).toBe(
      '12 products checked · 2 cost changes · 1 supplier paused (loss) · 1 product switched supplier · 1 taken off sale',
    )
  })

  it('flags warning-only mode and unreachable suppliers', () => {
    const r = { ...base, dryRun: true, hidden: 1, suppliers: [{ id: 11, name: 'speedcard', status: 'unavailable' as const }] }
    expect(syncSummary(r)).toBe("warnings only, nothing changed — 12 products checked · 1 would go off sale · couldn't reach speedcard")
  })
})
