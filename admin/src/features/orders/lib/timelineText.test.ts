import { afterEach, beforeAll, describe, expect, it } from 'vitest'
import i18n from '@/i18n/config'
import { timelineNote, timelineStatus } from './timelineText'

// The keys this test reads (the locale files gain them when the fragments merge).
beforeAll(() => {
  i18n.addResources('ar', 'common', {
    tl_note_reversal_failed: 'تعذّرت إعادة المبلغ المدفوع',
    tl_note_codes_delivered: 'تم تسليم الأكواد — عدد الأسطر: {{n}}',
    tl_note_sent_another: 'أرسله المسؤول إلى مورد آخر (#{{n}})',
    tl_note_fallback: 'تعذّر على المورد {{from}} تنفيذ الطلب — تجري المحاولة تلقائياً مع المورد #{{to}}',
    tl_none: '(لا يوجد)',
    tl_note_stuck: 'عالق عند المورد — يحتاج انتباهك: {{why}}',
    tl_why_rejected: 'رفضه المورد',
    tl_status_stuck_upstream: 'عالق عند المورد',
  })
})

const t = i18n.t.bind(i18n)

describe('timelineNote', () => {
  afterEach(() => i18n.changeLanguage('en'))

  it('returns the server English unchanged in English', () => {
    expect(timelineNote('charge reversal failed', t, 'en')).toBe('charge reversal failed')
    expect(timelineNote('codes delivered for 2 line(s)', t, 'en')).toBe('codes delivered for 2 line(s)')
  })

  it('translates a fixed sentence', async () => {
    await i18n.changeLanguage('ar')
    expect(timelineNote('charge reversal failed', t, 'ar')).toBe('تعذّرت إعادة المبلغ المدفوع')
  })

  it('translates a sentence with a number, keeping the number', async () => {
    await i18n.changeLanguage('ar')
    expect(timelineNote('codes delivered for 3 line(s)', t, 'ar')).toBe('تم تسليم الأكواد — عدد الأسطر: 3')
    expect(timelineNote('sent to another supplier (#12) by admin', t, 'ar')).toBe('أرسله المسؤول إلى مورد آخر (#12)')
    expect(
      timelineNote("supplier (none) couldn't take the order — trying supplier #7 automatically", t, 'ar'),
    ).toBe('تعذّر على المورد (لا يوجد) تنفيذ الطلب — تجري المحاولة تلقائياً مع المورد #7')
  })

  it('translates the stuck prefix and keeps the supplier reason as it is', async () => {
    await i18n.changeLanguage('ar')
    expect(timelineNote('stuck upstream — needs attention: Player ID not found', t, 'ar')).toBe(
      'عالق عند المورد — يحتاج انتباهك: Player ID not found',
    )
    // A reason the server writes itself is translated too.
    expect(timelineNote('stuck upstream — needs attention: rejected by the supplier', t, 'ar')).toBe(
      'عالق عند المورد — يحتاج انتباهك: رفضه المورد',
    )
  })

  it('leaves an unknown or admin-typed note unchanged', async () => {
    await i18n.changeLanguage('ar')
    expect(timelineNote('customer asked by phone', t, 'ar')).toBe('customer asked by phone')
    expect(timelineNote('', t, 'ar')).toBe('')
  })
})

describe('timelineStatus', () => {
  afterEach(() => i18n.changeLanguage('en'))

  it('translates a known status and falls back to the raw value', async () => {
    await i18n.changeLanguage('ar')
    expect(timelineStatus('stuck_upstream', t)).toBe('عالق عند المورد')
    expect(timelineStatus('some_new_status', t)).toBe('some_new_status')
  })
})
