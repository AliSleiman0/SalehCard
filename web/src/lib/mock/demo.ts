export interface MockVariant {
  l: string
  p: number
}

export interface MockProduct {
  id: string
  brand: string
  title: string
  cat: string
  art: string
  needsId?: boolean
  idLabel?: string
  rating: number
  reviews: number
  sold?: string
  instant: boolean
  agentDisc: number
  variants: MockVariant[]
}

export interface MockCategory {
  id: string
  key: string
  art: string
  tagKey: string
  count: number
}

export const CATEGORIES: MockCategory[] = [
  { id: 'games', key: 'cat_games', art: 'battle', tagKey: 'cat_games_tag', count: 480 },
  { id: 'credits', key: 'cat_credits', art: 'live', tagKey: 'cat_credits_tag', count: 210 },
  { id: 'recharge', key: 'cat_recharge', art: 'recharge', tagKey: 'cat_recharge_tag', count: 95 },
  { id: 'giftcards', key: 'cat_giftcards', art: 'gift', tagKey: 'cat_gift_tag', count: 320 },
  { id: 'software', key: 'cat_software', art: 'soft', tagKey: 'cat_soft_tag', count: 140 },
  { id: 'crypto', key: 'cat_crypto', art: 'crypto', tagKey: 'cat_crypto_tag', count: 28 },
  { id: 'transfer', key: 'cat_transfer', art: 'transfer', tagKey: 'cat_transfer_tag', count: 16 },
]

