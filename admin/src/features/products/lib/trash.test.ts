import { describe, it, expect } from 'vitest'
import type { Product } from '@/types'
import { bulkActionsFor, bulkConfirmText, canSelectAllMatching } from './trash'
import { productStatus } from './view'

describe('bulkActionsFor', () => {
  it('normal view: delete means move to trash; no restore or purge', () => {
    expect(bulkActionsFor(false, true, false)).toEqual(['activate', 'deactivate', 'assign-category', 'delete'])
  })

  it('trash view: restore + move to category; purge only for super admins', () => {
    expect(bulkActionsFor(true, false, false)).toEqual(['restore', 'assign-category'])
    expect(bulkActionsFor(true, true, false)).toEqual(['restore', 'assign-category', 'purge'])
  })

  it('never offers purge for "all matching" (explicit ids only)', () => {
    expect(bulkActionsFor(true, true, true)).not.toContain('purge')
  })
})

describe('canSelectAllMatching', () => {
  it('needs the whole page selected and more matches than shown', () => {
    expect(canSelectAllMatching(true, 20, 120)).toBe(true)
    expect(canSelectAllMatching(false, 20, 120)).toBe(false)
    expect(canSelectAllMatching(true, 20, 20)).toBe(false)
  })
})

describe('bulkConfirmText', () => {
  it('trash confirm says it is reversible, not "cannot be undone"', () => {
    const msg = bulkConfirmText('delete', 3, false)!
    expect(msg).toMatch(/trash/i)
    expect(msg).toMatch(/restore/i)
    expect(msg).not.toMatch(/cannot be undone/i)
  })

  it('purge confirm warns it is permanent', () => {
    expect(bulkConfirmText('purge', 2, false)).toMatch(/cannot be undone/i)
  })

  it('"all matching" names the full count', () => {
    expect(bulkConfirmText('delete', 1240, true)).toContain('all 1,240 products')
    expect(bulkConfirmText('activate', 1240, true)).toContain('1,240')
  })

  it('page-only restore/activate need no confirm', () => {
    expect(bulkConfirmText('restore', 2, false)).toBeNull()
    expect(bulkConfirmText('activate', 2, false)).toBeNull()
  })
})

describe('productStatus', () => {
  const base = { available: true, fulfillmentType: 'account_credit', stock: 0 } as unknown as Product

  it('a trashed product reads as deleted whatever its availability', () => {
    expect(productStatus({ ...base, deletedAt: '2026-11-14T00:00:00Z' })).toBe('deleted')
    expect(productStatus({ ...base, available: false, deletedAt: '2026-11-14T00:00:00Z' })).toBe('deleted')
  })

  it('keeps the existing active / draft rules otherwise', () => {
    expect(productStatus(base)).toBe('active')
    expect(productStatus({ ...base, available: false })).toBe('draft')
  })
})
