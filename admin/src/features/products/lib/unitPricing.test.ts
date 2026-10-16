import { describe, it, expect } from 'vitest'
import {
  toForm,
  fromForm,
  issues,
  enableRows,
  disableRows,
  droppedPackages,
  fixedDenomination,
  preview,
  driverRow,
  unitDenomination,
  amountIgnoredByPrice,
  type UnitPricingForm,
} from './unitPricing'
import type { InputFieldRow } from './fieldRows'

describe('amountIgnoredByPrice', () => {
  const fields = [
    { key: 'field_1', type: 'text', label: { en: 'ID' } },
    { key: 'amount', type: 'amount', label: { en: 'Enter Amount' } },
  ]

  it('flags a fixed-price credit product that collects an amount', () => {
    expect(amountIgnoredByPrice({ fulfillmentType: 'account_credit', inputFields: fields })).toEqual(['Enter Amount'])
  })

  it('does not flag a per-unit product', () => {
    expect(
      amountIgnoredByPrice({ fulfillmentType: 'account_credit', unitPricing: { blockUnits: 1000 }, inputFields: fields }),
    ).toEqual([])
  })

  it('flags again when per-unit is cleared (blockUnits 0)', () => {
    expect(
      amountIgnoredByPrice({ fulfillmentType: 'account_credit', unitPricing: { blockUnits: 0 }, inputFields: fields }),
    ).toHaveLength(1)
  })

  it('never flags transfer or code products', () => {
    expect(amountIgnoredByPrice({ fulfillmentType: 'transfer', inputFields: fields })).toEqual([])
    expect(amountIgnoredByPrice({ fulfillmentType: 'code', inputFields: fields })).toEqual([])
  })

  it('ignores products without an amount field and falls back to the key', () => {
    expect(amountIgnoredByPrice({ fulfillmentType: 'account_credit', inputFields: [fields[0]] })).toEqual([])
    expect(
      amountIgnoredByPrice({ fulfillmentType: 'account_credit', inputFields: [{ key: 'coins', type: 'amount' }] }),
    ).toEqual(['coins'])
  })
})

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

const on = (over: Partial<UnitPricingForm> = {}): UnitPricingForm => ({
  enabled: true,
  fieldKey: 'amount',
  blockUnits: '1000',
  ...over,
})

const amount = row({ key: 'amount', type: 'amount', min: '100', max: '200000' })

describe('toForm', () => {
  it('starts a product without a spec in fixed mode with a 1,000 block', () => {
    expect(toForm({})).toEqual({ enabled: false, fieldKey: '', blockUnits: '1000' })
    expect(toForm(null).enabled).toBe(false)
  })

  it('reads a stored spec', () => {
    expect(toForm({ unitPricing: { fieldKey: 'coins', blockUnits: 500 } })).toEqual({
      enabled: true,
      fieldKey: 'coins',
      blockUnits: '500',
    })
  })
})

describe('fromForm', () => {
  it('is undefined when per-unit is off', () => {
    expect(fromForm(on({ enabled: false }), [amount])).toBeUndefined()
  })

  it('emits the spec with the parsed block size', () => {
    expect(fromForm(on({ blockUnits: ' 1000 ' }), [amount])).toEqual({ fieldKey: 'amount', blockUnits: 1000 })
  })

  it('falls back to the first amount field when the chosen key is stale', () => {
    const rows = [row({ key: 'id' }), row({ key: 'coins', type: 'amount' })]
    expect(fromForm(on({ fieldKey: 'gone' }), rows)?.fieldKey).toBe('coins')
    expect(driverRow(on({ fieldKey: '' }), rows)?.key).toBe('coins')
  })
})

