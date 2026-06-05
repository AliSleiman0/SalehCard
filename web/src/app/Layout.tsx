import { useEffect } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { Header } from '@/components/layout/Header'
import { Footer } from '@/components/layout/Footer'
import { BottomNav } from '@/components/layout/BottomNav'
import { useUiStore } from '@/stores/ui'

export default function Layout() {
  const { agent, acctDrawer, setAcctDrawer } = useUiStore()
  const { pathname } = useLocation()

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
        <Header />
        <main>
          <Outlet />
        </main>
        <Footer />
        <BottomNav />
      </div>
      <div className="drawer-backdrop" onClick={() => setAcctDrawer(false)} />
    </div>
  )
}