// Defaults from data.js P(): rating 4.8, reviews 1200, instant true, needsId false, agentDisc 0.08
export const MOCK_PRODUCTS: MockProduct[] = [
  {
    id: 'pubg-uc',
    brand: 'PUBG MOBILE',
    title: 'UC Top-up',
    cat: 'games',
    art: 'battle',
    needsId: true,
    idLabel: 'Player ID',
    rating: 4.9,
    reviews: 48210,
    sold: '2.1M',
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '60 UC', p: 0.99 },
      { l: '325 UC', p: 4.99 },
      { l: '660 UC', p: 9.99 },
      { l: '1800 UC', p: 24.99 },
      { l: '3850 UC', p: 49.99 },
      { l: '8100 UC', p: 99.99 },
    ],
  },
  {
    id: 'mlbb',
    brand: 'MOBILE LEGENDS',
    title: 'Diamonds',
    cat: 'games',
    art: 'sand',
    needsId: true,
    idLabel: 'User ID (Zone ID)',
    rating: 4.8,
    reviews: 31022,
    sold: '980K',
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '86 💎', p: 1.49 },
      { l: '172 💎', p: 2.99 },
      { l: '344 💎', p: 5.99 },
      { l: '706 💎', p: 11.99 },
      { l: '1412 💎', p: 23.49 },
    ],
  },
  {
    id: 'freefire',
    brand: 'FREE FIRE',
    title: 'Diamonds',
    cat: 'games',
    art: 'recharge',
    needsId: true,
    idLabel: 'Player ID',
    rating: 4.8,
    reviews: 1200,
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '100 💎', p: 0.99 },
      { l: '310 💎', p: 2.99 },
      { l: '520 💎', p: 4.99 },
      { l: '1060 💎', p: 9.99 },
    ],
  },
  {
    id: 'valorant',
    brand: 'VALORANT',
    title: 'VP Points',
    cat: 'games',
    art: 'battle',
    needsId: true,
    idLabel: 'Riot ID',
    rating: 4.8,
    reviews: 1200,
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '475 VP', p: 4.99 },
      { l: '1000 VP', p: 9.99 },
      { l: '2050 VP', p: 19.99 },
    ],
  },
  {
    id: 'tiktok',
    brand: 'TIKTOK',
    title: 'Coins',
    cat: 'credits',
    art: 'social',
    needsId: true,
    idLabel: 'TikTok Username',
    rating: 4.9,
    reviews: 22001,
    sold: '1.4M',
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '70 coins', p: 0.99 },
      { l: '350 coins', p: 4.49 },
      { l: '700 coins', p: 8.99 },
      { l: '1400 coins', p: 17.49 },
      { l: '3500 coins', p: 42.99 },
    ],
  },
  {
    id: 'bigo',
    brand: 'BIGO LIVE',
    title: 'Diamonds',
    cat: 'credits',
    art: 'live',
    needsId: true,
    idLabel: 'Bigo ID',
    rating: 4.8,
    reviews: 1200,
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '100 💎', p: 1.49 },
      { l: '500 💎', p: 6.99 },
      { l: '1000 💎', p: 13.49 },
      { l: '5000 💎', p: 64.99 },
    ],
  },
  {
    id: 'uplive',
    brand: 'UP LIVE',
    title: 'Coins',
    cat: 'credits',
    art: 'sand',
    needsId: true,
    idLabel: 'Up Live ID',
    rating: 4.8,
    reviews: 1200,
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '200 coins', p: 2.99 },
      { l: '1000 coins', p: 13.99 },
      { l: '3000 coins', p: 39.99 },
    ],
  },
  {
    id: 'itunes',
    brand: 'iTUNES',
    title: 'Gift Card (US)',
    cat: 'giftcards',
    art: 'ice',
    needsId: false,
    rating: 4.9,
    reviews: 18430,
    sold: '760K',
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '$10', p: 10.9 },
      { l: '$25', p: 26.5 },
      { l: '$50', p: 52 },
      { l: '$100', p: 103 },
    ],
  },
  {
    id: 'xbox',
    brand: 'XBOX',
    title: 'Gift Card',
    cat: 'giftcards',
    art: 'gift',
    needsId: false,
    rating: 4.8,
    reviews: 1200,
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '$10', p: 10.5 },
      { l: '$25', p: 25.9 },
      { l: '$50', p: 51 },
      { l: '$100', p: 101 },
    ],
  },
  {
    id: 'psn',
    brand: 'PLAYSTATION',
    title: 'Store Card',
    cat: 'giftcards',
    art: 'social',
    needsId: false,
    rating: 4.9,
    reviews: 20110,
    sold: '1.1M',
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '$10', p: 10.6 },
      { l: '$25', p: 26 },
      { l: '$50', p: 51.5 },
      { l: '$100', p: 102 },
    ],
  },
  {
    id: 'steam',
    brand: 'STEAM',
    title: 'Wallet Card',
    cat: 'giftcards',
    art: 'soft',
    needsId: false,
    rating: 4.8,
    reviews: 1200,
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '$10', p: 10.4 },
      { l: '$20', p: 20.6 },
      { l: '$50', p: 51 },
      { l: '$100', p: 101.5 },
    ],
  },
  {
    id: 'recharge-tr',
    brand: 'MOBILE',
    title: 'Recharge (TR)',
    cat: 'recharge',
    art: 'recharge',
    needsId: true,
    idLabel: 'Phone Number',
    rating: 4.8,
    reviews: 1200,
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '50 ₺', p: 1.6 },
      { l: '100 ₺', p: 3.1 },
      { l: '250 ₺', p: 7.6 },
      { l: '500 ₺', p: 15 },
    ],
  },
  {
    id: 'netflix',
    brand: 'NETFLIX',
    title: 'Subscription',
    cat: 'software',
    art: 'battle',
    needsId: false,
    rating: 4.7,
    reviews: 9120,
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '1 Month', p: 9.99 },
      { l: '3 Months', p: 27.99 },
      { l: '12 Months', p: 99.99 },
    ],
  },
  {
    id: 'spotify',
    brand: 'SPOTIFY',
    title: 'Premium',
    cat: 'software',
    art: 'gift',
    needsId: false,
    rating: 4.8,
    reviews: 1200,
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '1 Month', p: 9.99 },
      { l: '3 Months', p: 28.99 },
      { l: '6 Months', p: 54.99 },
    ],
  },
  {
    id: 'office',
    brand: 'MS OFFICE 365',
    title: 'License (1yr)',
    cat: 'software',
    art: 'soft',
    needsId: false,
    rating: 4.8,
    reviews: 1200,
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: 'Personal', p: 39.99 },
      { l: 'Family', p: 64.99 },
    ],
  },
  {
    id: 'usdt',
    brand: 'USDT',
    title: 'Tether (TRC-20)',
    cat: 'crypto',
    art: 'crypto',
    needsId: false,
    rating: 4.8,
    reviews: 1200,
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '$25', p: 25.4 },
      { l: '$50', p: 50.6 },
      { l: '$100', p: 100.9 },
      { l: '$500', p: 503 },
    ],
  },
  {
    id: 'btc',
    brand: 'BITCOIN',
    title: 'Voucher',
    cat: 'crypto',
    art: 'crypto',
    needsId: false,
    rating: 4.8,
    reviews: 1200,
    instant: true,
    agentDisc: 0.08,
    variants: [
      { l: '$50', p: 51.5 },
      { l: '$100', p: 102 },
      { l: '$250', p: 254 },
    ],
  },
  {
    id: 'wise',
    brand: 'MONEY TRANSFER',
    title: 'Send Abroad',
    cat: 'transfer',
    art: 'transfer',
    needsId: true,
    idLabel: 'Recipient',
    rating: 4.8,
    reviews: 1200,
    instant: false,
    agentDisc: 0.08,
    variants: [
      { l: '$100', p: 101.5 },
      { l: '$250', p: 252 },
      { l: '$500', p: 503 },
    ],
  },
]

export const mockProduct = (id: string): MockProduct | undefined =>
  MOCK_PRODUCTS.find((p) => p.id === id)

export const mockByCat = (cat: string): MockProduct[] =>
  MOCK_PRODUCTS.filter((p) => p.cat === cat)

// Best sellers (curated order)
export const BEST: MockProduct[] = ['pubg-uc', 'tiktok', 'psn', 'mlbb', 'itunes', 'usdt', 'bigo', 'steam']
  .map(mockProduct)
  .filter((p): p is MockProduct => p !== undefined)

export const FEATURED: MockProduct[] = ['pubg-uc', 'tiktok', 'itunes', 'freefire']
  .map(mockProduct)
  .filter((p): p is MockProduct => p !== undefined)

