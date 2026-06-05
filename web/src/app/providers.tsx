import { useEffect, type ReactNode } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { ToastProvider } from '@/components'
import { queryClient } from '@/lib/query-client'
import { useThemeStore } from '@/stores/theme'
import { useLocaleStore } from '@/stores/locale'
import { useAuthStore } from '@/stores/auth'
import { refresh } from '@/features/auth/api/auth'
import { applyAuth } from '@/features/auth/hooks/applyAuth'

interface ProvidersProps {
  children: ReactNode
}

export default function Providers({ children }: ProvidersProps) {
  useEffect(() => {
    useThemeStore.getState().init()
    useLocaleStore.getState().setLocale('en')

    // Restore an existing session from the httpOnly refresh cookie. Marks the
    // auth store hydrated either way so route guards can resolve.
    refresh()
      .then((res) => {
        if (res.success && res.data) applyAuth(res.data)
      })
      .catch(() => {
        /* no/expired session — remain logged out */
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
