// formatUsdtAmount renders an on-chain USDT amount at full precision: 6 decimals,
// trailing zeros trimmed, but never fewer than 2. The sub-cent digits ARE the
// payment's identity (a salted counter), so this MUST NOT be rounded — never run
// intent amounts through fmtPrice/Price. Port of the app's formatUsdtAmount.
export function formatUsdtAmount(n: number): string {
  let s = n.toFixed(6)
  // trim trailing zeros but keep at least 2 decimals
  s = s.replace(/(\.\d{2}\d*?)0+$/, '$1')
  return s
}
