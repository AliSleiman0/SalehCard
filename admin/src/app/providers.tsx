import { useEffect, type ReactNode } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { queryClient } from '@/lib/query-client'
import { useThemeStore } from '@/stores/theme'
import { useLocaleStore } from '@/stores/locale'

export default function Providers({ children }: { children: ReactNode }) {
  useEffect(() => {
    useThemeStore.getState().init()
    // Force English on every load (handoff requirement, same as /web).
    useLocaleStore.getState().setLocale('en')
  }, [])

  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>{children}</BrowserRouter>
    </QueryClientProvider>
  )
}
