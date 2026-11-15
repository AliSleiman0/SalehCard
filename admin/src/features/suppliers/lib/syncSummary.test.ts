import { describe, expect, it } from 'vitest'
import i18n from '@/i18n/config'
import { syncSummary } from './syncSummary'
import { alertText } from './alertText'
import type { SupplierAlert, SyncReport } from '../api/suppliers'

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

const en = i18n.getFixedT('en')
const ar = i18n.getFixedT('ar')

describe('syncSummary', () => {
  it('says "no changes" for a quiet pass', () => {
    expect(syncSummary(base, en)).toBe('12 products checked · no changes')
  })

  it('lists what changed, singular and plural', () => {
    expect(syncSummary({ ...base, costChanges: 2, paused: 1, promoted: 1, hidden: 1 }, en)).toBe(
      '12 products checked · 2 cost changes · 1 supplier paused (loss) · 1 product switched supplier · 1 taken off sale',
    )
  })

  it('flags warning-only mode and unreachable suppliers', () => {
    const r = { ...base, dryRun: true, hidden: 1, suppliers: [{ id: 11, name: 'speedcard', status: 'unavailable' as const }] }
    expect(syncSummary(r, en)).toBe("warnings only, nothing changed — 12 products checked · 1 would go off sale · couldn't reach speedcard")
  })

  it('is fully Arabic in Arabic (every count form)', () => {
    for (const n of [0, 1, 2, 3, 11, 100]) {
      const s = syncSummary({ ...base, checked: n, dryRun: true, costChanges: n, paused: n, hidden: n, errors: n }, ar)
      expect(s).not.toMatch(/[a-z]{3}/i)
      expect(s).toContain(`المنتجات المفحوصة: ${n}`)
    }
  })
})

const alert = (over: Partial<SupplierAlert>): SupplierAlert => ({
  id: '1',
  kind: 'loss_paused',
  key: 'k',
  open: true,
  message: 'English text',
  count: 1,
  firstAt: base.startedAt,
  lastAt: base.startedAt,
  ...over,
})

describe('alertText', () => {
  it('keeps the server text in English and for alerts without args', () => {
    expect(alertText(alert({ args: { product: 'X' } }), en, 'en')).toBe('English text')
    expect(alertText(alert({}), ar, 'ar')).toBe('English text')
  })

  it('writes a warning-only loss switch in Arabic', () => {
    const a = alert({
      dryRun: true,
      args: {
        product: 'SOUL STAR',
        supplier: 'jentel (#10)',
        cost: '$1.30',
        sell: '$1.20',
        per: '10,000',
        outcome: 'switched',
        to: 'speedcard (#11)',
        toProduct: '1045',
      },
    })
    expect(alertText(a, ar, 'ar')).toBe(
      'SOUL STAR: تكلفة jentel (#10) الآن $1.30 لكل 10,000 بينما نبيع بـ $1.20 — هذه خسارة. ستنتقل الطلبات إلى speedcard (#11) (المنتج 1045). لم يتغيّر شيء بعد.',
    )
  })

  it('has an Arabic sentence for every kind', () => {
    const kinds: SupplierAlert['kind'][] = [
      'cost_changed',
      'loss_paused',
      'loss_recovered',
      'product_hidden',
      'product_shown',
      'fx_missing',
      'fallback_exhausted',
    ]
    for (const kind of kinds) {
      for (const dryRun of [false, true]) {
        const s = alertText(alert({ kind, dryRun, args: { product: 'P', per: '1', reason: 'loss', count: '2' } }), ar, 'ar')
        expect(s).not.toContain('sup_')
        expect(s).not.toBe('English text')
        expect(s).toMatch(/[؀-ۿ]/)
      }
    }
  })
})
