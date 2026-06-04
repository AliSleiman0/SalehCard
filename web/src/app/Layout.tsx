import { Outlet } from 'react-router-dom'
import { Header } from '@/components/layout/Header'
import { Footer } from '@/components/layout/Footer'
import { BottomNav } from '@/components/layout/BottomNav'
import { useUiStore } from '@/stores/ui'

export default function Layout() {
  const { agent } = useUiStore()
  return (
    <div className="app" data-mode="desktop" data-agent={agent ? '1' : '0'}>
      <div className="appscroll">
        <Header />
        <main>
          <Outlet />
        </main>
        <Footer />
        <BottomNav />
      </div>
    </div>
  )
}
