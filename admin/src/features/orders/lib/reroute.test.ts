import { describe, it, expect } from 'vitest'
import { rerouteState, supplierQty } from './reroute'
import type { AdminOrder } from '../api/orders'

function order(over: Partial<AdminOrder>, ff: Partial<AdminOrder['fulfillment']> = {}, mode = 'api', type = 'account_credit'): AdminOrder {
  return {
    status: 'processing',
    items: [{ fulfillmentMode: mode, fulfillmentType: type, qty: 1 }],
    fulfillment: { statusTimeline: [], ...ff },
    ...over,
  } as unknown as AdminOrder
}

describe('rerouteState', () => {
  it('stuck or ref-less processing supplier orders are eligible', () => {
    expect(rerouteState(order({}, { supplierStuckAt: '2026-11-13T00:00:00Z', providerRef: 'UP' })).eligible).toBe(true)
    expect(rerouteState(order({}, {})).eligible).toBe(true)
    expect(rerouteState(order({}, {})).fromManual).toBe(false)
  })
  it('a supplier still working on it or a finished order is not', () => {
    expect(rerouteState(order({}, { providerRef: 'UP' })).eligible).toBe(false)
    expect(rerouteState(order({ status: 'completed' })).eligible).toBe(false)
  })
  it('a single-line manual account-credit order can be sent via a supplier', () => {
    const s = rerouteState(order({}, {}, 'manual_operator'))
    expect(s.eligible).toBe(true)
    expect(s.fromManual).toBe(true)
    expect(rerouteState(order({}, {}, '')).fromManual).toBe(true) // legacy mode-less line
  })
  it('transfers, bridge, code, multi-line and completed manual orders are not', () => {
    expect(rerouteState(order({}, {}, 'manual_operator', 'transfer')).eligible).toBe(false)
    expect(rerouteState(order({}, {}, 'bridge_device')).eligible).toBe(false)
    expect(rerouteState(order({}, {}, 'inventory', 'code')).eligible).toBe(false)
    const two = order({}, {}, 'manual_operator')
    two.items = [two.items[0], two.items[0]]
    expect(rerouteState(two).eligible).toBe(false)
    expect(rerouteState(order({ status: 'completed' }, {}, 'manual_operator')).eligible).toBe(false)
  })
  it('flags an unknown previous outcome as ambiguous', () => {
    expect(rerouteState(order({}, { providerRef: 'search:ALFA:x', supplierStuckAt: 'x' })).ambiguous).toBe(true)
  })
})

describe('supplierQty', () => {
  it('sends the typed units on a per-unit line, else the cart qty', () => {
    expect(supplierQty({ qty: 1, amount: 10000 })).toBe(10000)
    expect(supplierQty({ qty: 3 })).toBe(3)
  })
})
