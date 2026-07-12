import { useEffect, useState } from 'react'

// useCountdown returns mm:ss until `expiresAt` and whether it has elapsed,
// ticking once per second.
export function useCountdown(expiresAt: string | undefined): { label: string; done: boolean } {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const h = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(h)
  }, [])
  if (!expiresAt) return { label: '--:--', done: false }
  const ms = new Date(expiresAt).getTime() - now
  if (ms <= 0) return { label: '00:00', done: true }
  const total = Math.floor(ms / 1000)
  const m = Math.floor(total / 60)
  const s = total % 60
  return { label: `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`, done: false }
}
