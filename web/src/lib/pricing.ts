/* Shared pricing helpers — agent (reseller) vs retail pricing. */

export interface PricedVariant {
  /** display label, e.g. "1800 UC" or "$50" */
  l: string
  /** retail price in USD */
  p: number
  /** explicit reseller/agent price in USD (from API resellerPrice), if any */
  agentP?: number
}

export interface PricedProduct {
  /** fallback agent discount fraction (e.g. 0.08) when no explicit agentP */
  agentDisc?: number
  variants: PricedVariant[]
}

/** Effective unit price for a variant given retail vs agent view. */
export function priceFor(p: PricedProduct, v: PricedVariant, agent: boolean): number {
  if (!agent) return v.p
  return v.agentP ?? v.p * (1 - (p.agentDisc ?? 0))
}

/** Lowest effective price across a product's variants ("from" price). */
export function fromPrice(p: PricedProduct, agent: boolean): number {
  return Math.min(...p.variants.map((v) => priceFor(p, v, agent)))
}
