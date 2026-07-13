import { describe, it, expect } from 'vitest'
import { toRows, fromRows, rowIssues, type InputFieldRow } from './fieldRows'
import type { InputField } from '@/types'

const row = (over: Partial<InputFieldRow> = {}): InputFieldRow => ({
  key: 'f',
  label: { en: '', ar: '' },
  type: 'text',
  sensitive: false,
  min: '',
  max: '',
  options: [],
  ...over,
})

describe('toRows', () => {
  it('renders numeric bounds as strings', () => {
    const rows = toRows([
      { key: 'qty', label: { en: 'Qty', ar: '' }, type: 'quantity', constraints: { min: 1, max: 5000 } },
    ])
    expect(rows[0].min).toBe('1')
    expect(rows[0].max).toBe('5000')
  })

  it('renders the legacy corrupt {0,0} shape as empty boxes', () => {
    const rows = toRows([
      { key: 'qty', label: { en: '', ar: '' }, type: 'quantity', constraints: { min: 0, max: 0 } },
    ])
    expect(rows[0].min).toBe('')
    expect(rows[0].max).toBe('')
  })

  it('keeps a legitimate 0 bound when the other side is non-zero', () => {
    const rows = toRows([
      { key: 'amt', label: { en: '', ar: '' }, type: 'amount', constraints: { min: 0, max: 10 } },
    ])
    expect(rows[0].min).toBe('0')
    expect(rows[0].max).toBe('10')
  })

  it('carries options and legacyName through', () => {
    const rows = toRows([
      { key: 'srv', label: { en: '', ar: '' }, legacyName: 'الخادم', type: 'select', constraints: { options: ['EU', 'NA'] } },
    ])
    expect(rows[0].options).toEqual(['EU', 'NA'])
    expect(rows[0].legacyName).toBe('الخادم')
  })
})

describe('fromRows', () => {
  it('round-trips a well-formed field losslessly', () => {
    const fields: InputField[] = [
      { key: 'qty', label: { en: 'Qty', ar: 'كمية' }, type: 'quantity', sensitive: false, constraints: { min: 1, max: 5000 } },
      { key: 'srv', label: { en: 'Server', ar: '' }, type: 'select', sensitive: false, constraints: { options: ['EU'] } },
      { key: 'id', label: { en: 'ID', ar: '' }, type: 'text', sensitive: false },
    ]
    expect(fromRows(toRows(fields))).toEqual(fields)
  })

  it('never emits {min:0,max:0} — the corrupt shape is repaired on round-trip', () => {
    const out = fromRows(
      toRows([{ key: 'qty', label: { en: '', ar: '' }, type: 'quantity', constraints: { min: 0, max: 0 } }]),
    )
    expect(out[0].constraints).toBeUndefined()
  })

  it('omits min/max for empty and garbage inputs', () => {
    const out = fromRows([row({ type: 'amount', min: ' ', max: 'abc' })])
    expect(out[0].constraints).toBeUndefined()
  })

  it('keeps a one-sided bound', () => {
    const out = fromRows([row({ type: 'amount', max: '100' })])
    expect(out[0].constraints).toEqual({ max: 100 })
  })

  it('omits constraints for an optionless select', () => {
    const out = fromRows([row({ type: 'select' })])
    expect(out[0].constraints).toBeUndefined()
  })

  it('emits nothing type-foreign: text sheds bounds, select sheds bounds', () => {
    expect(fromRows([row({ type: 'text', min: '1', max: '2', options: ['a'] })])[0].constraints).toBeUndefined()
    expect(fromRows([row({ type: 'select', min: '1', options: ['a'] })])[0].constraints).toEqual({ options: ['a'] })
  })

  it('trims the key', () => {
    expect(fromRows([row({ key: '  accountId ' })])[0].key).toBe('accountId')
  })
})

describe('rowIssues', () => {
  it('passes a clean set', () => {
    expect(
      rowIssues([
        row({ key: 'qty', type: 'quantity', min: '1', max: '5000' }),
        row({ key: 'srv', type: 'select', options: ['EU'] }),
        row({ key: 'id' }),
      ]),
    ).toEqual([])
  })

  it('catches empty and duplicate keys', () => {
    const issues = rowIssues([row({ key: ' ' }), row({ key: 'a' }), row({ key: 'a' })])
    expect(issues.some((m) => m.includes('needs a key'))).toBe(true)
    expect(issues.some((m) => m.includes('Duplicate'))).toBe(true)
  })

  it('catches min>max, negatives, and non-numeric bounds', () => {
    const issues = rowIssues([
      row({ key: 'a', type: 'quantity', min: '10', max: '2' }),
      row({ key: 'b', type: 'amount', min: '-1' }),
      row({ key: 'c', type: 'amount', min: 'abc' }),
    ])
    expect(issues.some((m) => m.includes('min cannot exceed max'))).toBe(true)
    expect(issues.some((m) => m.includes('negative'))).toBe(true)
    expect(issues.some((m) => m.includes('not a number'))).toBe(true)
  })

  it('blocks an optionless select', () => {
    const issues = rowIssues([row({ key: 'srv', type: 'select' })])
    expect(issues.some((m) => m.includes('at least one option'))).toBe(true)
  })
})
