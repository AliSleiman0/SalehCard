/* Shared pricing helpers — agent (reseller) vs retail pricing. */

import { unitLinePrice } from './productFields'

export interface PricedVariant {
  /** display label, e.g. "1800 UC" or "$50" */
  l: string
  /** retail price in USD */
  p: number
  /** discounted retail price in USD while an offer is live, if any */
  offerP?: number
  /** explicit reseller/agent price in USD (from API resellerPrice), if any */
  agentP?: number
}

export interface PricedProduct {
  /** fallback agent discount fraction when no explicit agentP (0 = retail) */
  agentDisc?: number
  variants: PricedVariant[]
  /** per-unit pricing: the single variant prices `blockUnits` units and the
   * cheapest purchasable line is `minAmount` units */
  unit?: { blockUnits: number; minAmount: number }
}

/** Effective unit price for a variant given retail vs agent view.
 * Display-only: the server re-prices authoritatively at checkout. */
export function priceFor(p: PricedProduct, v: PricedVariant, agent: boolean): number {
  if (!agent) return v.offerP ?? v.p
  return v.agentP ?? v.p * (1 - (p.agentDisc ?? 0))
}

/** Lowest effective price across a product's variants ("from" price). */
export function fromPrice(p: PricedProduct, agent: boolean): number {
  if (p.unit && p.unit.blockUnits > 0 && p.variants.length > 0) {
    // Unit-priced: "from" = the price of the minimum allowed amount (no
    // minimum → one block, i.e. the variant price itself).
    const min = p.unit.minAmount > 0 ? p.unit.minAmount : p.unit.blockUnits
    return unitLinePrice(priceFor(p, p.variants[0], agent), min, p.unit.blockUnits)
  }
  return Math.min(...p.variants.map((v) => priceFor(p, v, agent)))
}
