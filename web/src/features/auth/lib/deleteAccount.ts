// canForceDelete reports whether a delete-account refusal may be overridden
// with an explicit "Delete anyway" (`force: true`). Only a leftover wallet
// balance or orders still processing qualify — the server forfeits the balance
// and leaves the orders for the operator. Pending payments / top-ups (and every
// other error) always block.
export function canForceDelete(code: string | undefined | null): boolean {
  return code === 'WALLET_NOT_EMPTY' || code === 'ORDERS_IN_FLIGHT'
}

// forfeitBalance returns the balance worth quoting in the forfeit warning
// (>= one cent, matching the server's "empty" threshold), else null.
export function forfeitBalance(balance: number | undefined | null): number | null {
  return typeof balance === 'number' && Number.isFinite(balance) && balance >= 0.01 ? balance : null
}
