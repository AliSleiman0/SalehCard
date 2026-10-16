export interface CodeLine {
  title: string
  codes: string[]
}

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
  /** Delivered codes grouped per line (one code per unit). Non-empty even on a
   *  processing order when its code lines were delivered alongside manual lines. */
  codeLines: CodeLine[]
  /** Kind of the first manual (non-code) line, when the order has one. */
  manualKind?: 'credit' | 'transfer'
  pin?: string
  account?: string
  amount?: string
  ts?: string
  ref?: string
  recipient?: { name: string; country: string; detail?: string }
  steps?: { k: string; ts: string }[]
}
