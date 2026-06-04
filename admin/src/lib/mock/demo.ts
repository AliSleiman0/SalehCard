// Demo data ported from the admin prototype's data.js. Content is intentionally
// English (per the design). Screens that are not yet wired to the backend render
// from here, each marked with a // TODO at its call site.
import type { FfKey } from '@/components'

export interface DemoProduct {
  id: string
  title: string
  art: string
  cat: string
  ff: FfKey
  variants: number
  stock: number | null
  status: string
  price: string
  sold: number
  rating: number
}

export const CATS = [
  'Game Top-ups',
  'App & Live Credits',
  'Mobile Recharge',
  'Gift Cards',
  'Software & Apps',
  'Crypto',
  'Money Transfers',
]

export const products: DemoProduct[] = [
  { id: 'PRD-1042', title: 'PUBG Mobile UC', art: 'pubg', cat: 'Game Top-ups', ff: 'credit', variants: 8, stock: 9999, status: 'active', price: '0.99–99.00', sold: 18420, rating: 4.9 },
  { id: 'PRD-1043', title: 'TikTok Coins', art: 'tiktok', cat: 'App & Live Credits', ff: 'credit', variants: 6, stock: 9999, status: 'active', price: '1.29–129.00', sold: 12090, rating: 4.8 },
  { id: 'PRD-1044', title: 'Bigo Live Diamonds', art: 'bigo', cat: 'App & Live Credits', ff: 'credit', variants: 7, stock: 9999, status: 'active', price: '1.99–199.00', sold: 7340, rating: 4.7 },
  { id: 'PRD-1045', title: 'iTunes Gift Card (US)', art: 'itunes', cat: 'Gift Cards', ff: 'code', variants: 5, stock: 213, status: 'active', price: '10–100', sold: 9610, rating: 4.9 },
  { id: 'PRD-1046', title: 'PlayStation Store (US)', art: 'psn', cat: 'Gift Cards', ff: 'code', variants: 4, stock: 41, status: 'active', price: '10–100', sold: 5230, rating: 4.8 },
  { id: 'PRD-1047', title: 'Xbox Gift Card', art: 'xbox', cat: 'Gift Cards', ff: 'code', variants: 4, stock: 8, status: 'active', price: '10–75', sold: 3110, rating: 4.7 },
  { id: 'PRD-1048', title: 'Netflix Subscription', art: 'netflix', cat: 'Software & Apps', ff: 'code', variants: 3, stock: 0, status: 'out', price: '15–60', sold: 6720, rating: 4.6 },
  { id: 'PRD-1049', title: 'Steam Wallet (US)', art: 'steam', cat: 'Gift Cards', ff: 'code', variants: 5, stock: 156, status: 'active', price: '5–100', sold: 8830, rating: 4.9 },
  { id: 'PRD-1050', title: 'Mobile Legends Diamonds', art: 'mlbb', cat: 'Game Top-ups', ff: 'credit', variants: 9, stock: 9999, status: 'active', price: '0.49–79.00', sold: 14250, rating: 4.8 },
  { id: 'PRD-1051', title: 'USDT Voucher (TRC20)', art: 'usdt', cat: 'Crypto', ff: 'code', variants: 6, stock: 27, status: 'active', price: '10–500', sold: 2140, rating: 4.9 },
  { id: 'PRD-1052', title: 'Microsoft Office 365', art: 'office', cat: 'Software & Apps', ff: 'code', variants: 2, stock: 19, status: 'active', price: '69–99', sold: 1420, rating: 4.7 },
  { id: 'PRD-1053', title: 'Free Fire Diamonds', art: 'freefire', cat: 'Game Top-ups', ff: 'credit', variants: 7, stock: 9999, status: 'active', price: '0.99–49.00', sold: 11030, rating: 4.8 },
  { id: 'PRD-1054', title: 'Turkcell Recharge', art: 'recharge', cat: 'Mobile Recharge', ff: 'credit', variants: 5, stock: 9999, status: 'active', price: '5–100', sold: 4560, rating: 4.6 },
  { id: 'PRD-1055', title: 'Spotify Premium', art: 'spotify', cat: 'Software & Apps', ff: 'code', variants: 3, stock: 62, status: 'draft', price: '10–60', sold: 0, rating: 0 },
  { id: 'PRD-1056', title: 'Western Union Transfer', art: 'visa', cat: 'Money Transfers', ff: 'transfer', variants: 1, stock: null, status: 'active', price: 'service', sold: 880, rating: 4.5 },
]

