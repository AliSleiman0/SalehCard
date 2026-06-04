export const ART: Record<string, [string, string]> = {
  battle: ['#f7971e', '#ff2d6b'],
  social: ['#00e0ff', '#3b5bff'],
  live: ['#ff4dd2', '#7a2bff'],
  gift: ['#2fd47a', '#0bbaa0'],
  recharge: ['#ffb02e', '#ff5e1a'],
  soft: ['#3b5bff', '#9a3bff'],
  crypto: ['#f7c948', '#ff8a00'],
  transfer: ['#22e3c8', '#3b5bff'],
  sand: ['#8a3bff', '#d633ff'],
  ice: ['#5b9bff', '#22d3e3'],
}

export type FulfillKind = 'code' | 'credit' | 'transfer'

export function fulfillType(cat: string): FulfillKind {
  if (cat === 'transfer') return 'transfer'
  if (cat === 'giftcards' || cat === 'software' || cat === 'crypto') return 'code'
  return 'credit'
}

export function artForCategory(cat: string): string {
  switch (cat) {
    case 'gaming':
    case 'games':
      return 'battle'
    case 'credits':
      return 'live'
    case 'recharge':
      return 'recharge'
    case 'giftcards':
    case 'gift':
      return 'gift'
    case 'software':
      return 'soft'
    case 'crypto':
      return 'crypto'
    case 'transfer':
    case 'finance':
      return 'transfer'
    default:
      return 'soft'
  }
}
