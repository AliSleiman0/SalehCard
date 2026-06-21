// Cosmetic presentation for the 8 stable root domains. The taxonomy itself —
// which domains exist, their slugs, Arabic names, and product counts — comes from
// the API (GET /api/v1/categories). Only the gradient art, a clean English label
// (legacy `name.en` values are inconsistent — some are Arabic), a short tagline,
// and the tile display order are curated here, keyed on the stable rootDomain.

export interface RootMeta {
  art: string // gradient key in @/lib/art ART
  label: string // curated English label
  tag: string // short tagline
  order: number // tile display order
}

export const ROOT_META: Record<string, RootMeta> = {
  games: { art: 'battle', label: 'Games', tag: 'PUBG, MLBB, Free Fire', order: 1 },
  app_topups: { art: 'live', label: 'App Top-ups', tag: 'TikTok, Bigo, Likee', order: 2 },
  telecom: { art: 'recharge', label: 'Telecom', tag: 'Syriatel, MTN, Alfa', order: 3 },
  giftcards: { art: 'gift', label: 'Gift Cards', tag: 'iTunes, Xbox, PSN', order: 4 },
  software: { art: 'soft', label: 'Software', tag: 'Netflix, Office', order: 5 },
  gsm_tools: { art: 'tools', label: 'GSM Tools', tag: 'Unlock & repair tools', order: 6 },
  wallets_crypto: { art: 'crypto', label: 'Crypto / Wallets', tag: 'USDT, BTC', order: 7 },
  money_transfers: { art: 'transfer', label: 'Money Transfers', tag: 'Send abroad', order: 8 },
}

const DEFAULT_ROOT_META: RootMeta = { art: 'soft', label: '', tag: '', order: 99 }

export function rootMeta(rootDomain: string): RootMeta {
  return ROOT_META[rootDomain] ?? DEFAULT_ROOT_META
}