export interface DemoInventory {
  id: string
  title: string
  art: string
  uploaded: number
  available: number
  delivered: number
  expired: number
  threshold: number
  level: 'hi' | 'mid' | 'lo'
}

export const inventory: DemoInventory[] = [
  { id: 'PRD-1045', title: 'iTunes Gift Card (US)', art: 'itunes', uploaded: 1200, available: 213, delivered: 962, expired: 25, threshold: 100, level: 'hi' },
  { id: 'PRD-1046', title: 'PlayStation Store (US)', art: 'psn', uploaded: 800, available: 41, delivered: 742, expired: 17, threshold: 50, level: 'mid' },
  { id: 'PRD-1047', title: 'Xbox Gift Card', art: 'xbox', uploaded: 600, available: 8, delivered: 588, expired: 4, threshold: 40, level: 'lo' },
  { id: 'PRD-1048', title: 'Netflix Subscription', art: 'netflix', uploaded: 900, available: 0, delivered: 894, expired: 6, threshold: 60, level: 'lo' },
  { id: 'PRD-1049', title: 'Steam Wallet (US)', art: 'steam', uploaded: 1500, available: 156, delivered: 1330, expired: 14, threshold: 120, level: 'mid' },
  { id: 'PRD-1051', title: 'USDT Voucher (TRC20)', art: 'usdt', uploaded: 400, available: 27, delivered: 369, expired: 4, threshold: 50, level: 'lo' },
  { id: 'PRD-1052', title: 'Microsoft Office 365', art: 'office', uploaded: 300, available: 19, delivered: 277, expired: 4, threshold: 30, level: 'lo' },
  { id: 'PRD-1055', title: 'Spotify Premium', art: 'spotify', uploaded: 200, available: 62, delivered: 134, expired: 4, threshold: 40, level: 'hi' },
]

export const uploadHistory = [
  { date: 'Jun 4, 2026 · 14:22', product: 'iTunes Gift Card (US)', count: 500, by: 'Omar A.', ok: 498, dupes: 2 },
  { date: 'Jun 3, 2026 · 09:10', product: 'Steam Wallet (US)', count: 750, by: 'Layla H.', ok: 750, dupes: 0 },
  { date: 'Jun 1, 2026 · 18:46', product: 'USDT Voucher (TRC20)', count: 200, by: 'Omar A.', ok: 196, dupes: 4 },
  { date: 'May 30, 2026 · 11:05', product: 'PlayStation Store (US)', count: 400, by: 'Mona K.', ok: 400, dupes: 0 },
]

const customers: [string, string][] = [
  ['Yusuf Demir', 'yusuf.demir@gmail.com'],
  ['Aisha Rahman', 'aisha.r@outlook.com'],
  ['Mehmet Kaya', 'mkaya@gmail.com'],
  ['Sara Nasser', 'sara.nasser@icloud.com'],
  ['Ali Hassan', 'ali.hassan@gmail.com'],
  ['Elif Yıldız', 'elif.yildiz@gmail.com'],
  ['Omar Farouk', 'omar.f@proton.me'],
  ['Lina Saleh', 'lina.saleh@gmail.com'],
  ['Burak Şahin', 'burak.sahin@gmail.com'],
  ['Noor Khalid', 'noor.k@outlook.com'],
  ['Hamza Aziz', 'hamza.aziz@gmail.com'],
  ['Zeynep Arslan', 'zeynep.a@gmail.com'],
]
const pays = ['wallet', 'visa', 'usdt']
const ostatus = ['delivered', 'delivered', 'delivered', 'processing', 'refunded', 'failed']

export interface DemoOrder {
  id: string
  customer: string
  email: string
  product: string
  art: string
  ff: FfKey
  amount: number
  cur: 'USD' | 'TRY'
  pay: string
  status: string
  date: string
  qty: number
}

