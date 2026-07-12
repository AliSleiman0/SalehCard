import { useCallback, useEffect, useRef, useState } from 'react'
import { verifyAccount } from '../api/verify'

export type VerifyStatus = 'idle' | 'checking' | 'found' | 'notFound' | 'unavailable'

export interface VerifyState {
  status: VerifyStatus
  username: string
  // Buy proceeds on a resolved account or a fail-open (upstream unavailable).
  // idle / checking / notFound all block.
  allowsPurchase: boolean
}

const IDLE: VerifyState = { status: 'idle', username: '', allowsPurchase: false }

// useAccountVerification debounces a game-ID → nickname lookup (500ms) with a
// pending-token guard that discards stale responses. Fail-open: a transport
// error or any non-"id_not_found" outcome returns `unavailable` so a third-party
// hiccup never blocks the sale. Port of the app's product-page verify flow.
export function useAccountVerification(productId: string, enabled: boolean) {
  const [state, setState] = useState<VerifyState>(IDLE)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const token = useRef(0)

  const onIdChange = useCallback(
    (raw: string) => {
      if (timer.current) clearTimeout(timer.current)
      const id = raw.trim()
      if (!enabled || !id) {
        token.current++ // invalidate any in-flight lookup
        setState(IDLE)
        return
      }
      setState({ status: 'checking', username: '', allowsPurchase: false })
      const myToken = ++token.current
      timer.current = setTimeout(async () => {
        try {
          const res = await verifyAccount(productId, id)
          if (myToken !== token.current) return // superseded
          const data = res.data
          if (res.success && data?.found) {
            setState({ status: 'found', username: data.username ?? '', allowsPurchase: true })
          } else if (data?.reason === 'id_not_found') {
            setState({ status: 'notFound', username: '', allowsPurchase: false })
          } else {
            // 'unavailable' or any unexpected shape → fail open
            setState({ status: 'unavailable', username: '', allowsPurchase: true })
          }
        } catch {
          if (myToken !== token.current) return
          setState({ status: 'unavailable', username: '', allowsPurchase: true })
        }
      }, 500)
    },
    [productId, enabled],
  )

  useEffect(() => {
    return () => {
      if (timer.current) clearTimeout(timer.current)
    }
  }, [])

  return { ...state, onIdChange }
}
