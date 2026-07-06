import { describe, it, expect } from 'vitest'
import { reconcileBridgePhoneField } from './bridgeFields'
import type { InputField } from '@/types'

describe('reconcileBridgePhoneField', () => {
  it('reuses a legacy text field as the phone field and moves it first', () => {
    const fields: InputField[] = [
      { key: 'qty', label: { en: 'Qty', ar: '' }, type: 'quantity' },
      { key: 'field_1', label: { en: 'Mobile number', ar: 'رقم' }, type: 'text', legacyName: 'رقم الهاتف' },
    ]
    const out = reconcileBridgePhoneField(fields)
    expect(out).toHaveLength(2)
    expect(out[0].key).toBe('phone') // renamed
    expect(out[0].type).toBe('text')
    expect(out[0].legacyName).toBe('رقم الهاتف') // preserved
    expect(out[1].key).toBe('qty')
    expect(out.filter((f) => f.key === 'phone')).toHaveLength(1) // no duplicate
  })

  it('dedupes to a single phone field, kept first', () => {
    const fields: InputField[] = [
      { key: 'field_1', label: { en: 'Mobile number', ar: 'رقم' }, type: 'text' }, // legacy dup
      { key: 'qty', label: { en: 'Qty', ar: '' }, type: 'quantity' },
      { key: 'phone', label: { en: 'Mobile number', ar: 'رقم' }, type: 'text' },
    ]
    const out = reconcileBridgePhoneField(fields)
    expect(out[0].key).toBe('phone')
    expect(out.filter((f) => f.key === 'phone')).toHaveLength(1)
    // the existing `phone` field is the canonical one; the stray text field is kept but not as phone
    expect(out.map((f) => f.key)).toEqual(['phone', 'field_1', 'qty'])
  })

  it('appends a phone field when there is no text field to reuse', () => {
    const fields: InputField[] = [{ key: 'qty', label: { en: 'Qty', ar: '' }, type: 'quantity' }]
    const out = reconcileBridgePhoneField(fields)
    expect(out).toHaveLength(2)
    expect(out[0].key).toBe('phone')
    expect(out[0].type).toBe('text')
    expect(out[1].key).toBe('qty')
  })
})