function mkOrder(i: number): DemoOrder {
  const c = customers[i % customers.length]
  const p = products[(i * 3 + 1) % products.length]
  const ff = p.ff
  let st = ostatus[i % ostatus.length]
  if (ff === 'transfer') st = ['processing', 'delivered', 'processing'][i % 3]
  const amt = [4.99, 9.99, 12.5, 19.99, 24.99, 29.0, 49.99, 75.0, 99.0, 129.0][i % 10]
  const cur: 'USD' | 'TRY' = i % 4 === 0 ? 'TRY' : 'USD'
  return {
    id: 'SC-' + (74210 - i),
    customer: c[0],
    email: c[1],
    product: p.title,
    art: p.art,
    ff,
    amount: amt,
    cur,
    pay: pays[i % 3],
    status: st,
    date: ['2m', '14m', '38m', '1h', '2h', '3h', '5h', '6h', 'Yesterday', 'Yesterday', '2d', '2d'][i % 12] + (i < 8 ? ' ago' : ''),
    qty: (i % 3) + 1,
  }
}
export const orders: DemoOrder[] = Array.from({ length: 14 }, (_, i) => mkOrder(i))

export interface DemoUser {
  id: string
  name: string
  email: string
  role: string
  balance: number
  cur: 'USD' | 'TRY'
  orders: number
  spent: number
  joined: string
  status: string
}

export const users: DemoUser[] = [
  { id: 'U-9001', name: 'Yusuf Demir', email: 'yusuf.demir@gmail.com', role: 'customer', balance: 142.5, cur: 'USD', orders: 38, spent: 1240.0, joined: 'Mar 2025', status: 'active' },
  { id: 'U-9002', name: 'Aisha Rahman', email: 'aisha.r@outlook.com', role: 'customer', balance: 12.0, cur: 'USD', orders: 9, spent: 318.4, joined: 'Jun 2025', status: 'active' },
  { id: 'U-9003', name: 'GameHub Store', email: 'ops@gamehub.co', role: 'reseller', balance: 4820.0, cur: 'USD', orders: 612, spent: 28400.0, joined: 'Jan 2024', status: 'active' },
  { id: 'U-9004', name: 'Mehmet Kaya', email: 'mkaya@gmail.com', role: 'customer', balance: 0.0, cur: 'TRY', orders: 3, spent: 64.2, joined: 'Apr 2026', status: 'active' },
  { id: 'U-9005', name: 'Omar Farouk', email: 'omar.f@proton.me', role: 'admin', balance: 0.0, cur: 'USD', orders: 0, spent: 0, joined: 'Dec 2023', status: 'active' },
  { id: 'U-9006', name: 'Sara Nasser', email: 'sara.nasser@icloud.com', role: 'customer', balance: 58.75, cur: 'USD', orders: 21, spent: 690.0, joined: 'Aug 2025', status: 'suspended' },
  { id: 'U-9007', name: 'TopUp Express', email: 'sales@topupexpress.io', role: 'reseller', balance: 9120.5, cur: 'USD', orders: 1840, spent: 71200.0, joined: 'Sep 2023', status: 'active' },
  { id: 'U-9008', name: 'Elif Yıldız', email: 'elif.yildiz@gmail.com', role: 'customer', balance: 24.0, cur: 'TRY', orders: 14, spent: 402.0, joined: 'Feb 2026', status: 'active' },
  { id: 'U-9009', name: 'Ali Hassan', email: 'ali.hassan@gmail.com', role: 'customer', balance: 6.5, cur: 'USD', orders: 7, spent: 188.9, joined: 'Nov 2025', status: 'active' },
  { id: 'U-9010', name: 'Noor Khalid', email: 'noor.k@outlook.com', role: 'customer', balance: 99.0, cur: 'USD', orders: 31, spent: 1510.0, joined: 'May 2024', status: 'active' },
  { id: 'U-9011', name: 'PixelKart Agency', email: 'hi@pixelkart.com', role: 'reseller', balance: 2310.0, cur: 'USD', orders: 408, spent: 19800.0, joined: 'Mar 2025', status: 'active' },
  { id: 'U-9012', name: 'Lina Saleh', email: 'lina.saleh@gmail.com', role: 'customer', balance: 33.2, cur: 'USD', orders: 12, spent: 356.0, joined: 'Jul 2025', status: 'active' },
]

export interface DemoReseller {
  id: string
  name: string
  email: string
  tier: 'Gold' | 'Silver' | 'Bronze'
  balance: number
  margin: number
  orders: number
  vol: number
  status: string
}

