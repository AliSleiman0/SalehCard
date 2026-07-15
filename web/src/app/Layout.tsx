import { useEffect } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { Header } from '@/components/layout/Header'
import { Footer } from '@/components/layout/Footer'
import { BottomNav } from '@/components/layout/BottomNav'
import { useUiStore } from '@/stores/ui'

export default function Layout() {
  const { agent, acctDrawer, setAcctDrawer } = useUiStore()
  const { pathname } = useLocation()

  // Auth screens are focused, chrome-free: no header, footer, or bottom nav — the
  // AuthCard carries its own logo and centers in the full viewport.
  const authRoute = pathname === '/login' || pathname === '/register'

  // Close the account drawer whenever the route changes (e.g. tapping a nav link).
  useEffect(() => {
    setAcctDrawer(false)
  }, [pathname, setAcctDrawer])

  // Lock background scroll while the drawer is open.
  useEffect(() => {
    document.body.style.overflow = acctDrawer ? 'hidden' : ''
    return () => {
      document.body.style.overflow = ''
    }
  }, [acctDrawer])

  return (
    <div
      className="app"
      data-mode="desktop"
      data-agent={agent ? '1' : '0'}
      data-acct-drawer={acctDrawer ? '1' : '0'}
    >
      <div className="appscroll">
        {!authRoute && <Header />}
        <main>
          <Outlet />
        </main>
        {!authRoute && <Footer />}
        {!authRoute && <BottomNav />}
      </div>
      <div className="drawer-backdrop" onClick={() => setAcctDrawer(false)} />
    </div>
  )
}
