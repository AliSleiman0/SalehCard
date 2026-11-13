import { describe, it, expect } from 'vitest'
import { methodFieldPayload, missingRequired } from './methodFields'
import type { TopUpMethodField } from '../api/wallet'

const fields: TopUpMethodField[] = [
  { key: 'ref', label: 'Reference', type: 'text', required: true },
  { key: 'bank', label: 'Bank', type: 'select', required: false, options: ['A', 'B'] },
  { key: 'proof', label: 'Transfer proof', type: 'file', required: true },
]

describe('missingRequired', () => {
  it('lists empty / blank required fields by label', () => {
    expect(missingRequired(fields, {})).toEqual(['Reference', 'Transfer proof'])
    expect(missingRequired(fields, { ref: '  ', proof: 'https://x/p.jpg' })).toEqual(['Reference'])
    expect(missingRequired(fields, { ref: '123', proof: 'https://x/p.jpg' })).toEqual([])
  })
})

describe('methodFieldPayload', () => {
  it('trims, drops empties and keeps method order', () => {
    expect(methodFieldPayload(fields, { proof: 'u', ref: ' 42 ', bank: '' })).toEqual([
      { key: 'ref', value: '42' },
      { key: 'proof', value: 'u' },
    ])
  })
})