export const resellers: DemoReseller[] = [
  { id: 'U-9007', name: 'TopUp Express', email: 'sales@topupexpress.io', tier: 'Gold', balance: 9120.5, margin: 12, orders: 1840, vol: 71200, status: 'active' },
  { id: 'U-9003', name: 'GameHub Store', email: 'ops@gamehub.co', tier: 'Gold', balance: 4820.0, margin: 12, orders: 612, vol: 28400, status: 'active' },
  { id: 'U-9011', name: 'PixelKart Agency', email: 'hi@pixelkart.com', tier: 'Silver', balance: 2310.0, margin: 8, orders: 408, vol: 19800, status: 'active' },
  { id: 'U-9021', name: 'CardNest', email: 'team@cardnest.io', tier: 'Silver', balance: 1180.0, margin: 8, orders: 287, vol: 12400, status: 'active' },
  { id: 'U-9022', name: 'QuickCredit MENA', email: 'ops@quickcredit.me', tier: 'Bronze', balance: 420.0, margin: 5, orders: 96, vol: 4100, status: 'suspended' },
  { id: 'U-9023', name: 'DigiVault Resell', email: 'sales@digivault.co', tier: 'Bronze', balance: 760.0, margin: 5, orders: 134, vol: 6300, status: 'active' },
]

export const tiers = [
  { name: 'Bronze', discount: 5, limit: 2000, count: 42, color: '#cd7f4d' },
  { name: 'Silver', discount: 8, limit: 5000, count: 18, color: '#9aa3b5' },
  { name: 'Gold', discount: 12, limit: 15000, count: 7, color: '#ffb02e' },
]

export const tierColor: Record<string, string> = { Gold: '#ffb02e', Silver: '#9aa3b5', Bronze: '#cd7f4d' }

export interface DemoTx {
  id: string
  user: string
  role: string
  type: string
  amount: number
  cur: 'USD' | 'TRY'
  method: string
  date: string
}

function mkTx(i: number): DemoTx {
  const u = users[i % users.length]
  const types = ['topup', 'purchase', 'purchase', 'refund', 'adjust', 'sub-balance', 'purchase', 'topup']
  const ty = types[i % types.length]
  const sign = ty === 'purchase' ? -1 : 1
  const amt = [50, 9.99, 24.99, 12.5, 100, 500, 4.99, 29.0][i % 8]
  return {
    id: 'TX-' + (50230 - i),
    user: u.name,
    role: u.role,
    type: ty,
    amount: sign * amt,
    cur: u.cur,
    method: ty === 'topup' ? ['visa', 'usdt'][i % 2] : ty === 'purchase' ? 'wallet' : '—',
    date: ['Jun 5 · 10:42', 'Jun 5 · 09:15', 'Jun 4 · 22:08', 'Jun 4 · 18:30', 'Jun 4 · 14:55', 'Jun 3 · 20:11', 'Jun 3 · 12:40', 'Jun 2 · 16:22'][i % 8],
  }
}
export const transactions: DemoTx[] = Array.from({ length: 14 }, (_, i) => mkTx(i))

export const usdtQueue = [
  { id: 'TX-50244', user: 'GameHub Store', amount: 500, txhash: '0x7a3f…c91b', submitted: '8m ago', network: 'TRC20' },
  { id: 'TX-50243', user: 'Hamza Aziz', amount: 120, txhash: '0x2e9d…41a0', submitted: '26m ago', network: 'TRC20' },
  { id: 'TX-50241', user: 'TopUp Express', amount: 1000, txhash: '0x9f12…7b3e', submitted: '1h ago', network: 'ERC20' },
  { id: 'TX-50238', user: 'Noor Khalid', amount: 75, txhash: '0x4c81…d2f7', submitted: '2h ago', network: 'TRC20' },
]

export interface DemoPromo {
  code: string
  type: 'percent' | 'fixed' | 'cashback'
  value: number
  used: number
  limit: number
  start: string
  end: string
  status: string
}

export const promos: DemoPromo[] = [
  { code: 'WELCOME10', type: 'percent', value: 10, used: 1840, limit: 5000, start: 'May 1', end: 'Jun 30', status: 'active' },
  { code: 'USDT5', type: 'fixed', value: 5, used: 412, limit: 1000, start: 'Jun 1', end: 'Jun 15', status: 'active' },
  { code: 'RAMADAN25', type: 'percent', value: 25, used: 9800, limit: 10000, start: 'Mar 1', end: 'Apr 9', status: 'expired' },
  { code: 'CASHBACK3', type: 'cashback', value: 3, used: 2310, limit: 99999, start: 'Jan 1', end: 'Dec 31', status: 'active' },
  { code: 'AGENT15', type: 'percent', value: 15, used: 88, limit: 200, start: 'Jun 1', end: 'Jul 1', status: 'active' },
  { code: 'STEAMDEAL', type: 'fixed', value: 8, used: 540, limit: 540, start: 'May 10', end: 'May 20', status: 'depleted' },
]

