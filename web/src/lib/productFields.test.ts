import { describe, it, expect } from 'vitest'
import {
  amountHasFraction,
  amountInRange,
  clampQty,
  fmtAmount,
  lineBelowMinimum,
  normalizeDigits,
  parseAmount,
  sanitizeAmountInput,
  qtyBounds,
  unitAmountField,
  unitLinePrice,
} from './productFields'
import { fromPrice } from './pricing'
import type { InputField } from '@/types'

function qtyField(min?: number, max?: number): InputField {
  return {
    key: 'qty',
    label: { en: 'Quantity' },
    type: 'quantity',
    sensitive: false,
    constraints: min === undefined && max === undefined ? undefined : { min, max },
  }
}

describe('qtyBounds', () => {
  it('defaults to 1..99 with no quantity field', () => {
    expect(qtyBounds({ inputFields: [] })).toEqual({ min: 1, max: 99 })
    expect(qtyBounds(undefined)).toEqual({ min: 1, max: 99 })
    expect(qtyBounds({ inputFields: [qtyField()] })).toEqual({ min: 1, max: 99 })
  })
  it('treats {0,0}, a missing bound and min>max as unbounded', () => {
    expect(qtyBounds({ inputFields: [qtyField(0, 0)] })).toEqual({ min: 1, max: 99 })
    expect(qtyBounds({ inputFields: [qtyField(5, undefined)] })).toEqual({ min: 1, max: 99 })
    expect(qtyBounds({ inputFields: [qtyField(10, 2)] })).toEqual({ min: 1, max: 99 })
  })
  it('uses the field bounds, which may exceed 99', () => {
    expect(qtyBounds({ inputFields: [qtyField(10, 500)] })).toEqual({ min: 10, max: 500 })
  })
  it('clamps the minimum to at least 1', () => {
    expect(qtyBounds({ inputFields: [qtyField(0, 20)] })).toEqual({ min: 1, max: 20 })
  })
  it('clampQty keeps a quantity inside the bounds', () => {
    expect(clampQty(1, { min: 10, max: 500 })).toBe(10)
    expect(clampQty(900, { min: 10, max: 500 })).toBe(500)
    expect(clampQty(42, { min: 10, max: 500 })).toBe(42)
  })
})

describe('unitAmountField', () => {
  const amount: InputField = {
    key: 'amount',
    label: { en: 'Enter Amount' },
    type: 'amount',
    sensitive: false,
    constraints: { min: 100, max: 200000 },
  }
  it('finds the driving amount field', () => {
    expect(
      unitAmountField({ inputFields: [amount], unitPricing: { fieldKey: 'amount', blockUnits: 1000 } }),
    ).toBe(amount)
  })
  it('is undefined when not (validly) unit-priced', () => {
    expect(unitAmountField({ inputFields: [amount] })).toBeUndefined()
    expect(
      unitAmountField({ inputFields: [amount], unitPricing: { fieldKey: 'amount', blockUnits: 0 } }),
    ).toBeUndefined()
    expect(
      unitAmountField({ inputFields: [amount], unitPricing: { fieldKey: 'other', blockUnits: 1000 } }),
    ).toBeUndefined()
  })
})

describe('unitLinePrice', () => {
  it('scales the block price to the amount, rounded to cents', () => {
    expect(unitLinePrice(1.5, 45000, 1000)).toBe(67.5)
    expect(unitLinePrice(0.99, 333, 1000)).toBe(0.33) // 0.32967 -> 0.33
    expect(unitLinePrice(10, 1000, 1000)).toBe(10)
  })
  it('returns 0 for a non-positive block size', () => {
    expect(unitLinePrice(10, 100, 0)).toBe(0)
  })
})

describe('amount parsing / range', () => {
  it('parses digits only', () => {
    expect(parseAmount('45,000')).toBe(45000)
    expect(parseAmount('')).toBeNaN()
  })
  it('normalises Arabic-Indic and Persian digits', () => {
    expect(normalizeDigits('٤٥٠٠٠')).toBe('45000')
    expect(normalizeDigits('۴۵۰۰۰')).toBe('45000')
    expect(parseAmount('٤٥٬٠٠٠')).toBe(45000)
    expect(parseAmount('۱۲۳')).toBe(123)
  })
  it('strips grouping separators and an all-zero decimal part', () => {
    expect(parseAmount('45,000.00')).toBe(45000) // not 4,500,000
    expect(parseAmount('45 000')).toBe(45000)
    expect(parseAmount('45000.')).toBe(45000)
    expect(parseAmount('٤٥٬٠٠٠٫٠٠')).toBe(45000)
  })
  it('rejects real decimals (whole units only)', () => {
    expect(amountHasFraction('45.5')).toBe(true)
    expect(parseAmount('45.5')).toBeNaN()
    expect(amountHasFraction('45,000.00')).toBe(false)
    expect(amountHasFraction('45000')).toBe(false)
  })
  it('sanitises box input, keeping one decimal point so fractions can be flagged', () => {
    expect(sanitizeAmountInput('45,000.00')).toBe('45000.00')
    expect(sanitizeAmountInput('a4.5.6')).toBe('4.56')
    expect(sanitizeAmountInput('٤٥')).toBe('45')
    expect(sanitizeAmountInput('1'.repeat(20))).toHaveLength(15)
  })
  it('flags line totals below the server minimum of $0.01', () => {
    expect(lineBelowMinimum(unitLinePrice(1, 4, 1000))).toBe(true) // 0.004 -> 0.00
    expect(lineBelowMinimum(unitLinePrice(1, 5, 1000))).toBe(false) // 0.005 -> 0.01
    expect(lineBelowMinimum(0.01)).toBe(false)
  })
  it('checks the range (max <= 0 = unbounded)', () => {
    expect(amountInRange(100, 100, 200000)).toBe(true)
    expect(amountInRange(99, 100, 200000)).toBe(false)
    expect(amountInRange(200001, 100, 200000)).toBe(false)
    expect(amountInRange(NaN, 100, 200000)).toBe(false)
    expect(amountInRange(9e9, 100, 0)).toBe(true)
  })
  it('formats with thousands separators', () => {
    expect(fmtAmount(45000)).toBe('45,000')
  })
})

describe('fromPrice for unit-priced products', () => {
  const variants = [{ l: '1,000', p: 2, offerP: 1.5, agentP: 1.2 }]
  it('prices the minimum amount', () => {
    const p = { variants, unit: { blockUnits: 1000, minAmount: 500 } }
    expect(fromPrice(p, false)).toBe(0.75)
    expect(fromPrice(p, true)).toBe(0.6)
  })
  it('falls back to one block with no minimum', () => {
    const p = { variants, unit: { blockUnits: 1000, minAmount: 0 } }
    expect(fromPrice(p, false)).toBe(1.5)
  })
})
