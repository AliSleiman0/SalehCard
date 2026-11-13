import { describe, it, expect } from 'vitest'
import { rerouteState } from './reroute'
import type { AdminOrder } from '../api/orders'

function order(over: Partial<AdminOrder>, ff: Partial<AdminOrder['fulfillment']> = {}, mode = 'api'): AdminOrder {
  return {
    status: 'processing',
    items: [{ fulfillmentMode: mode }],
    fulfillment: { statusTimeline: [], ...ff },
    ...over,
  } as unknown as AdminOrder
}

describe('rerouteState', () => {
  it('stuck or ref-less processing supplier orders are eligible', () => {
    expect(rerouteState(order({}, { supplierStuckAt: '2026-11-13T00:00:00Z', providerRef: 'UP' })).eligible).toBe(true)
    expect(rerouteState(order({}, {})).eligible).toBe(true)
  })
  it('a supplier still working on it, a finished order or a non-supplier order is not', () => {
    expect(rerouteState(order({}, { providerRef: 'UP' })).eligible).toBe(false)
    expect(rerouteState(order({ status: 'completed' })).eligible).toBe(false)
    expect(rerouteState(order({}, {}, 'manual_operator')).eligible).toBe(false)
  })
  it('flags an unknown previous outcome as ambiguous', () => {
    expect(rerouteState(order({}, { providerRef: 'search:ALFA:x', supplierStuckAt: 'x' })).ambiguous).toBe(true)
  })
})