export interface DemoReview {
  id: string
  product: string
  art: string
  user: string
  rating: number
  body: string
  date: string
  status: string
}

export const reviews: DemoReview[] = [
  { id: 'R-3301', product: 'PUBG Mobile UC', art: 'pubg', user: 'Yusuf Demir', rating: 5, body: 'Instant delivery as always, UC credited in under 10 seconds. Best top-up store in the region.', date: '12m ago', status: 'pending' },
  { id: 'R-3302', product: 'Netflix Subscription', art: 'netflix', user: 'Sara Nasser', rating: 2, body: 'Code worked but took a while to show up in my vault. Support sorted it quickly though.', date: '40m ago', status: 'pending' },
  { id: 'R-3303', product: 'Steam Wallet (US)', art: 'steam', user: 'Ali Hassan', rating: 5, body: 'Cheaper than buying directly and the reseller pricing is unbeatable. Highly recommend.', date: '1h ago', status: 'pending' },
  { id: 'R-3304', product: 'iTunes Gift Card (US)', art: 'itunes', user: 'Lina Saleh', rating: 1, body: 'This is spam content with a suspicious link, do not click anything in here lol.', date: '2h ago', status: 'pending' },
  { id: 'R-3305', product: 'Mobile Legends Diamonds', art: 'mlbb', user: 'Elif Yıldız', rating: 4, body: 'Good prices, fast credit. Wish there were more denomination options for diamonds.', date: '3h ago', status: 'approved' },
  { id: 'R-3306', product: 'USDT Voucher (TRC20)', art: 'usdt', user: 'Noor Khalid', rating: 5, body: 'Smooth crypto voucher purchase, redeemed without any issues. Will buy again.', date: '5h ago', status: 'approved' },
  { id: 'R-3307', product: 'Bigo Live Diamonds', art: 'bigo', user: 'Hamza Aziz', rating: 3, body: 'Delivery was fine but the player ID field was a little confusing at checkout.', date: 'Yesterday', status: 'rejected' },
]

export const admins = [
  { name: 'Omar Farouk', email: 'omar.f@proton.me', role: 'Super admin', last: 'Online now', status: 'active' },
  { name: 'Layla Haddad', email: 'layla.h@salehcard.co', role: 'Editor', last: '2h ago', status: 'active' },
  { name: 'Mona Khoury', email: 'mona.k@salehcard.co', role: 'Editor', last: 'Yesterday', status: 'active' },
  { name: 'Tarek Sami', email: 'tarek.s@salehcard.co', role: 'Viewer', last: '3d ago', status: 'invited' },
]

export const revSeries: Record<string, number[]> = {
  daily: [4.2, 5.1, 4.8, 6.3, 7.1, 6.6, 8.2, 7.4, 9.1, 8.7, 10.2, 9.6, 11.4, 12.1],
  weekly: [28, 32, 30, 38, 41, 44, 47, 52, 49, 58, 61, 67],
  monthly: [98, 112, 121, 134, 128, 156, 162, 178, 171, 198, 214, 240],
}
export const revLabels: Record<string, string[]> = {
  daily: ['', '', '', '', '', '', '', '', '', '', '', '', '', 'Today'],
  weekly: ['W1', 'W2', 'W3', 'W4', 'W5', 'W6', 'W7', 'W8', 'W9', 'W10', 'W11', 'W12'],
  monthly: ['Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec', 'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun'],
}
export const ffBreakdown = [
  { key: 'code', label: 'Code / PIN', value: 42, color: '#5b8bff' },
  { key: 'credit', label: 'Account credit', value: 46, color: '#2fd47a' },
  { key: 'transfer', label: 'Money transfer', value: 12, color: '#ff9b3d' },
]
export const payBreakdown = [
  { label: 'Store wallet', value: 58, color: '#8a3bff' },
  { label: 'Visa / Mastercard', value: 29, color: '#3b5bff' },
  { label: 'USDT', value: 13, color: '#22e3c8' },
]
export const catRevenue = [
  { label: 'Game Top-ups', value: 38 },
  { label: 'Gift Cards', value: 24 },
  { label: 'App & Live', value: 16 },
  { label: 'Software', value: 11 },
  { label: 'Crypto', value: 7 },
  { label: 'Transfers', value: 4 },
]
