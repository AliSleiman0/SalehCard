export interface OrderView {
  id: string
  product: string
  art: string
  total: number
  method: string
  fulfill: 'code' | 'credit' | 'transfer'
  status?: string
  date?: string
  code?: string
  pin?: string
  account?: string
  amount?: string
  ts?: string
  ref?: string
  recipient?: { name: string; country: string; detail?: string }
  steps?: { k: string; ts: string }[]
}
