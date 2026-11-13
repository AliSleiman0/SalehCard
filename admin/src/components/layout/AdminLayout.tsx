import { Suspense } from 'react'
import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { Topbar } from './Topbar'
import { LoadingSpinner } from '@/components'
import { Toaster } from '@/components/Toaster'
import { useUiStore } from '@/stores/ui'
import { useNewTopUpAlert } from '@/features/topups/hooks/useNewTopUpAlert'

export function AdminLayout() {
  const { collapsed } = useUiStore()
  useNewTopUpAlert()
  return (
    <div className="admin" data-collapsed={collapsed}>
      <Sidebar />
      <div className="main">
        <Topbar />
        <main className="content">
          <Suspense fallback={<LoadingSpinner />}>
            <Outlet />
          </Suspense>
        </main>
      </div>
      <Toaster />
    </div>
  )
}
