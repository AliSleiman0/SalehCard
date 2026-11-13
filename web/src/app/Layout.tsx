import { useEffect } from 'react'
import { Outlet, useLocation, useNavigationType } from 'react-router-dom'
import { Header } from '@/components/layout/Header'
import { Footer } from '@/components/layout/Footer'
import { BottomNav } from '@/components/layout/BottomNav'
import { useUiStore } from '@/stores/ui'
import { NotificationPopups } from '@/features/notifications/components/NotificationPopups'
import { ChatWidget } from '@/features/chatbot/components/ChatWidget'

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

  // A forward navigation (link, category switch, search result…) starts at the
  // top of the new page instead of keeping the previous page's scroll offset
  // (client msg #51). Back/forward (POP) is left alone so returning to a list
  // keeps the browser's own position.
  const navType = useNavigationType()
  useEffect(() => {
    if (navType !== 'POP') window.scrollTo(0, 0)
  }, [pathname, navType])

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
      {!authRoute && <NotificationPopups />}
      {!authRoute && <ChatWidget />}
      <div className="drawer-backdrop" onClick={() => setAcctDrawer(false)} />
    </div>
  )
}