// ---- Demo account state ----
export interface SavedId {
  id: number
  game: string
  art: string
  label: string
  value: string
  nick: string
}

export interface TimelineStep {
  k: string
  ts: string
}

export interface MockOrder {
  id: string
  date: string
  product: string
  art: string
  total: number
  status: string
  method: string
  fulfill: 'code' | 'credit' | 'transfer'
  code?: string
  pin?: string
  account?: string
  amount?: string
  ts?: string
  ref?: string
  recipient?: { name: string; country: string; detail: string }
  steps?: TimelineStep[]
}

export interface WalletTx {
  id: number
  t: string
  amt: number
  date: string
}

export interface DemoUser {
  name: string
  email: string
  initials: string
  tier: string
  cashback: number
  loyalty: number
}

export interface Demo {
  user: DemoUser
  wallet: number
  agentBalance: number
  savedIds: SavedId[]
  orders: MockOrder[]
  walletTx: WalletTx[]
}

export const DEMO: Demo = {
  user: { name: 'Yusuf Demir', email: 'yusuf@demir.co', initials: 'YD', tier: 'customer', cashback: 12.4, loyalty: 1840 },
  wallet: 142.6,
  agentBalance: 4820.0,
  savedIds: [
    { id: 1, game: 'PUBG MOBILE', art: 'battle', label: 'Main account', value: '5129384761', nick: 'YZF_Sniper' },
    { id: 2, game: 'MOBILE LEGENDS', art: 'sand', label: 'Smurf', value: '1029384 (2841)', nick: 'DarkMage' },
    { id: 3, game: 'TIKTOK', art: 'social', label: 'Creator', value: '@yusuf.demir', nick: '' },
  ],
  orders: [
    { id: 'SC-90421', date: 'Jun 2, 2026', product: 'PUBG MOBILE — 1800 UC', art: 'battle', total: 24.99, status: 'delivered', method: 'Wallet', fulfill: 'credit', account: '5129384761', amount: '1800 UC', ts: 'Jun 2, 2026 · 14:22' },
    { id: 'SC-90388', date: 'Jun 1, 2026', product: 'iTUNES — $50 Gift Card', art: 'ice', total: 52, status: 'delivered', method: 'Visa', fulfill: 'code', code: 'X7M4-K92P-LQ8A-44ZB-9WQ1', pin: '' },
    { id: 'SC-90301', date: 'May 29, 2026', product: 'TIKTOK — 700 Coins', art: 'social', total: 8.99, status: 'delivered', method: 'Wallet', fulfill: 'credit', account: '@yusuf.demir', amount: '700 Coins', ts: 'May 29, 2026 · 09:05' },
    { id: 'SC-90288', date: 'May 28, 2026', product: 'MONEY TRANSFER — $250', art: 'transfer', total: 252, status: 'completed', method: 'USDT', fulfill: 'transfer', ref: 'MT-5521-9080', recipient: { name: 'Ayşe Yılmaz', country: 'Türkiye', detail: 'IBAN •••• 4408' }, steps: [{ k: 'submitted', ts: 'May 28 · 11:20' }, { k: 'processing', ts: 'May 28 · 11:24' }, { k: 'completed', ts: 'May 28 · 13:02' }] },
    { id: 'SC-90244', date: 'May 27, 2026', product: 'USDT — $100 (TRC-20)', art: 'crypto', total: 100.9, status: 'delivered', method: 'USDT', fulfill: 'code', code: 'TRC20:TX...8f29ab', pin: '' },
    { id: 'SC-90150', date: 'May 24, 2026', product: 'NETFLIX — 3 Months', art: 'battle', total: 27.99, status: 'processing', method: 'Wallet', fulfill: 'code', code: '', pin: '' },
  ],
  walletTx: [
    { id: 1, t: 'Top-up · Visa ••42', amt: +100, date: 'Jun 1, 2026' },
    { id: 2, t: 'PUBG MOBILE — 1800 UC', amt: -24.99, date: 'Jun 2, 2026' },
    { id: 3, t: 'Cashback reward', amt: +1.25, date: 'Jun 2, 2026' },
    { id: 4, t: 'Top-up · USDT', amt: +50, date: 'May 28, 2026' },
    { id: 5, t: 'TIKTOK — 700 Coins', amt: -8.99, date: 'May 29, 2026' },
  ],
}

export interface MockReview {
  i: string
  n: string
  s: number
  d: string
  tx: string
}

export const REVIEWS: MockReview[] = [
  { i: 'AM', n: 'Ahmed M.', s: 5, d: '2 days ago', tx: 'Code arrived in literally 3 seconds. Topped up my squad before the match started 🔥' },
  { i: 'SK', n: 'Selin K.', s: 5, d: '5 days ago', tx: 'Wallet balance makes repeat buys instant. Never re-enter my ID. Trusted.' },
  { i: 'JR', n: 'Jorge R.', s: 4, d: '1 week ago', tx: "Best prices I found and the vault keeps every code I've ever bought. Solid." },
]
