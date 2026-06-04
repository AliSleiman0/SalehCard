import { useEffect, type ReactNode } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { ToastProvider } from '@/components'
import { queryClient } from '@/lib/query-client'
import { useThemeStore } from '@/stores/theme'
import { useLocaleStore } from '@/stores/locale'

interface ProvidersProps {
  children: ReactNode
}

export default function Providers({ children }: ProvidersProps) {
  useEffect(() => {
    useThemeStore.getState().init()
    useLocaleStore.getState().setLocale('en')
  }, [])

  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <BrowserRouter>{children}</BrowserRouter>
      </ToastProvider>
    </QueryClientProvider>
  )
}
