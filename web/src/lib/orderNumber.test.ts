import { describe, expect, it } from 'vitest'
import { orderNumber } from './orderNumber'

describe('orderNumber', () => {
  it('is # + the last 8 characters, upper-cased', () => {
    expect(orderNumber('6af90254ce0bb4698d03458d')).toBe('#8D03458D')
  })
})
