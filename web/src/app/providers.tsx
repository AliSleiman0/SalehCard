import { useEffect, type ReactNode } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { ToastProvider } from '@/components'
import { queryClient } from '@/lib/query-client'
import { useThemeStore } from '@/stores/theme'
import { useLocaleStore } from '@/stores/locale'
import { useAuthStore } from '@/stores/auth'
import { clearSessionHint, hasSessionHint } from '@/lib/sessionHint'
import { refresh } from '@/features/auth/api/auth'
import { applyAuth } from '@/features/auth/hooks/applyAuth'

interface ProvidersProps {
  children: ReactNode
}

export default function Providers({ children }: ProvidersProps) {
  useEffect(() => {
    useThemeStore.getState().init()
    useLocaleStore.getState().setLocale('en')

    // Restore an existing session from the httpOnly refresh cookie — but only
    // if a prior login left a session hint. Anonymous/first-time visitors skip
    // the probe entirely (it would be a guaranteed 401), going straight to
    // hydrated. Marks the auth store hydrated either way so route guards resolve.
    if (!hasSessionHint()) {
      useAuthStore.getState().setHydrated(true)
      return
    }

    refresh()
      .then((res) => {
        if (res.success && res.data) applyAuth(res.data)
        // Hint present but the cookie is gone/expired — drop it so the next
        // cold load stays silent instead of probing again.
        else clearSessionHint()
      })
      .catch(() => {
        /* no/expired session — remain logged out */
        clearSessionHint()
      })
      .finally(() => {
        useAuthStore.getState().setHydrated(true)
      })
  }, [])

  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <BrowserRouter>{children}</BrowserRouter>
      </ToastProvider>
    </QueryClientProvider>
  )
}