describe('issues', () => {
  it('is empty when off or valid', () => {
    expect(issues(on({ enabled: false }), [], [])).toEqual([])
    expect(issues(on(), [amount], [{ price: '17.5' }])).toEqual([])
  })

  it('flags a missing block size and price', () => {
    const out = issues(on({ blockUnits: '0' }), [amount], [{ price: '' }])
    expect(out).toHaveLength(2)
  })

  it('flags no driving amount field', () => {
    const out = issues(on(), [row({ key: 'id' })], [{ price: '1' }])
    expect(out[0]).toMatch(/amount/)
  })

  it('flags bad driver bounds', () => {
    expect(issues(on(), [row({ key: 'amount', type: 'amount', min: '0', max: '10' })], [{ price: '1' }])).toHaveLength(1)
    expect(issues(on(), [row({ key: 'amount', type: 'amount', min: '50', max: '10' })], [{ price: '1' }])).toHaveLength(1)
    expect(issues(on(), [row({ key: 'amount', type: 'amount', min: '50' })], [{ price: '1' }])).toHaveLength(1)
  })

  it('flags a quantity field', () => {
    const out = issues(on(), [amount, row({ key: 'qty', type: 'quantity' })], [{ price: '1' }])
    expect(out).toEqual([expect.stringContaining('"qty"')])
  })
})

describe('enableRows', () => {
  it('removes quantity fields and adds an amount field', () => {
    const { rows, removed } = enableRows([row({ key: 'id' }), row({ key: 'qty', type: 'quantity' })])
    expect(removed).toEqual(['qty'])
    expect(rows.map((r) => r.key)).toEqual(['id', 'amount'])
    expect(rows[1]).toMatchObject({ type: 'amount', label: { en: 'Enter Amount' }, min: '', max: '' })
  })

  it('keeps an existing amount field', () => {
    const { rows, removed } = enableRows([amount])
    expect(removed).toEqual([])
    expect(rows).toEqual([amount])
  })

  it('avoids colliding with a non-amount field keyed amount', () => {
    const { rows } = enableRows([row({ key: 'amount', type: 'text' })])
    expect(rows[1].key).toBe('units')
  })
})

describe('disableRows (switching back to fixed before saving)', () => {
  it('restores removed quantity fields in place and drops the untouched auto amount field', () => {
    const before = [row({ key: 'qty', type: 'quantity' }), row({ key: 'id' }), row({ key: 'q2', type: 'quantity' })]
    const { rows, stash } = enableRows(before)
    expect(rows.map((r) => r.key)).toEqual(['id', 'amount'])
    expect(disableRows(rows, stash)).toEqual(before)
  })

  it('keeps the auto amount field once the admin edited it', () => {
    const before = [row({ key: 'id' }), row({ key: 'qty', type: 'quantity' })]
    const { rows, stash } = enableRows(before)
    const edited = rows.map((r) => (r.type === 'amount' ? { ...r, min: '100' } : r))
    expect(disableRows(edited, stash).map((r) => r.key)).toEqual(['id', 'qty', 'amount'])
  })

  it('is a no-op without a stash (product loaded already per-unit)', () => {
    const rows = [amount]
    expect(disableRows(rows, null)).toBe(rows)
  })
})

describe('droppedPackages', () => {
  it('lists the saveable packages after the first', () => {
    expect(
      droppedPackages([
        { denomination: '1,000', price: '1' },
        { denomination: '5,000', price: '5' },
        { denomination: '', price: '' },
        { denomination: '10,000', price: '9' },
      ]),
    ).toEqual(['5,000', '10,000'])
    expect(droppedPackages([{ denomination: '1,000', price: '1' }])).toEqual([])
  })
})

describe('fixedDenomination', () => {
  it('replaces only the auto "per N" label with the block size', () => {
    expect(fixedDenomination('per 1,000', on())).toBe('1,000')
    expect(fixedDenomination('90000 coins', on())).toBe('90000 coins')
    expect(fixedDenomination('per unit', on({ blockUnits: '' }))).toBe('')
  })
})

describe('preview & denomination', () => {
  it('prices the min sample like the server', () => {
    expect(preview(on(), [amount], '17.5')).toBe('Customer enters 100 – 200,000 · e.g. 100 = $1.75')
    expect(preview(on(), [row({ key: 'amount', type: 'amount', min: '45000', max: '200000' })], '17.5')).toBe(
      'Customer enters 45,000 – 200,000 · e.g. 45,000 = $787.50',
    )
  })

  it('is empty until usable', () => {
    expect(preview(on(), [amount], '')).toBe('')
    expect(preview(on({ blockUnits: '' }), [amount], '1')).toBe('')
  })

  it('formats the denomination', () => {
    expect(unitDenomination(on())).toBe('per 1,000')
    expect(unitDenomination(on({ blockUnits: '' }))).toBe('per unit')
  })
})
